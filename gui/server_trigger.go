package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
)

// A Backup Set's "When to run" triggers besides its schedules (#157):
//
//   - "When the server can be reached" (ServerTrigger): the set runs when its
//     PBS server answers, at most every so many hours counted from its last
//     backup that worked, optionally only on some networks, kinds of
//     connection, or not on metered connections. A laptop that is away for a
//     week backs up when it is next home; one that stays home still backs up
//     every EveryHours; Wi-Fi that drops and comes back starts nothing extra.
//   - A schedule's "If the server cannot be reached at the time"
//     (Schedule.IfUnreachable "wait"): the run waits, and starts as soon as
//     the server can be reached, for up to WaitHours from its time; only then
//     is it recorded as failed. While it waits the set shows "Waiting for the
//     server" (ScheduledJob.WaitingForServer) and no failure e-mail is sent.
//     A schedule's networks and not metered choice: a run on another network
//     waits for one of them with "wait", and is skipped without it; a
//     computer woken for a run that is skipped, or waits, for its network
//     goes back to sleep, as for the battery (battery_rule.go).
//
// Both only start runs a scheduled run would: the set's On battery rule and
// its other options apply, and a set runs once at a time (runningJobs).

// triggerServer is the history Trigger of a run the server trigger started.
const triggerServer = "server"

// ServerTrigger is a set's "When the server can be reached" trigger.
type ServerTrigger struct {
	Enabled bool `json:"enabled"`
	// EveryHours is the least time from the last backup that worked to the
	// next run (0 means defaultServerEveryHours).
	EveryHours int `json:"everyHours,omitempty"`
	// Networks limits the runs to these networks (names, as netinfo.go reads
	// them); Connections to these kinds ("wired", "wifi", "vpn", "mobile").
	// Empty: any.
	Networks    []string `json:"networks,omitempty"`
	Connections []string `json:"connections,omitempty"`
	NotMetered  bool     `json:"notMetered,omitempty"`
}

const (
	defaultServerEveryHours = 24
	maxServerEveryHours     = 24 * 31
)

// every is the least time between the trigger's runs.
func (t *ServerTrigger) every() time.Duration {
	h := t.EveryHours
	if h <= 0 {
		h = defaultServerEveryHours
	}
	return time.Duration(min(h, maxServerEveryHours)) * time.Hour
}

// validateServerTrigger checks a set's server trigger before it is saved.
func validateServerTrigger(t *ServerTrigger) error {
	if t == nil {
		return nil
	}
	if t.EveryHours < 0 || t.EveryHours > maxServerEveryHours {
		return fmt.Errorf("when the server can be reached: at most every 1 to %d hours", maxServerEveryHours)
	}
	for _, c := range t.Connections {
		if !connectionKinds[c] {
			return fmt.Errorf("when the server can be reached: unknown connection %q", c)
		}
	}
	return nil
}

// serverTriggerOn reports whether job's server trigger may start runs.
func serverTriggerOn(job ScheduledJob) bool {
	return job.Enabled && job.ScheduleMode != ScheduleModeManual && job.ServerTrigger != nil && job.ServerTrigger.Enabled
}

var (
	// serverDialEvery: the server of a set whose trigger is due is tried at
	// most this often while it does not answer.
	serverDialEvery = 2 * time.Minute
	// serverRetryAfterRun: after the trigger started a run, it starts no
	// other for this long, so a run that fails (a login refused, a disk
	// full) is not started again on every tick while the server answers.
	serverRetryAfterRun = time.Hour
)

// serverTries is when each set's server was last tried, and when the
// trigger last started a run of it.
var serverTries = struct {
	sync.Mutex
	dialed  map[string]time.Time
	started map[string]time.Time
}{dialed: map[string]time.Time{}, started: map[string]time.Time{}}

// lastGoodRun is when the set's last backup that worked ended, from the
// run history (newest first); zero when none has.
func lastGoodRun(history []JobHistory, setID string) time.Time {
	for _, h := range history {
		if h.JobID != setID || h.Kind == "restore" || h.Status != "success" {
			continue
		}
		if t, err := time.Parse(time.RFC3339, h.Timestamp); err == nil {
			return t
		}
	}
	return time.Time{}
}

// dialOnce tries addr once (networkDialTimeout).
func dialOnce(addr string) error {
	ctx, cancel := context.WithTimeout(context.Background(), networkDialTimeout)
	defer cancel()
	return dialServer(ctx, addr)
}

// checkServerTriggers starts a run of each set whose server trigger is due,
// on an allowed network, and whose server answers. Called on every scheduler
// tick; the server is tried in a goroutine, so a tick never waits for it.
func (a *App) checkServerTriggers(jobs []ScheduledJob, now time.Time) {
	var history []JobHistory
	loaded := false
	for _, job := range jobs {
		if !serverTriggerOn(job) || job.WaitingForServer != "" {
			continue
		}
		// A scheduled run that is due goes first (checkAndRunScheduledJobs).
		if next, err := time.Parse(time.RFC3339, job.NextRun); err == nil && !now.Before(next) {
			continue
		}
		runningJobsMutex.Lock()
		running := runningJobs[job.ID]
		runningJobsMutex.Unlock()
		if running {
			continue
		}
		if !loaded {
			h, err := a.GetJobHistory()
			if err != nil {
				writeDebugLog(fmt.Sprintf("[Trigger] the run history could not be read: %v", err))
				return
			}
			history, loaded = h, true
		}
		if good := lastGoodRun(history, job.ID); !good.IsZero() && now.Sub(good) < job.ServerTrigger.every() {
			continue
		}
		serverTries.Lock()
		recentStart := now.Sub(serverTries.started[job.ID]) < serverRetryAfterRun
		recentDial := now.Sub(serverTries.dialed[job.ID]) < serverDialEvery
		if !recentStart && !recentDial {
			serverTries.dialed[job.ID] = now
		}
		serverTries.Unlock()
		if recentStart || recentDial {
			continue
		}
		// The On battery rule: a run it would hold back is not started; the
		// trigger tries again later, with no Reports entry for each try.
		if hold, _ := job.batteryRule().Holds(readBattery()); hold {
			continue
		}
		t := job.ServerTrigger
		if len(t.Networks) > 0 || len(t.Connections) > 0 || t.NotMetered {
			if ok, _ := networkAllows(currentNetworkInfo(netInfoMaxAge), t.Networks, t.Connections, t.NotMetered); !ok {
				continue
			}
		}
		name, addr := a.jobServerAddress(job)
		if addr == "" {
			continue
		}
		go func(job ScheduledJob) {
			if err := dialOnce(addr); err != nil {
				return
			}
			serverTries.Lock()
			serverTries.started[job.ID] = time.Now()
			serverTries.Unlock()
			writeDebugLog(fmt.Sprintf("[Trigger] %s: %s can be reached and the set is due (at most every %s): starting a backup", job.Name, name, formatDuration(job.ServerTrigger.every())))
			LogMessage("Backup", "info",
				fmt.Sprintf("%s: the server can be reached; starting a backup (at most every %d hours).", job.Name, int(job.ServerTrigger.every().Hours())),
				job.Name, MsgLogServerTriggerStart, msgParams{"hours": int(job.ServerTrigger.every().Hours())})
			a.executeScheduledJob(job, triggerServer)
		}(job)
	}
}

// dueSchedule is the schedule that gave the set's due run.
func dueSchedule(job ScheduledJob) (Schedule, bool) {
	for _, s := range job.Schedules {
		if s.ID == job.NextRunBy && s.Enabled {
			return s, true
		}
	}
	return Schedule{}, false
}

// serverGate decides, as a scheduled run of job is about to start, whether
// its schedule's server and network choices let it (see the top of this
// file). With ok false the run does not start now: it waits (and a later
// tick tries again), or was skipped or failed and has been recorded.
func (a *App) serverGate(job ScheduledJob, info *batteryRunInfo, now time.Time) bool {
	s, found := dueSchedule(job)
	wait := found && s.IfUnreachable == ifUnreachableWait
	if !found || (!wait && !s.hasNetworkFilter()) {
		return true
	}
	woken := time.Time{}
	if info != nil {
		woken = info.wokenAt
	}
	netOK, why := true, netBlockNone
	var net NetworkInfo
	if s.hasNetworkFilter() {
		net = currentNetworkInfo(netInfoMaxAge)
		netOK, why = networkAllows(net, s.Networks, nil, s.NotMetered)
	}
	if !wait {
		if netOK {
			return true
		}
		a.skipForNetwork(job, net, why, woken, now)
		return false
	}

	due, err := time.Parse(time.RFC3339, job.NextRun)
	if err != nil {
		return true
	}
	since := due
	if t, err := time.Parse(time.RFC3339, job.WaitingForServer); err == nil {
		since = t
	}
	until := due.Add(s.waitFor())
	name, addr := a.jobServerAddress(job)
	reachable := false
	if netOK {
		if addr == "" {
			reachable = true // not known here: the run finds out for itself
		} else if why := resumedRecently(now); why != "" {
			// Just woken: the network may take a moment to come back.
			reachable, _, _ = waitForServer(name, addr, why)
		} else {
			reachable = dialOnce(addr) == nil
		}
	}
	if reachable {
		if job.WaitingForServer != "" {
			a.setWaitingForServer(job.ID, "")
			start := time.Now()
			if info != nil {
				info.notes = append(info.notes, HistoryNote{
					Key:    NoteServerWaited,
					Params: msgParams{"due": due.Format(time.RFC3339), "start": start.Format(time.RFC3339)},
					Text:   fmt.Sprintf("Ran late: due at %s, waited for the server until %s.", clock(due), clock(start)),
				})
			}
			writeDebugLog(fmt.Sprintf("[Trigger] %s: %s can be reached again: starting the run due at %s", job.Name, name, clock(due)))
		}
		return true
	}
	if now.After(until) {
		a.setWaitingForServer(job.ID, "")
		a.recordServerGaveUp(job, name, since, until, info)
		a.recordJobRan(job)
		return false
	}
	if job.WaitingForServer == "" {
		a.setWaitingForServer(job.ID, now.Format(time.RFC3339))
		a.recordServerWaiting(job, name, net, why, now, until)
	}
	// Woken for this run on a network it does not run on: back to sleep.
	if !netOK && !woken.IsZero() {
		a.sleepAfterSkip(job)
	}
	return false
}

// setWaitingForServer records when a run of the set began waiting for its
// server ("" when it no longer is).
func (a *App) setWaitingForServer(setID, at string) {
	if a.delegated() {
		return
	}
	jobsFileMu.Lock()
	defer jobsFileMu.Unlock()
	jobs, err := a.GetScheduledJobs()
	if err != nil {
		writeDebugLog(fmt.Sprintf("Warning: could not record the wait for the server: %v", err))
		return
	}
	for i := range jobs {
		if jobs[i].ID != setID {
			continue
		}
		if jobs[i].WaitingForServer == at {
			return
		}
		jobs[i].WaitingForServer = at
		if err := writeJobs(jobs); err != nil {
			writeDebugLog(fmt.Sprintf("Warning: could not record the wait for the server: %v", err))
		}
		return
	}
}

// writeJobs saves the sets (callers hold jobsFileMu).
func writeJobs(jobs []ScheduledJob) error {
	path, err := getScheduledJobsPath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(jobs, "", "  ")
	if err != nil {
		return err
	}
	return atomicWriteFile(path, data, 0600)
}

// netBlockText is why a network choice stopped a run, for the English text.
func netBlockText(why string, net NetworkInfo) string {
	switch why {
	case netBlockMetered:
		return "the connection is metered"
	case netBlockConnection:
		return "not on a chosen kind of connection (" + netNames(net) + ")"
	}
	return "not on one of the set's networks (" + netNames(net) + ")"
}

// recordServerWaiting records, once, that a run began waiting for its server
// (or its network): a Reports entry and a Message Log line.
func (a *App) recordServerWaiting(job ScheduledJob, name string, net NetworkInfo, why string, now, until time.Time) {
	reason := name + " could not be reached"
	key := MsgServerWaiting
	if why != netBlockNone {
		reason, key = netBlockText(why, net), MsgServerWaitingNetwork
	}
	entry := JobHistory{
		ID:            fmt.Sprintf("%d", now.Unix()),
		Name:          job.Name,
		JobID:         job.ID,
		Timestamp:     now.Format(time.RFC3339),
		Status:        "held",
		BackupDirs:    job.BackupDirs,
		BackupID:      job.BackupID,
		UseVSS:        job.UseVSS,
		BackupType:    job.BackupType,
		Trigger:       "scheduled",
		Message:       fmt.Sprintf("At %s %s. The backup will start as soon as the server can be reached, until %s.", clock(now), reason, until.Format("Mon 15:04")),
		MessageKey:    key,
		MessageParams: msgParams{"time": now.Format(time.RFC3339), "until": until.Format(time.RFC3339)},
		logLine: &MessageLogEntry{Level: "warning", MessageKey: MsgLogServerWaiting, MessageParams: msgParams{"until": until.Format(time.RFC3339)},
			Message: fmt.Sprintf("waiting for the server (%s), until %s.", reason, until.Format("Mon 15:04"))},
	}
	writeDebugLog(fmt.Sprintf("[Trigger] %s: %s; the run waits for the server until %s", job.Name, reason, until.Format(time.RFC3339)))
	if err := a.AddJobHistory(entry); err != nil {
		writeDebugLog(fmt.Sprintf("Warning: Failed to add job history: %v", err))
	}
}

// recordServerGaveUp records a run that waited for its server until its
// wait ran out: a failed Reports entry, its log, and what the set does after
// a failed run (the failure e-mail).
func (a *App) recordServerGaveUp(job ScheduledJob, name string, since, until time.Time, info *batteryRunInfo) {
	cause := fmt.Errorf("the backup did not run: %s could not be reached from %s to %s", name, since.Format("Mon 15:04"), until.Format("Mon 15:04"))
	writeDebugLog(fmt.Sprintf("[Trigger] %s: %v", job.Name, cause))
	writeNotStartedRunLog(job.BackupID, job.Name, "scheduled", cause)
	entry := JobHistory{
		ID:            fmt.Sprintf("%d", time.Now().Unix()),
		Name:          job.Name,
		JobID:         job.ID,
		Timestamp:     time.Now().Format(time.RFC3339),
		Status:        "failed",
		Message:       "Error: " + cause.Error(),
		BackupDirs:    job.BackupDirs,
		BackupID:      job.BackupID,
		UseVSS:        job.UseVSS,
		BackupType:    job.BackupType,
		Trigger:       "scheduled",
		MessageKey:    MsgServerGaveUp,
		MessageParams: msgParams{"name": name, "since": since.Format(time.RFC3339), "until": until.Format(time.RFC3339)},
		logLine: &MessageLogEntry{Level: "error", MessageKey: MsgLogServerGaveUp, MessageParams: msgParams{"since": since.Format(time.RFC3339)},
			Message: fmt.Sprintf("the server could not be reached since %s; the run is recorded as failed.", since.Format("Mon 15:04"))},
	}
	if err := a.AddJobHistory(entry); err != nil {
		writeDebugLog(fmt.Sprintf("Warning: Failed to add job history: %v", err))
	}
	a.runPostBackupActions(job, false, entry.Message, nil, info)
}

// skipForNetwork records a scheduled run skipped as the computer is not on
// one of its schedule's networks (or the connection is metered), and puts a
// computer woken for it back to sleep. The set's next run is worked out.
func (a *App) skipForNetwork(job ScheduledJob, net NetworkInfo, why string, woken, now time.Time) {
	reason := netBlockText(why, net)
	entry := JobHistory{
		ID:            fmt.Sprintf("%d", now.Unix()),
		Name:          job.Name,
		JobID:         job.ID,
		Timestamp:     now.Format(time.RFC3339),
		Status:        "skipped",
		BackupDirs:    job.BackupDirs,
		BackupID:      job.BackupID,
		UseVSS:        job.UseVSS,
		BackupType:    job.BackupType,
		Trigger:       "scheduled",
		Message:       fmt.Sprintf("At %s the backup did not run: %s.", clock(now), reason),
		MessageKey:    MsgNetworkSkipped,
		MessageParams: msgParams{"time": now.Format(time.RFC3339)},
		logLine: &MessageLogEntry{Level: "warning", MessageKey: MsgLogNetworkSkipped,
			Message: fmt.Sprintf("the run is skipped: %s.", reason)},
	}
	if !woken.IsZero() {
		entry.Message += " The computer was woken for it and was put back to sleep."
		entry.MessageKey = MsgNetworkSkippedWoken
	}
	writeDebugLog(fmt.Sprintf("[Trigger] %s: %s: the run is skipped", job.Name, reason))
	if err := a.AddJobHistory(entry); err != nil {
		writeDebugLog(fmt.Sprintf("Warning: Failed to add job history: %v", err))
	}
	a.recordJobRan(job)
	if !woken.IsZero() {
		a.sleepAfterSkip(job)
	}
}

// sleepAfterSkip puts a computer woken for a run of job that did not start
// back to sleep, as for a run held for the battery (sleepForHeldRun).
func (a *App) sleepAfterSkip(job ScheduledJob) {
	a.refreshWakeTimer()
	go func() {
		time.Sleep(sleepAfterHoldDelay)
		if otherBackupsActive(job.ID) {
			writeDebugLog(fmt.Sprintf("[Trigger] %s: not put back to sleep: another backup is running or waiting", job.Name))
			return
		}
		writeDebugLog(fmt.Sprintf("[Trigger] %s: putting the computer back to sleep (woken on a network the set does not run on)", job.Name))
		syncWakeAlarm()
		if err := sleepComputer(); err != nil {
			writeDebugLog(fmt.Sprintf("[Trigger] could not put the computer back to sleep: %v", err))
		}
	}()
}

// ---- "Right now", for the editor

// TriggerCheck is one line of the server trigger's "Right now" panel: Key
// and Params for the window to say it in the user's language.
type TriggerCheck struct {
	OK     bool                   `json:"ok"`
	Key    string                 `json:"key"`
	Params map[string]interface{} `json:"params,omitempty"`
}

// ServerTriggerStatus is the "Right now" panel: the checks, and whether the
// trigger would start a backup now ("now"), would once it is due
// ("notYet", from NextAt), or would not ("no").
type ServerTriggerStatus struct {
	Checks  []TriggerCheck `json:"checks"`
	Verdict string         `json:"verdict"`
	NextAt  string         `json:"nextAt,omitempty"`
}

// GetServerTriggerStatus works out the "Right now" panel for the set as the
// editor has it (not yet saved): its server, its trigger's choices, its
// last backup that worked.
func (a *App) GetServerTriggerStatus(job ScheduledJob) ServerTriggerStatus {
	if a.delegated() {
		var out ServerTriggerStatus
		if err := a.apiClient.Call("GetServerTriggerStatus", &out, job); err == nil {
			return out
		}
		return ServerTriggerStatus{Checks: []TriggerCheck{}, Verdict: "no"}
	}
	t := job.ServerTrigger
	if t == nil {
		t = &ServerTrigger{Enabled: true}
	}
	st := ServerTriggerStatus{Checks: []TriggerCheck{}}
	allOK := true
	add := func(ok bool, key string, params map[string]interface{}) {
		st.Checks = append(st.Checks, TriggerCheck{OK: ok, Key: key, Params: params})
		allOK = allOK && ok
	}
	net := currentNetworkInfo(5 * time.Second)
	if net.Supported {
		if len(net.Current) == 0 {
			add(false, "trigNowNoNetwork", nil)
		} else {
			var names []string
			for _, c := range net.Current {
				names = append(names, c.Name)
			}
			add(true, "trigNowNetwork", map[string]interface{}{"names": strings.Join(names, ", ")})
		}
		if len(t.Networks) > 0 || len(t.Connections) > 0 {
			ok, _ := networkAllows(net, t.Networks, t.Connections, false)
			if ok {
				add(true, "trigNowNetOk", nil)
			} else {
				add(false, "trigNowNetNo", nil)
			}
		}
		if t.NotMetered && net.MeteredKnown {
			if net.Metered {
				add(false, "trigNowMetered", nil)
			} else {
				add(true, "trigNowNotMetered", nil)
			}
		}
	}
	name, addr := a.jobServerAddress(job)
	if addr != "" {
		if dialOnce(addr) == nil {
			add(true, "trigNowServerOk", map[string]interface{}{"name": name})
		} else {
			add(false, "trigNowServerNo", map[string]interface{}{"name": name})
		}
	}
	due := true
	if job.ID != "" {
		if hist, err := a.GetJobHistory(); err == nil {
			if good := lastGoodRun(hist, job.ID); !good.IsZero() {
				next := good.Add(t.every())
				due = !time.Now().Before(next)
				if !due {
					st.NextAt = next.Format(time.RFC3339)
				}
				st.Checks = append(st.Checks, TriggerCheck{OK: due, Key: "trigNowLastGood", Params: map[string]interface{}{"time": good.Format(time.RFC3339)}})
			} else {
				st.Checks = append(st.Checks, TriggerCheck{OK: true, Key: "trigNowNeverGood"})
			}
		}
	}
	switch {
	case allOK && due:
		st.Verdict = "now"
	case allOK:
		st.Verdict = "notYet"
	default:
		st.Verdict = "no"
	}
	return st
}
