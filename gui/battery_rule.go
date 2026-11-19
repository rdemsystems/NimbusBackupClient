package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"clientcommon"
	"pbscommon"
)

// A Backup Set's "On battery" rule. On a laptop running on its battery a
// scheduled or startup run of the set starts anyway, starts only when the
// charge is at least a level, or waits for mains power; and a running backup
// of the set (Run now too) can be cancelled when the charge drops below a
// level. On mains, and on a computer with no battery, nothing changes.
//
// A run that may not start is held back, not lost: the set's stored next run
// is left where it was, so the scheduler's catch-up finds it due on every
// tick and starts it, once, when the computer is plugged in. The hold is
// recorded once (a Reports entry, a Message Log line, a [Battery] line in the
// service log), and remembered across a restart of the service; the run that
// later starts says it ran late, from when it was first held. When the computer
// was woken by the set's wake timer (wake_timer.go) for a run that is then
// held, it is put straight back to sleep, with the wake for the next run set.

// readBattery reads the computer's battery. A variable so the tests can
// stand in for the system.
var readBattery = clientcommon.ReadBattery

// sleepComputer puts the computer to sleep (pbscommon's suspend_*.go); tests
// replace it.
var sleepComputer = pbscommon.SleepComputer

// batteryCheckEvery is how often a running backup's cancel level is checked.
var batteryCheckEvery = 30 * time.Second

// sleepAfterHoldDelay gives the Reports entry and the logs a moment to be
// written before the computer goes back to sleep.
var sleepAfterHoldDelay = 5 * time.Second

const (
	// wokenGap: the scheduler ticks every minute while the computer is awake,
	// so a tick this long after the one before means it slept in between.
	wokenGap = 3 * time.Minute
	// wokenFresh: how long after waking a run of the set the wake was for
	// counts as woken for (the run is due wakeLead after the wake).
	wokenFresh = 15 * time.Minute
	// wokenLate: the first tick after the gap must come this soon after the
	// wake timer was due. Later than that, the timer did not wake the
	// computer (Modern Standby ignored it, or it was hibernated or off) and
	// someone woke it by hand: it must never be put back to sleep under them.
	wokenLate = 5 * time.Minute
	// wakeEarly: a wake this long before the wake timer was due still counts
	// as the timer's (noteSchedulerTick).
	wakeEarly = time.Minute
	// shortSleepGap: a tick this long after the one before (a minute
	// apart while awake) may follow a short sleep; the system's record of a
	// wake tells (noteSchedulerTick).
	shortSleepGap = 75 * time.Second
)

// batteryRuleAsSaved is the set's rule as stored, for checking it.
func (j ScheduledJob) batteryRuleAsSaved() clientcommon.BatteryRule {
	r := clientcommon.BatteryRule{Mode: j.OnBattery, StartPercent: j.BatteryStartPercent}
	if j.BatteryCancel {
		r.CancelBelow = j.BatteryCancelBelow
	}
	return r
}

// batteryRule is the set's rule to act on.
func (j ScheduledJob) batteryRule() clientcommon.BatteryRule {
	return j.batteryRuleAsSaved().Normal()
}

// BatteryInfo is the computer's battery, for the Schedule tab: a computer
// with none shows the On battery section greyed out.
type BatteryInfo struct {
	HasBattery bool `json:"hasBattery"`
	OnBattery  bool `json:"onBattery"`
	Percent    int  `json:"percent"`
	Known      bool `json:"known"`
}

// GetBatteryInfo reads this computer's battery (the app and its service are
// on the same computer, so the app reads it itself).
func (a *App) GetBatteryInfo() BatteryInfo {
	b := readBattery()
	return BatteryInfo{HasBattery: b.HasBattery, OnBattery: b.OnBattery, Percent: b.Percent, Known: b.Known}
}

func onBattery(b clientcommon.BatteryStatus) bool { return b.HasBattery && b.OnBattery }

// pctParam is the charge for a message: the number, or "?" when unknown.
func pctParam(b clientcommon.BatteryStatus) interface{} {
	if b.Known {
		return b.Percent
	}
	return "?"
}

func pctText(p interface{}) string { return fmt.Sprint(p) }

// clock is a time as the English messages give it; the window shows the
// RFC 3339 form of the same time (the message's parameter) in its language.
func clock(t time.Time) string { return t.Format("15:04") }

// ---- held runs

// batteryHold is a run of a set held back for the battery. The holds are
// kept in battery_holds.json as well as in memory: a service restarted while
// a run was held (an upgrade, say) recorded the hold a second time in
// Reports, and the run that later started said it was held back from the
// restart, not from when it really was.
type batteryHold struct {
	At      time.Time `json:"at"`
	Pct     *int      `json:"pct,omitempty"` // nil when the charge could not be read
	Trigger string    `json:"trigger"`       // "scheduled" or "startup"
}

func newBatteryHold(at time.Time, b clientcommon.BatteryStatus, trigger string) batteryHold {
	h := batteryHold{At: at, Trigger: trigger}
	if b.Known {
		p := b.Percent
		h.Pct = &p
	}
	return h
}

// pct is the charge when the run was held, for a message: the number, or
// "?" when unknown (as pctParam).
func (h batteryHold) pct() interface{} {
	if h.Pct == nil {
		return "?"
	}
	return *h.Pct
}

// batteryHolds is the held runs by set ID. loaded is false until the file
// has been read, which is done the first time a hold is looked at.
var batteryHolds = struct {
	sync.Mutex
	m      map[string]batteryHold
	loaded bool
}{m: map[string]batteryHold{}}

func batteryHoldsPath() (string, error) {
	dir, err := getConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "battery_holds.json"), nil
}

// loadBatteryHoldsLocked reads the holds an earlier run of this program left,
// once. A file that cannot be read loses only the memory of the holds: a run
// still held is recorded once more, as before they were kept. Called with
// batteryHolds locked.
func loadBatteryHoldsLocked() {
	if batteryHolds.loaded {
		return
	}
	batteryHolds.loaded = true
	path, err := batteryHoldsPath()
	if err != nil {
		writeDebugLog(fmt.Sprintf("[Battery] the held runs could not be read: %v", err))
		return
	}
	data, err := os.ReadFile(path) // #nosec G703 -- the settings folder's file with a fixed name: not user input
	if err != nil {
		if !os.IsNotExist(err) {
			writeDebugLog(fmt.Sprintf("[Battery] the held runs could not be read: %v", err))
		}
		return
	}
	saved := map[string]batteryHold{}
	if err := json.Unmarshal(stripUTF8BOM(data), &saved); err != nil {
		writeDebugLog(fmt.Sprintf("[Battery] the held runs could not be read: %v", err))
		return
	}
	for id, h := range saved {
		if _, ok := batteryHolds.m[id]; !ok && !h.At.IsZero() {
			batteryHolds.m[id] = h
		}
	}
}

// saveBatteryHoldsLocked writes the holds, removing the file when there are
// none. Called with batteryHolds locked.
func saveBatteryHoldsLocked() {
	path, err := batteryHoldsPath()
	if err != nil {
		writeDebugLog(fmt.Sprintf("[Battery] the held runs could not be saved: %v", err))
		return
	}
	if len(batteryHolds.m) == 0 {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) { // #nosec G703 -- the settings folder's file with a fixed name: not user input
			writeDebugLog(fmt.Sprintf("[Battery] the held runs could not be saved: %v", err))
		}
		return
	}
	data, err := json.MarshalIndent(batteryHolds.m, "", "  ")
	if err == nil {
		err = atomicWriteFile(path, data, 0600)
	}
	if err != nil {
		writeDebugLog(fmt.Sprintf("[Battery] the held runs could not be saved: %v", err))
	}
}

// batteryHeld reports whether a run of the set is held back, and by which
// trigger.
func batteryHeld(setID string) (trigger string, held bool) {
	batteryHolds.Lock()
	defer batteryHolds.Unlock()
	loadBatteryHoldsLocked()
	h, ok := batteryHolds.m[setID]
	return h.Trigger, ok
}

func takeBatteryHold(setID string) (batteryHold, bool) {
	batteryHolds.Lock()
	defer batteryHolds.Unlock()
	loadBatteryHoldsLocked()
	h, ok := batteryHolds.m[setID]
	if ok {
		delete(batteryHolds.m, setID)
		saveBatteryHoldsLocked()
	}
	return h, ok
}

// dropStaleBatteryHolds forgets holds no run will take: of a set deleted or
// disabled, or of a scheduled run that is no longer due (its schedule was
// changed, or Run now ran it).
func dropStaleBatteryHolds(jobs []ScheduledJob, now time.Time) {
	byID := map[string]ScheduledJob{}
	for _, j := range jobs {
		byID[j.ID] = j
	}
	batteryHolds.Lock()
	defer batteryHolds.Unlock()
	loadBatteryHoldsLocked()
	dropped := false
	for id, h := range batteryHolds.m {
		j, ok := byID[id]
		stale := !ok || !j.Enabled
		if ok && h.Trigger == "scheduled" {
			next, err := time.Parse(time.RFC3339, j.NextRun)
			stale = stale || err != nil || next.After(now)
		}
		if stale {
			delete(batteryHolds.m, id)
			dropped = true
			writeDebugLog(fmt.Sprintf("[Battery] %s: the run held back at %s is no longer due", j.Name, clock(h.At)))
		}
	}
	if dropped {
		saveBatteryHoldsLocked()
	}
}

// ---- woken for a run

// wokenRun is the last time the computer was found woken by this app's wake
// timer, and the set it was woken for. The first run of that set to look,
// within wokenFresh, takes it.
var wokenRun struct {
	sync.Mutex
	setID string
	at    time.Time
	used  bool
}

// noteSchedulerTick is called on every scheduler tick with the tick before:
// a gap of wokenGap or more means the computer slept (a shorter one, when the
// system recorded a wake within it), and when the wake
// timer armed then was due within the gap, it woke the computer for its set.
// Wall clock times are compared (Round(0)): a monotonic clock may not count
// the time asleep.
func noteSchedulerTick(prev, now time.Time) {
	if prev.IsZero() {
		return
	}
	p, n := prev.Round(0), now.Round(0)
	// A gap shorter than wokenGap is a sleep only if the system recorded a
	// wake within it: found on a laptop 2026-11-19, hibernated 2.5 minutes
	// before its wake timer, the wake was not seen, so the run had no hold
	// and no wait for its server and started before the Wi-Fi was back. A
	// tick late by less than shortSleepGap is just late.
	var short *wakeVerdict
	if gap := n.Sub(p); gap < wokenGap {
		if gap < shortSleepGap {
			return
		}
		v := whatWokeTheComputer(p)
		if !v.known {
			return
		}
		short = &v
	}
	// However it was woken, runs starting now first wait for the network
	// to come back (network_wait.go).
	noteResume(n, "after the computer woke")
	wakeState.mu.Lock()
	t, armed := wakeState.target, wakeState.armed
	wakeState.mu.Unlock()
	// The first tick after a resume comes the moment the computer wakes,
	// when the timer has only just fired: the computer may wake a moment
	// before it was due, or read its clock before it is set again from the
	// hardware clock. Found on a laptop: a wake at the timer's second was
	// missed, as the tick read just before it, and the ticks after it were
	// too close together to show a wake (#142).
	if !armed || !t.At.After(p) || t.At.After(n.Add(wakeEarly)) {
		if armed {
			writeDebugLog(fmt.Sprintf("[Wake] the computer woke at about %s; the wake timer is for %q at %s: not woken by it", n.Format(time.RFC3339), t.Set, t.At.Format(time.RFC3339)))
		}
		return
	}
	if n.Sub(t.At) > wokenLate {
		writeDebugLog(fmt.Sprintf("[Wake] the computer woke at about %s, %s after the wake timer for %q was due: not woken by the timer", n.Format(time.RFC3339), n.Sub(t.At).Round(time.Second), t.Set))
		return
	}
	// In the timer's window, but the system may say the power button woke it
	// (wake_source.go).
	v := short
	if v == nil {
		w := whatWokeTheComputer(p)
		v = &w
	}
	if v.known && !v.ours {
		writeDebugLog(fmt.Sprintf("[Wake] the computer woke at about %s, around the wake timer for %q, but Windows says %s woke it: not woken by the timer", n.Format(time.RFC3339), t.Set, v.what))
		return
	}
	wokenRun.Lock()
	wokenRun.setID, wokenRun.at, wokenRun.used = t.SetID, n, false
	wokenRun.Unlock()
	writeDebugLog(fmt.Sprintf("[Wake] the computer was woken at about %s for %q (run at %s)", n.Format(time.RFC3339), t.Set, t.Run.Format(time.RFC3339)))
	// Left alone, a computer woken by a timer sleeps again within minutes,
	// before the run has started (wake_hold.go).
	holdAwakeAfterWake(t.SetID, t.Set)
}

// takeWoken returns when the computer was woken for this scheduled run of
// job (zero when it was not), once.
func takeWoken(job ScheduledJob, trigger string, now time.Time) time.Time {
	if trigger != "scheduled" {
		return time.Time{}
	}
	wokenRun.Lock()
	defer wokenRun.Unlock()
	if wokenRun.used || wokenRun.setID != job.ID || now.Round(0).Sub(wokenRun.at) > wokenFresh {
		return time.Time{}
	}
	wokenRun.used = true
	return wokenRun.at
}

// ---- starting a run

// batteryRunInfo is what the battery did around one run of a set, for its
// Reports entry.
type batteryRunInfo struct {
	start     clientcommon.BatteryStatus
	startedAt time.Time
	wokenAt   time.Time // zero unless the computer was woken for this run
	heldAt    time.Time // zero unless the run was held back and ran late
	heldPct   interface{}

	mu        sync.Mutex
	cancelled bool
	cancelAt  time.Time
	cancelPct interface{}
	cancelMin int
}

// batteryGate decides, as a run of job is about to start, whether the
// battery lets it. With ok false the run does not start now: it is held back
// (recorded the first time) and starts on a later tick. trigger is
// "scheduled", "startup" or "manual"; Run now always starts.
func (a *App) batteryGate(job ScheduledJob, trigger string, now time.Time) (info *batteryRunInfo, ok bool) {
	b := readBattery()
	info = &batteryRunInfo{start: b, startedAt: now, wokenAt: takeWoken(job, trigger, now)}
	if trigger == "scheduled" || trigger == "startup" {
		if hold, min := job.batteryRule().Holds(b); hold {
			a.holdForBattery(job, trigger, now, b, min, info.wokenAt)
			return nil, false
		}
	}
	if h, held := takeBatteryHold(job.ID); held {
		if trigger != "manual" && !onBattery(b) {
			info.heldAt, info.heldPct = h.At, h.pct()
			writeDebugLog(fmt.Sprintf("[Battery] %s: computer plugged in (battery %s%%): starting the run held back at %s", job.Name, pctText(pctParam(b)), clock(h.At)))
			LogMessage("Backup", "info",
				fmt.Sprintf("%s: computer plugged in (battery %s%%); starting the run held back at %s.", job.Name, pctText(pctParam(b)), clock(h.At)),
				job.Name, MsgLogBatteryPluggedIn, msgParams{"pct": pctParam(b), "time": h.At.Format(time.RFC3339)})
		} else {
			writeDebugLog(fmt.Sprintf("[Battery] %s: the run held back at %s is replaced by this %s run", job.Name, clock(h.At), trigger))
		}
	}
	if onBattery(b) {
		if !info.wokenAt.IsZero() {
			writeDebugLog(fmt.Sprintf("[Battery] %s: on battery at %s%%, computer woken for it at %s: run starts", job.Name, pctText(pctParam(b)), clock(info.wokenAt)))
			LogMessage("Backup", "info",
				fmt.Sprintf("%s: started on battery at %s%% (computer woken for it at %s).", job.Name, pctText(pctParam(b)), clock(info.wokenAt)),
				job.Name, MsgLogBatteryStartedWoken, msgParams{"pct": pctParam(b), "time": info.wokenAt.Format(time.RFC3339)})
		} else {
			writeDebugLog(fmt.Sprintf("[Battery] %s: on battery at %s%%: run starts", job.Name, pctText(pctParam(b))))
			LogMessage("Backup", "info",
				fmt.Sprintf("%s: started on battery at %s%%.", job.Name, pctText(pctParam(b))),
				job.Name, MsgLogBatteryStarted, msgParams{"pct": pctParam(b)})
		}
	}
	return info, true
}

// otherBackupsActive reports whether a backup or restore other than a run of
// the set selfID is running or waiting its turn in this process.
func otherBackupsActive(selfID string) bool {
	runningJobsMutex.Lock()
	others := 0
	for id := range runningJobs {
		if id != selfID {
			others++
		}
	}
	runningJobsMutex.Unlock()
	return others > 0 || runningOperationCount() > 0 || localBackupRunning()
}

// holdForBattery holds a run of job back for the battery: a Reports entry
// ("held", or "skipped" when the computer was woken for it and goes straight
// back to sleep), a Message Log line and a service log line, the first time
// only. min is the start level, 0 when the set waits for mains power.
func (a *App) holdForBattery(job ScheduledJob, trigger string, now time.Time, b clientcommon.BatteryStatus, min int, woken time.Time) {
	pct := pctParam(b)
	batteryHolds.Lock()
	loadBatteryHoldsLocked()
	_, already := batteryHolds.m[job.ID]
	if !already {
		batteryHolds.m[job.ID] = newBatteryHold(now, b, trigger)
		saveBatteryHoldsLocked()
	}
	batteryHolds.Unlock()

	sleep := !woken.IsZero()
	if sleep && otherBackupsActive(job.ID) {
		sleep = false
		writeDebugLog(fmt.Sprintf("[Battery] %s: the computer was woken for this run, but is not put back to sleep: another backup is running or waiting", job.Name))
	}
	if already {
		// Already recorded; woken again for the set's next run while still
		// held, it goes back to sleep without a further entry.
		if sleep {
			writeDebugLog(fmt.Sprintf("[Battery] %s: woken again, still on battery (%s%%): the run stays held back", job.Name, pctText(pct)))
			a.sleepForHeldRun(job)
		}
		return
	}

	entry := JobHistory{
		ID:         fmt.Sprintf("%d", now.Unix()),
		Name:       job.Name,
		JobID:      job.ID,
		Timestamp:  now.Format(time.RFC3339),
		Status:     "held",
		BackupDirs: job.BackupDirs,
		BackupID:   job.BackupID,
		UseVSS:     job.UseVSS,
		BackupType: job.BackupType,
		Trigger:    trigger,
	}
	log := &MessageLogEntry{Level: "info"}
	switch {
	case sleep && min > 0:
		entry.Status = "skipped"
		entry.Message = fmt.Sprintf("The computer was woken for this backup at %s, but the battery was at %s%%, below the required %d%%. The computer was put straight back to sleep. The backup will start when the computer is plugged in.", clock(woken), pctText(pct), min)
		entry.MessageKey, entry.MessageParams = MsgBatterySkippedWoken, msgParams{"time": woken.Format(time.RFC3339), "pct": pct, "min": min}
		log.Level = "warning"
		log.Message = fmt.Sprintf("computer woken for this backup, battery %s%%, below the required %d%%. Putting the computer back to sleep.", pctText(pct), min)
		log.MessageKey, log.MessageParams = MsgLogBatteryWokenLow, msgParams{"pct": pct, "min": min}
	case sleep:
		entry.Status = "skipped"
		entry.Message = fmt.Sprintf("The computer was woken for this backup at %s, but it was on battery, and this set waits for the computer to be plugged in. The computer was put straight back to sleep. The backup will start when the computer is plugged in.", clock(woken))
		entry.MessageKey, entry.MessageParams = MsgBatterySkippedWokenPlugged, msgParams{"time": woken.Format(time.RFC3339)}
		log.Level = "warning"
		log.Message = "computer woken for this backup, on battery, and this set waits for the computer to be plugged in. Putting the computer back to sleep."
		log.MessageKey = MsgLogBatteryWokenPlugged
	case min > 0:
		entry.Message = fmt.Sprintf("At %s the computer was on battery at %s%%, below the required %d%%. The backup will start when the computer is plugged in.", clock(now), pctText(pct), min)
		entry.MessageKey, entry.MessageParams = MsgBatteryHeld, msgParams{"time": now.Format(time.RFC3339), "pct": pct, "min": min}
		log.Message = fmt.Sprintf("held back, on battery at %s%%, below the required %d%%; waiting for the computer to be plugged in.", pctText(pct), min)
		log.MessageKey, log.MessageParams = MsgLogBatteryHeld, msgParams{"pct": pct, "min": min}
	default:
		entry.Message = fmt.Sprintf("At %s the computer was on battery, and this set waits for the computer to be plugged in.", clock(now))
		entry.MessageKey, entry.MessageParams = MsgBatteryHeldPlugged, msgParams{"time": now.Format(time.RFC3339)}
		log.Message = "held back, on battery; waiting for the computer to be plugged in."
		log.MessageKey = MsgLogBatteryHeldPlugged
	}
	entry.logLine = log
	if min > 0 {
		writeDebugLog(fmt.Sprintf("[Battery] %s: on battery at %s%%, start level %d%%: run held back", job.Name, pctText(pct), min))
	} else {
		writeDebugLog(fmt.Sprintf("[Battery] %s: on battery at %s%%, the set waits for mains power: run held back", job.Name, pctText(pct)))
	}
	if err := a.AddJobHistory(entry); err != nil {
		writeDebugLog(fmt.Sprintf("Warning: Failed to add job history: %v", err))
	}
	if sleep {
		a.sleepForHeldRun(job)
	}
}

// sleepForHeldRun puts the computer, woken for a run of job that is held
// back, straight back to sleep: the wake timer is set for the next run
// first, and nothing is done if another backup has started meanwhile.
func (a *App) sleepForHeldRun(job ScheduledJob) {
	a.refreshWakeTimer()
	go func() {
		time.Sleep(sleepAfterHoldDelay)
		if otherBackupsActive(job.ID) {
			writeDebugLog(fmt.Sprintf("[Battery] %s: not put back to sleep: another backup is running or waiting", job.Name))
			return
		}
		writeDebugLog(fmt.Sprintf("[Battery] %s: putting the computer back to sleep (the run is held back for the battery)", job.Name))
		syncWakeAlarm()
		if err := sleepComputer(); err != nil {
			writeDebugLog(fmt.Sprintf("[Battery] could not put the computer back to sleep: %v", err))
		}
	}()
}

// ---- cancelling a running backup

// watchBatteryCancel checks the battery every batteryCheckEvery while a run
// of job is under way and cancels it below the set's cancel level. Call the
// returned stop when the run has ended.
func (a *App) watchBatteryCancel(job ScheduledJob, info *batteryRunInfo) (stop func()) {
	rule := job.batteryRule()
	if rule.CancelBelow <= 0 || info == nil {
		return func() {}
	}
	done := make(chan struct{})
	go func() {
		t := time.NewTicker(batteryCheckEvery)
		defer t.Stop()
		for {
			select {
			case <-done:
				return
			case <-t.C:
			}
			if a.checkBatteryCancel(job, rule, info, time.Now()) {
				return
			}
		}
	}()
	var once sync.Once
	return func() { once.Do(func() { close(done) }) }
}

// setRun is the running backup of the set named name started since since
// (set names are unique, and a set runs once at a time), or nil.
func setRun(name string, since time.Time) *backupRun {
	for _, r := range activeBackupRuns() {
		if r.name == name && !r.started.Before(since) {
			return r
		}
	}
	return nil
}

// checkBatteryCancel cancels job's running backup through its Stop (it ends
// at a safe point and commits no snapshot) when the computer is on battery
// below the set's cancel level; true when it did. A run still waiting its
// turn reads nothing and is left to start.
func (a *App) checkBatteryCancel(job ScheduledJob, rule clientcommon.BatteryRule, info *batteryRunInfo, now time.Time) bool {
	b := readBattery()
	if !rule.Cancels(b) {
		return false
	}
	r := setRun(job.Name, info.startedAt)
	if r == nil || r.queued.Load() {
		return false
	}
	info.mu.Lock()
	info.cancelled, info.cancelAt, info.cancelPct, info.cancelMin = true, now, pctParam(b), rule.CancelBelow
	info.mu.Unlock()
	msg := fmt.Sprintf("[Battery] %s: battery at %d%%, below the cancel level of %d%%: cancelling the backup (nothing from this run is saved)", job.Name, b.Percent, rule.CancelBelow)
	writeDebugLog(msg)
	runLog(r.logCtx(), msg)
	r.cancel()
	return true
}

// ---- the run's Reports entry

// isBatteryCancelKey reports whether key is a battery cancel's message.
func isBatteryCancelKey(key MessageKey) bool {
	return key == MsgBatteryCancelled || key == MsgBatteryCancelledWoken || key == MsgBatteryCancelledMains
}

// lastRunBatteryCancelled reports whether the set's last run before this one
// was cancelled for the battery (held and skipped entries are not runs).
func (a *App) lastRunBatteryCancelled(setID string) bool {
	if setID == "" {
		return false
	}
	hist, err := a.GetJobHistory()
	if err != nil {
		return false
	}
	for _, h := range hist { // newest first
		if h.JobID != setID || h.Kind == "restore" || h.Status == "running" || h.Status == "held" || h.Status == "skipped" {
			continue
		}
		return h.Status == "cancelled" && isBatteryCancelKey(h.MessageKey)
	}
	return false
}

// applyBatteryToHistory adds to a finished run's Reports entry what the
// battery did: a battery cancel in its own words (with the note after two in
// a row, which only informs: nothing changes by itself), the hold of a run
// that ran late, and the charge at the start and end of a successful run on
// battery. info nil (a one-off backup, an older caller) adds nothing.
func (a *App) applyBatteryToHistory(entry *JobHistory, info *batteryRunInfo) {
	if info == nil {
		return
	}
	end := readBattery()
	if !info.heldAt.IsZero() {
		entry.Notes = append(entry.Notes, HistoryNote{
			Key:    NoteBatteryRanLate,
			Params: msgParams{"time": info.heldAt.Format(time.RFC3339), "pct": info.heldPct, "start": info.startedAt.Format(time.RFC3339)},
			Text:   fmt.Sprintf("Ran late: held back at %s (battery %s%%), started at %s when the computer was plugged in.", clock(info.heldAt), pctText(info.heldPct), clock(info.startedAt)),
		})
	}
	info.mu.Lock()
	cancelled, cancelAt, cancelPct, cancelMin := info.cancelled, info.cancelAt, info.cancelPct, info.cancelMin
	info.mu.Unlock()

	if cancelled && entry.Status != "success" {
		twice := a.lastRunBatteryCancelled(entry.JobID)
		entry.Status = "cancelled"
		startPct := pctParam(info.start)
		params := msgParams{"start": info.startedAt.Format(time.RFC3339), "startPct": startPct, "end": cancelAt.Format(time.RFC3339), "min": cancelMin}
		ending := fmt.Sprintf("but it was automatically cancelled at %s when the battery charge dropped below your setting of %d%%, before it could complete. Nothing from this run was saved; the next run starts it again in full.", clock(cancelAt), cancelMin)
		switch {
		case !info.wokenAt.IsZero():
			entry.MessageKey = MsgBatteryCancelledWoken
			entry.Message = fmt.Sprintf("The computer was woken for this backup at %s and the backup started (battery %s%%), %s", clock(info.startedAt), pctText(startPct), ending)
		case onBattery(info.start):
			entry.MessageKey = MsgBatteryCancelled
			entry.Message = fmt.Sprintf("The backup started at %s on battery (%s%%), %s", clock(info.startedAt), pctText(startPct), ending)
		default:
			entry.MessageKey = MsgBatteryCancelledMains
			entry.Message = fmt.Sprintf("The backup started at %s on mains power, %s", clock(info.startedAt), ending)
		}
		entry.MessageParams = params
		log := &MessageLogEntry{Level: "error", MessageKey: MsgLogBatteryCancelled, MessageParams: msgParams{"min": cancelMin, "pct": cancelPct},
			Message: fmt.Sprintf("cancelled, battery charge dropped below your setting of %d%% (now %s%%). Nothing from this run was saved.", cancelMin, pctText(cancelPct))}
		if twice {
			const twiceText = "This backup has not completed on battery twice in a row. Plug the computer in to let it finish."
			entry.Notes = append(entry.Notes, HistoryNote{Key: NoteBatteryTwice, Text: twiceText})
			log.MessageKey = MsgLogBatteryCancelledTwice
			log.Message += " " + twiceText
		}
		entry.logLine = log
		writeDebugLog(fmt.Sprintf("[Battery] %s: run cancelled for the battery at %s (started at %s, battery %s%%)", entry.Name, clock(cancelAt), clock(info.startedAt), pctText(startPct)))
		return
	}
	if entry.Status == "success" && (onBattery(info.start) || onBattery(end)) {
		entry.Notes = append(entry.Notes, HistoryNote{
			Key:    NoteBatteryStartEnd,
			Params: msgParams{"startPct": pctParam(info.start), "endPct": pctParam(end)},
			Text:   fmt.Sprintf("On battery: %s%% at start, %s%% at the end.", pctText(pctParam(info.start)), pctText(pctParam(end))),
		})
	}
}

// HistoryNote is one further line of a Reports entry: Key and Params for the
// window to show it in the user's language, Text the same in English.
type HistoryNote struct {
	Key    MessageKey `json:"key"`
	Params msgParams  `json:"params,omitempty"`
	Text   string     `json:"text"`
}

// batteryInfo is the battery record of the run under the entry (nil for a
// one-off backup).
func (e *pendingPostActionsEntry) batteryInfo() *batteryRunInfo {
	if e == nil {
		return nil
	}
	return e.battery
}

// setRunBattery records info on the run registered under key.
func (a *App) setRunBattery(key string, info *batteryRunInfo) {
	if key == "" || info == nil {
		return
	}
	if v, ok := a.pendingPostActions.Load(key); ok {
		v.(*pendingPostActionsEntry).battery = info
	}
}
