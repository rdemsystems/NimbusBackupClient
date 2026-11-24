package main

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/salvanza/salvanza_client/gui/api"
	"pbscommon"
)

// What the computer does once a run of a Backup Set has ended: it stays as it
// is (its own sleep settings apply; the default), goes to sleep, hibernates or
// shuts down. After a wait of 0 to 240 minutes, and, when the set says so,
// only after a run its wake timer woke the computer for (wake_timer.go, and
// takeWoken in battery_rule.go). It comes after the e-mail and the "after"
// application (runPostBackupActions), after a run that worked or failed, Run
// now included; a run held back or skipped for the battery never started, so
// nothing follows it.
//
// The action waits out the set's wait, then for this run to be recorded and
// for every other backup and restore of this process, running or queued, to
// end: sleeping would cut them short. While it waits it is listed
// (GetAfterBackupPending, and the service's through GET /runs), and an open
// window shows it with Cancel. When one is carried out, any other set's action
// still waiting is dropped: the computer has done what was asked of it.

const (
	afterBackupStay      = ""
	afterBackupSleep     = "sleep"
	afterBackupHibernate = "hibernate"
	afterBackupShutdown  = "shutdown"
	// afterBackupMaxWait is the longest wait, in minutes.
	afterBackupMaxWait = 240
)

// The ways the computer is acted on, and the length of the waits: variables so
// the tests can stand in for the system and shorten the minutes.
var (
	hibernateComputer   = pbscommon.HibernateComputer
	canHibernate        = pbscommon.CanHibernate
	shutdownComputerNow = func() (string, error) {
		return pbscommon.ShutdownComputer("Shutting down after a backup, as its Backup Set asks.")
	}
	afterBackupMinute = time.Minute
	// afterBackupPoll is how often a waiting action looks again at the runs.
	afterBackupPoll = 5 * time.Second
)

// afterBackupVerb is the action as the English messages say it: "the
// computer goes to sleep"; afterBackupInfinitive as in "could not go to sleep".
func afterBackupVerb(action string) string {
	switch action {
	case afterBackupSleep:
		return "goes to sleep"
	case afterBackupHibernate:
		return "hibernates"
	case afterBackupShutdown:
		return "shuts down"
	}
	return "stays as it is"
}

func afterBackupInfinitive(action string) string {
	switch action {
	case afterBackupSleep:
		return "go to sleep"
	case afterBackupHibernate:
		return "hibernate"
	case afterBackupShutdown:
		return "shut down"
	}
	return "stay as it is"
}

var afterBackupLogKey = map[string]MessageKey{
	afterBackupSleep:     MsgLogAfterBackupSleep,
	afterBackupHibernate: MsgLogAfterBackupHibernate,
	afterBackupShutdown:  MsgLogAfterBackupShutdown,
}

// ---- the set's choice

// validateAfterBackup refuses an unknown choice and a wait out of range.
func validateAfterBackup(job ScheduledJob) error {
	switch strings.ToLower(strings.TrimSpace(job.AfterBackup)) {
	case "", "leave", "stay", afterBackupSleep, afterBackupHibernate, afterBackupShutdown:
	default:
		return fmt.Errorf("what the computer does after the backup must be stay, sleep, hibernate or shutdown, not %q", job.AfterBackup)
	}
	if job.AfterBackupWait < 0 || job.AfterBackupWait > afterBackupMaxWait {
		return fmt.Errorf("the wait after the backup must be 0 to %d minutes, not %d", afterBackupMaxWait, job.AfterBackupWait)
	}
	return nil
}

// normalizeAfterBackup puts a set's choice in order. ShutdownAfter decides
// whether it shuts down: a set saved before the choice existed has only that
// (it becomes "shutdown", at once, after any run: what it did before), and an
// older app or service that keeps the fields it does not know turns only that
// off. ShutdownAfter is then written from the choice.
func normalizeAfterBackup(job *ScheduledJob) {
	job.AfterBackup = strings.ToLower(strings.TrimSpace(job.AfterBackup))
	if job.AfterBackup == "leave" || job.AfterBackup == "stay" {
		job.AfterBackup = afterBackupStay
	}
	switch {
	case job.ShutdownAfter:
		job.AfterBackup = afterBackupShutdown
	case job.AfterBackup == afterBackupShutdown:
		job.AfterBackup = afterBackupStay
	}
	if job.AfterBackupWait < 0 {
		job.AfterBackupWait = 0
	} else if job.AfterBackupWait > afterBackupMaxWait {
		job.AfterBackupWait = afterBackupMaxWait
	}
	if job.AfterBackup == afterBackupStay {
		job.AfterBackupWait, job.AfterBackupOnlyIfWoken = 0, false
		job.AfterBackupEvenIfStopped, job.AfterBackupEvenIfInUse = false, false
	}
	job.ShutdownAfter = job.AfterBackup == afterBackupShutdown
}

// afterBackupFor is what the computer does after this run of job: "" when it
// stays as it is. woken is when the computer was woken for the run (zero when
// it was not). why says, for the log, why a set's choice is not acted on.
func afterBackupFor(job ScheduledJob, woken time.Time) (action, why string) {
	normalizeAfterBackup(&job)
	if job.AfterBackup == afterBackupStay {
		return "", ""
	}
	if job.AfterBackupOnlyIfWoken && woken.IsZero() {
		return "", fmt.Sprintf("the computer %s only after a run it was woken for, and it was not woken for this one: it stays as it is", afterBackupVerb(job.AfterBackup))
	}
	return job.AfterBackup, ""
}

// ---- actions waiting

// pendingAfterBackup is an action waiting: its set's wait, then the runs.
// stop is closed when it is cancelled, or dropped for another set's action.
type pendingAfterBackup struct {
	id, setID, set, action string
	at                     time.Time
	// since is the wake the run was woken for, else the run's start: use of
	// the computer after it keeps the computer as it is.
	since            time.Time
	waitingForOthers atomic.Bool
	stop             chan struct{}
	once             sync.Once
}

func (p *pendingAfterBackup) cancel() { p.once.Do(func() { close(p.stop) }) }

func (p *pendingAfterBackup) cancelled() bool {
	select {
	case <-p.stop:
		return true
	default:
		return false
	}
}

// wait sleeps d; false when the action was cancelled meanwhile.
func (p *pendingAfterBackup) wait(d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-p.stop:
		return false
	case <-t.C:
		return true
	}
}

// afterBackupSleptGap: wall time this much longer than a wait means the
// computer slept during it. afterBackupNow is the wall clock (tests move it).
var (
	afterBackupSleptGap = 2 * time.Minute
	afterBackupNow      = time.Now
)

// waitAwake waits d in steps of poll, watching the clock; false when the
// action was cancelled, or the computer slept meanwhile (slept is then set).
// The computer is kept awake while an action waits, so it sleeps only when
// someone puts it to sleep, or closes a laptop's lid: what was asked before
// then is dropped, or a later wake would carry it out on whoever woke it.
func (p *pendingAfterBackup) waitAwake(d, poll time.Duration, slept *bool) bool {
	for d > 0 {
		step := poll
		if step <= 0 || step > d {
			step = d
		}
		before := afterBackupNow().Round(0) // wall clock: it counts the time asleep
		if !p.wait(step) {
			return false
		}
		if afterBackupNow().Round(0).Sub(before) > step+afterBackupSleptGap {
			*slept = true
			return false
		}
		d -= step
	}
	return true
}

// keepAwakeForAfterBackup keeps the computer awake while an action waits; a
// variable so the tests do not take the real system's request.
var keepAwakeForAfterBackup = func(set, action string) func() {
	end, err := preventSleep(product().Title + ": waiting to " + afterBackupInfinitive(action) + " after a backup")
	if err != nil {
		writeDebugLog(fmt.Sprintf("[After backup] %s: %v: the computer may go to sleep on its own before it %s", set, err, afterBackupVerb(action)))
		return func() {}
	}
	return end
}

// afterBackups are the actions waiting in this process, one per set: a set's
// next run replaces its earlier one's.
var afterBackups = struct {
	sync.Mutex
	bySet map[string]*pendingAfterBackup
	seq   int
}{bySet: map[string]*pendingAfterBackup{}}

func registerAfterBackup(job ScheduledJob, action string, at time.Time) *pendingAfterBackup {
	afterBackups.Lock()
	defer afterBackups.Unlock()
	if old := afterBackups.bySet[job.ID]; old != nil {
		old.cancel()
		writeDebugLog(fmt.Sprintf("[After backup] %s: what its previous run was waiting to do is replaced by this run's", job.Name))
	}
	afterBackups.seq++
	p := &pendingAfterBackup{id: fmt.Sprintf("after-%d", afterBackups.seq), setID: job.ID, set: job.Name, action: action, at: at, stop: make(chan struct{})}
	afterBackups.bySet[job.ID] = p
	return p
}

func forgetAfterBackup(p *pendingAfterBackup) {
	afterBackups.Lock()
	defer afterBackups.Unlock()
	if afterBackups.bySet[p.setID] == p {
		delete(afterBackups.bySet, p.setID)
	}
}

// waitForAfterBackup returns once nothing is waiting to be done to the
// computer after a run of the set: carried out, cancelled, or none at all.
func waitForAfterBackup(setID string) {
	for {
		afterBackups.Lock()
		_, waiting := afterBackups.bySet[setID]
		afterBackups.Unlock()
		if !waiting {
			return
		}
		time.Sleep(afterBackupPoll)
	}
}

// claimAfterBackup takes p's turn to act; false when it was cancelled. Every
// other action still waiting is dropped: the computer is about to do what was
// asked of it.
func claimAfterBackup(p *pendingAfterBackup) bool {
	afterBackups.Lock()
	defer afterBackups.Unlock()
	if p.cancelled() {
		return false
	}
	for id, o := range afterBackups.bySet {
		if o != p {
			o.cancel()
			writeDebugLog(fmt.Sprintf("[After backup] %s: dropped, as the computer %s for %s", o.set, afterBackupVerb(p.action), p.set))
		}
		delete(afterBackups.bySet, id)
	}
	return true
}

// cancelAfterBackup cancels the action waiting under id (the Cancel of the
// window's notice): the computer stays as it is.
func cancelAfterBackup(id string) error {
	afterBackups.Lock()
	var p *pendingAfterBackup
	for setID, o := range afterBackups.bySet {
		if o.id == id {
			p = o
			delete(afterBackups.bySet, setID)
		}
	}
	afterBackups.Unlock()
	if p == nil {
		return fmt.Errorf("nothing is waiting to be done after that backup any more")
	}
	p.cancel()
	writeDebugLog(fmt.Sprintf("[After backup] %s: cancelled in the app: the computer stays as it is", p.set))
	LogMessage("Backup", "info", fmt.Sprintf("%s: the computer stays as it is: what it was to do after the backup was cancelled.", p.set),
		p.set, MsgLogAfterBackupCancelled, nil)
	return nil
}

// listAfterBackups is the actions waiting in this process, oldest first.
func listAfterBackups() []api.AfterBackupPending {
	afterBackups.Lock()
	list := make([]*pendingAfterBackup, 0, len(afterBackups.bySet))
	for _, p := range afterBackups.bySet {
		list = append(list, p)
	}
	afterBackups.Unlock()
	sort.Slice(list, func(i, j int) bool {
		if !list[i].at.Equal(list[j].at) {
			return list[i].at.Before(list[j].at)
		}
		return list[i].id < list[j].id
	})
	out := make([]api.AfterBackupPending, 0, len(list))
	for _, p := range list {
		out = append(out, api.AfterBackupPending{ID: p.id, Set: p.set, Action: p.action, At: p.at.Format(time.RFC3339), WaitingForOthers: p.waitingForOthers.Load()})
	}
	return out
}

// ---- after a run

// startAfterBackup sets off what the computer does after this run of job, if
// anything. run is what the battery did around the run (nil for none): it
// holds whether the computer was woken for it.
func (a *App) startAfterBackup(job ScheduledJob, run *batteryRunInfo) {
	var woken time.Time
	if run != nil {
		woken = run.wokenAt
	}
	action, why := afterBackupFor(job, woken)
	if action == "" {
		if why != "" {
			writeDebugLog(fmt.Sprintf("[After backup] %s: %s", job.Name, why))
		}
		return
	}
	normalizeAfterBackup(&job)
	wait := time.Duration(job.AfterBackupWait) * afterBackupMinute
	p := registerAfterBackup(job, action, time.Now().Add(wait))
	switch {
	case run != nil && !run.wokenAt.IsZero():
		p.since = run.wokenAt
	case run != nil && !run.startedAt.IsZero():
		p.since = run.startedAt
	default:
		p.since = time.Now()
	}
	if job.AfterBackupWait > 0 {
		writeDebugLog(fmt.Sprintf("[After backup] %s: the computer %s in %d minute(s), at about %s, once no other backup or restore is running, unless cancelled",
			job.Name, afterBackupVerb(action), job.AfterBackupWait, p.at.Format("15:04")))
	} else {
		writeDebugLog(fmt.Sprintf("[After backup] %s: the computer %s once this run has been recorded and no other backup or restore is running, unless cancelled",
			job.Name, afterBackupVerb(action)))
	}
	go a.runAfterBackup(job, p, wait, afterBackupPoll)
}

// setIsRunning reports whether a run of the set is under way in this process
// (executeScheduledJob holds it until the run is recorded).
func setIsRunning(setID string) bool {
	runningJobsMutex.Lock()
	defer runningJobsMutex.Unlock()
	return runningJobs[setID]
}

// runAfterBackup waits for p's turn and carries it out, unless it is
// cancelled first. poll is how often it looks again at the runs.
func (a *App) runAfterBackup(job ScheduledJob, p *pendingAfterBackup, wait, poll time.Duration) {
	defer forgetAfterBackup(p)
	// Awake while it waits: Windows puts a computer woken by a timer back to
	// sleep after 2 minutes with no one at it, before a longer wait was over.
	defer keepAwakeForAfterBackup(job.Name, p.action)()
	slept := false
	defer func() {
		if slept {
			writeDebugLog(fmt.Sprintf("[After backup] %s: the computer slept while it waited to %s: dropped, so it is not done when it next wakes", job.Name, afterBackupInfinitive(p.action)))
		}
	}()
	if wait > 0 && !p.waitAwake(wait, poll, &slept) {
		return
	}
	// This run first: its set is held as running until the run is recorded,
	// a moment after these actions start.
	for setIsRunning(job.ID) {
		if !p.waitAwake(poll, poll, &slept) {
			return
		}
	}
	for otherBackupsActive(job.ID) {
		if !p.waitingForOthers.Swap(true) {
			writeDebugLog(fmt.Sprintf("[After backup] %s: waiting for the other backups and restores to finish before the computer %s", job.Name, afterBackupVerb(p.action)))
		}
		if !p.waitAwake(poll, poll, &slept) {
			return
		}
	}
	// Not when someone has used the computer since it woke or the run began,
	// unless the set says so: they are at it.
	if !job.AfterBackupEvenIfInUse {
		at, used := usedSince(p.since)
		if used {
			writeDebugLog(fmt.Sprintf("[After backup] %s: someone used the computer at %s, after %s: it stays as it is", job.Name, at.Format("15:04:05"), p.since.Format("15:04:05")))
			LogMessage("Backup", "info", fmt.Sprintf("%s: the computer stays as it is: someone has used it since it woke or the backup started.", job.Name), job.Name, MsgLogAfterBackupInUse, nil)
			return
		}
		if at.IsZero() {
			writeDebugLog(fmt.Sprintf("[After backup] %s: no one is known to be using the computer (no one signed in, or no way to tell)", job.Name))
		} else {
			writeDebugLog(fmt.Sprintf("[After backup] %s: the computer was last used at %s, before %s", job.Name, at.Format("2006-01-02 15:04:05"), p.since.Format("15:04:05")))
		}
	}
	if !claimAfterBackup(p) {
		return
	}
	a.carryOutAfterBackup(job, p.action)
}

// carryOutAfterBackup puts the computer to sleep, hibernates it or shuts it
// down. Before sleep or hibernation the wake timer is set for the next run,
// so a set that wakes the computer still runs on time, and the system's wake
// alarm is checked to match it (syncWakeAlarm) before the call is made.
func (a *App) carryOutAfterBackup(job ScheduledJob, action string) {
	// The app offers Hibernate only where it works, but a set saved otherwise
	// (a settings file, the API), or on a computer whose hibernation was
	// turned off since, was found to log "hibernates now" and do nothing at
	// all (a VM 2026-11-20): say why instead, and only that.
	if action == afterBackupHibernate {
		if c := canHibernate(); !c.OK {
			writeDebugLog(fmt.Sprintf("[After backup] %s: not hibernated, it stays as it is: %s", job.Name, c.Reason()))
			LogMessage("Backup", "warning", fmt.Sprintf("%s: the computer did not go to sleep, hibernate or shut down after the backup: %s", job.Name, c.Reason()),
				job.Name, MsgLogAfterBackupFailed, msgParams{"error": c.Reason()})
			return
		}
	}
	writeDebugLog(fmt.Sprintf("[After backup] %s: the computer %s now", job.Name, afterBackupVerb(action)))
	LogMessage("Backup", "info", fmt.Sprintf("%s: the computer %s after the backup, as this set is set to do.", job.Name, afterBackupVerb(action)),
		job.Name, afterBackupLogKey[action], nil)
	var err error
	switch action {
	case afterBackupSleep:
		a.refreshWakeTimer()
		syncWakeAlarm()
		err = sleepComputer()
	case afterBackupHibernate:
		a.refreshWakeTimer()
		syncWakeAlarm()
		err = hibernateComputer()
	case afterBackupShutdown:
		var method string
		if method, err = shutdownComputerNow(); err == nil {
			writeDebugLog("[After backup] shutdown requested through " + method)
		}
	}
	if err != nil {
		writeDebugLog(fmt.Sprintf("[After backup] %s: the computer could not %s: %v", job.Name, afterBackupInfinitive(action), err))
		LogMessage("Backup", "warning", fmt.Sprintf("%s: the computer did not go to sleep, hibernate or shut down after the backup: %v", job.Name, err),
			job.Name, MsgLogAfterBackupFailed, msgParams{"error": err.Error()})
	}
}

// ---- for the window

// AfterBackupNotice is an action waiting after a run, for the window's notice
// with Cancel: Set's name, Action ("sleep", "hibernate" or "shutdown"), At
// when the set's wait ends (RFC 3339), WaitingForOthers once the wait is over
// and other backups or restores are still running.
type AfterBackupNotice struct {
	ID               string `json:"id"`
	Set              string `json:"set"`
	Action           string `json:"action"`
	At               string `json:"at"`
	WaitingForOthers bool   `json:"waitingForOthers"`
}

// GetAfterBackupPending lists the actions waiting after a run: this
// process's, and the service's while it runs the backups (their IDs carry
// serviceRunPrefix, for CancelAfterBackup).
func (a *App) GetAfterBackupPending() []AfterBackupNotice {
	out := []AfterBackupNotice{}
	add := func(list []api.AfterBackupPending, prefix string) {
		for _, p := range list {
			out = append(out, AfterBackupNotice{ID: prefix + p.ID, Set: p.Set, Action: p.Action, At: p.At, WaitingForOthers: p.WaitingForOthers})
		}
	}
	add(listAfterBackups(), "")
	if a.serviceRunsBackups() {
		if runs, err := a.apiClient.GetRuns(); err == nil {
			add(runs.AfterBackup, serviceRunPrefix)
		}
	}
	return out
}

// CancelAfterBackup cancels the action listed under id: the computer stays as
// it is.
func (a *App) CancelAfterBackup(id string) error {
	if strings.HasPrefix(id, serviceRunPrefix) {
		if !a.serviceRunsBackups() {
			return fmt.Errorf("not connected to the background service")
		}
		return a.apiClient.Call("CancelAfterBackup", nil, strings.TrimPrefix(id, serviceRunPrefix))
	}
	return cancelAfterBackup(id)
}

// AfterBackupPendingForAPI is the service's list for GET /runs
// (api.AfterBackupAPI).
func (a *App) AfterBackupPendingForAPI() []api.AfterBackupPending {
	return listAfterBackups()
}

// PowerActionsInfo is what the Before and after tab needs to offer the choices: whether
// this computer can hibernate and, when not, why, as a code the app
// translates: "off" (turned off, Windows), "none" (no hibernate state),
// "kernel" (turned off in the Linux kernel), "swap" (no or not enough swap
// space: SwapNeedGB is about what is needed, SwapHaveGB what there is, 0 for
// none) or "noresume" (no resume device set). HibernateReason is the same in
// English.
type PowerActionsInfo struct {
	Hibernate       bool   `json:"hibernate"`
	HibernateWhy    string `json:"hibernateWhy"`
	HibernateReason string `json:"hibernateReason"`
	SwapNeedGB      uint64 `json:"swapNeedGB"`
	SwapHaveGB      uint64 `json:"swapHaveGB"`
}

// GetPowerActionsInfo reads it from this computer (the app and its service
// are on the same one, so the app reads it itself).
func (a *App) GetPowerActionsInfo() PowerActionsInfo {
	c := canHibernate()
	info := PowerActionsInfo{Hibernate: c.OK, HibernateWhy: c.Why, HibernateReason: c.Reason()}
	if c.Why == pbscommon.HibernateSwap {
		info.SwapNeedGB, info.SwapHaveGB = c.SwapGB()
	}
	return info
}
