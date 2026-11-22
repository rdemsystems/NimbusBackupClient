package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"pbscommon"

	"github.com/salvanza/salvanza_client/gui/api"
)

// runningJobs tracks currently executing jobs to prevent duplicates
var runningJobs = make(map[string]bool)
var runningJobsMutex sync.Mutex

// ScheduledJob represents a scheduled backup job
type ScheduledJob struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	ScheduleTime string `json:"scheduleTime"` // HH:MM format, used when TriggerMode == "daily"
	// RunAtStartup and WakeToRun are the model before ComputerEvents and a
	// wake choice per schedule: only read to move a set saved before them, and
	// written back from them for an older app (computer_events.go).
	RunAtStartup bool `json:"runAtStartup"`
	WakeToRun    bool `json:"wakeToRun,omitempty"`
	// The set's "On battery" rule (battery_rule.go). OnBattery is "" (or
	// "anyway"): start whatever the charge, which every set saved before the
	// rule existed does; "above": start only at BatteryStartPercent or more
	// (0 means 30); "plugged": wait for mains power. BatteryCancel with
	// BatteryCancelBelow over 0 cancels a running backup below that charge.
	// The zero values are the behaviour before the rule: start, never cancel.
	OnBattery           string   `json:"onBattery,omitempty"`
	BatteryStartPercent int      `json:"batteryStartPercent,omitempty"`
	BatteryCancel       bool     `json:"batteryCancel,omitempty"`
	BatteryCancelBelow  int      `json:"batteryCancelBelow,omitempty"`
	BackupDirs          []string `json:"backupDirs"`
	DriveLetters        []string `json:"driveLetters"` // physical disks for machine backups
	// DiskIDs is the identity of each disk in DriveLetters, at the same index
	// ("" where none could be read), so a run backs up the same disks after
	// the system numbers them differently (diskpin.go). Empty for a set saved
	// before identities were recorded: it is backed up by its paths, and gets
	// identities when it is next saved.
	DiskIDs []string `json:"diskIds,omitempty"`
	// SkipPartitions are partitions of the set's disks that it leaves out:
	// they are not read or snapshotted and go into the image as zeros, so a
	// restore gives them back empty and unformatted. A run leaves one out only
	// while its disk still has it (skipsByDevice); otherwise it is backed up.
	SkipPartitions []PartitionSkip `json:"skipPartitions,omitempty"`
	BackupID       string          `json:"backupId"`
	// PBSServerID is which configured PBS server this set backs up to; ""
	// means the app's current default server (same convention Restore's own
	// server dropdown already uses).
	PBSServerID string `json:"pbsServerId,omitempty"`
	UseVSS      bool   `json:"useVSS"`
	BackupType  string `json:"backupType"`
	// MachineAsVM stores a machine backup set as a Proxmox VE "vm" snapshot
	// instead of a "host" one. BackupID must then be the
	// numeric PBS ID, computed by the editor, so the same set always lands in
	// the same vm/<id> group.
	MachineAsVM bool `json:"machineAsVm,omitempty"`
	// ReadFreeSpace makes a machine set a forensic copy: free space and the
	// paging/hibernation files are read too, instead of going into the image
	// as zeros (the CLI's -readfreespace). Ignored for folder sets.
	ReadFreeSpace bool `json:"readFreeSpace,omitempty"`
	// SnapshotsOff is a Linux machine set with Use snapshots turned off, and
	// LiveRead the owner's acceptance that its image may then be
	// inconsistent. With both, mounted disks are read live; with
	// SnapshotsOff alone the run stops before it starts, saying the override
	// is not ticked. With neither, a mounted disk that cannot be snapshotted
	// stops the run. Ignored for folder sets and on Windows.
	SnapshotsOff bool `json:"snapshotsOff,omitempty"`
	LiveRead     bool `json:"liveRead,omitempty"`
	// ChangeDetectionMode is the folder-backup format and change detection,
	// as in the official client: "" or "legacy" (classic archive, every file
	// read), "data" (split archive, every file read) or "metadata" (split
	// archive, unchanged files reused without reading). Ignored for machine
	// sets, which back up disk images.
	ChangeDetectionMode string `json:"changeDetectionMode,omitempty"`
	// FullReadEvery makes every Nth run of a metadata-mode set read every
	// file (a data-mode run), catching changes that leave size and mtime
	// untouched and dropping padding from reuse. 0 means never.
	FullReadEvery int `json:"fullReadEvery,omitempty"`
	// MetadataRunsSinceFullRead is run state, not a setting: metadata-mode
	// runs that completed successfully since the last successful full read
	// (recordChangeDetectionOutcome). Kept across edits.
	MetadataRunsSinceFullRead int `json:"metadataRunsSinceFullRead,omitempty"`
	// FullReadDay is run state too: the local day (2006-01-02) this set's
	// last scheduled full read was dispatched. Only one set a day gets one
	// (claimFullReadDay); a set due on a day another set has taken reads
	// unchanged files quickly and is due again on its next run.
	FullReadDay string `json:"fullReadDay,omitempty"`
	// LastFullRead is when a run of this metadata-mode set that read every
	// file last completed (RFC 3339), for the set's card.
	LastFullRead string `json:"lastFullRead,omitempty"`
	// ScheduledFullRead is per-run state carried by the job copy a run is
	// dispatched with, never stored: this run is the set's scheduled full read.
	ScheduledFullRead bool `json:"-"`
	// RequestedRead is per-run state too: "full" or "quick" when this run was
	// started with Run now (full read) / Run now (quick), "" otherwise.
	RequestedRead string `json:"-"`
	// FullReadRequested: this run reads every file because Run now (full
	// read) asked for it (the card says so instead of "scheduled").
	FullReadRequested bool     `json:"-"`
	ExcludeList       []string `json:"excludeList"`
	Compression       string   `json:"compression"`       // kept for older sets; not read: Config.CompressionLevel applies
	LastRun           string   `json:"lastRun,omitempty"` // ISO timestamp
	NextRun           string   `json:"nextRun,omitempty"` // ISO timestamp
	// NextRunBy is the ID of the schedule that gives NextRun.
	NextRunBy string `json:"nextRunBy,omitempty"`
	Enabled   bool   `json:"enabled"`

	// ScheduleMode is "scheduled" (the set runs on Schedules) or "manual"
	// (Run Now only; Schedules are kept for when it is switched back). Empty
	// for a set saved before schedules existed: jobSchedules reads it from
	// TriggerMode and the fields below, and every load moves it over
	// (normalizeJobSchedules). See schedules.go.
	ScheduleMode string     `json:"scheduleMode,omitempty"`
	Schedules    []Schedule `json:"schedules,omitempty"`
	// ServerTrigger runs the set when its server can be reached, at most
	// every so many hours (server_trigger.go). nil for a set without one,
	// which is every set saved before it existed; an older app reading the
	// file ignores it.
	ServerTrigger *ServerTrigger `json:"serverTrigger,omitempty"`
	// ComputerEvents runs the set when the computer starts, wakes, someone
	// signs in or it has been idle (computer_events.go). nil for a set without
	// one; a set saved before it existed has RunAtStartup moved into it.
	ComputerEvents *ComputerEvents `json:"computerEvents,omitempty"`
	// WaitingForServer is run state, not a setting: when a run of a schedule
	// that waits for its server began waiting (RFC 3339), "" when none is.
	// The set's card shows it; the run ends it, however it ends.
	WaitingForServer string `json:"waitingForServer,omitempty"`

	// Sets saved by the project's app before the fork was merged (1.2.0-rc)
	// keep their server, machine kind and weekly or monthly schedule in these
	// keys. They are only read: migrateUpstreamJob moves them into the fields
	// above when the file is loaded, and they are never written again.
	UpstreamFrequency  string `json:"frequency,omitempty"`   // "daily" | "weekly" | "monthly"
	UpstreamWeekdays   []int  `json:"weekdays,omitempty"`    // weekly: 0 = Sunday ... 6 = Saturday
	UpstreamDayOfMonth int    `json:"dayOfMonth,omitempty"`  // monthly: 1-31
	UpstreamPBSID      string `json:"pbs_id,omitempty"`      // the set's PBS server
	UpstreamBackupKind string `json:"backup_kind,omitempty"` // "host" | "vm" (machine sets)

	// TriggerMode and the fields below are the model before Schedules. They
	// are still written, from the first enabled schedule where they can say
	// it and "manual" where they cannot (mirrorLegacyTrigger), so an older
	// app or service reading this file never runs a set at the wrong times.
	//
	// TriggerMode selects how ScheduleTime/the interval fields below are
	// interpreted. "" (the zero value, matching every job saved before this
	// field existed) and "daily" both mean the original single
	// once-a-day-at-ScheduleTime behavior — deliberately kept as the default
	// so existing saved jobs keep running exactly as before with no
	// migration step. "interval" means "repeat every IntervalMinutes,
	// optionally only within a daily time window" (mirrors BFW's own
	// "Repeatedly backup every N minutes" scheduler mode). "manual" means
	// the job (a "Backup Set" in the UI) never fires on its own — NextRun is
	// always "" and the only way it runs is RunScheduledJobNow.
	TriggerMode string `json:"triggerMode,omitempty"` // "daily" | "interval" | "manual"

	// Interval-mode fields. Only meaningful when TriggerMode == "interval".
	IntervalMinutes int    `json:"intervalMinutes,omitempty"`
	WindowAllDay    bool   `json:"windowAllDay,omitempty"` // true = no time-of-day restriction on the interval
	WindowStart     string `json:"windowStart,omitempty"`  // HH:MM, used when !WindowAllDay
	WindowEnd       string `json:"windowEnd,omitempty"`    // HH:MM, used when !WindowAllDay

	// DaysOfWeek restricts BOTH trigger modes to specific days
	// ("Mon".."Sun", English 3-letter, matching the frontend checkboxes).
	// Empty/nil means every day — the behavior every job had before this
	// field existed, so this is also a safe no-migration default.
	DaysOfWeek []string `json:"daysOfWeek,omitempty"`

	// ImageFolders are the folders of a machine set's disks that Undelete and
	// Roll back work on (read from the disk images). Empty means the user's
	// Desktop, Documents, Pictures, Music, Videos and Downloads.
	ImageFolders []string `json:"imageFolders,omitempty"`

	// ==================== POST-BACKUP ACTIONS ====================
	// Modelled on Backup for Workgroups' "Special Items" wizard step. All
	// independent of each other and off by default — every field here is a
	// safe no-migration default (zero value) for jobs saved before they
	// existed. EmailOnSuccess/EmailOnFailure each reuse the one global SMTP
	// account (Config.SMTP*/EmailFrom) but have their own recipient, since
	// a failure notification often wants to go somewhere more urgent than a
	// routine success one.
	EmailOnSuccess   bool   `json:"emailOnSuccess,omitempty"`
	EmailOnSuccessTo string `json:"emailOnSuccessTo,omitempty"`
	EmailOnFailure   bool   `json:"emailOnFailure,omitempty"`
	EmailOnFailureTo string `json:"emailOnFailureTo,omitempty"`
	// RunAppBefore/RunAppAfter are shell commands (cmd /C on Windows, sh -c
	// elsewhere), empty means disabled. Best-effort: a failing command is
	// logged, never blocks or fails the backup itself.
	RunAppBefore string `json:"runAppBefore,omitempty"`
	RunAppAfter  string `json:"runAppAfter,omitempty"`
	// ExitAppAfter and AfterBackup fire after RunAppAfter and the notification
	// email, in that order: a shutdown is irreversible, so it is always the
	// last thing attempted.
	ExitAppAfter bool `json:"exitAppAfter,omitempty"`
	// AfterBackup is what the computer does once a run of the set has ended
	// (after_backup.go): "" stays as it is, "sleep", "hibernate" or
	// "shutdown"; after AfterBackupWait minutes, and with
	// AfterBackupOnlyIfWoken only after a run the set's wake timer woke the
	// computer for.
	AfterBackup            string `json:"afterBackup,omitempty"`
	AfterBackupWait        int    `json:"afterBackupWait,omitempty"`
	AfterBackupOnlyIfWoken bool   `json:"afterBackupOnlyIfWoken,omitempty"`
	// The two that keep the computer as it is when someone is at it, both on
	// unless the set says otherwise, so a set saved before they existed keeps
	// them: after a run stopped by hand (AfterBackupEvenIfStopped turns that
	// off), and when someone used the computer after it woke or the run began
	// (AfterBackupEvenIfInUse turns that off).
	AfterBackupEvenIfStopped bool `json:"afterBackupEvenIfStopped,omitempty"`
	AfterBackupEvenIfInUse   bool `json:"afterBackupEvenIfInUse,omitempty"`
	// ShutdownAfter is the older form of AfterBackup "shutdown" (at once,
	// after any run). It is still written, true exactly when AfterBackup is
	// "shutdown", so an older app or service reading the file shuts down
	// when it should and at no other time (normalizeAfterBackup).
	ShutdownAfter bool `json:"shutdownAfter,omitempty"`
	// MinimizeAfter hides the window to the tray when the backup finishes
	// (ignored when ExitAppAfter is set; AfterBackup still follows).
	MinimizeAfter bool `json:"minimizeAfter,omitempty"`
}

// encryptionLabel maps "was an encryption key in use" to the JobHistory value.
func encryptionLabel(keyInUse bool) string {
	if keyInUse {
		return "encrypted"
	}
	return "unencrypted"
}

// JobHistory represents a completed backup job
type JobHistory struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// JobID is the ID of the Backup Set that ran this backup, so the run stays
	// with its set through a rename and a later set of the same name does not
	// show it as its own (history_set_id.go). Empty for a restore and for a
	// one-off backup; historyNoSet for an older entry whose set is gone.
	JobID      string   `json:"jobId,omitempty"`
	Timestamp  string   `json:"timestamp"` // ISO format
	Status     string   `json:"status"`    // "success", "failed", "running"
	Message    string   `json:"message"`
	BackupDirs []string `json:"backupDirs"`
	BackupID   string   `json:"backupId"`
	UseVSS     bool     `json:"useVSS"`

	// Encryption is "encrypted" when the run uploaded with an unlocked
	// encryption key, "unencrypted" when it ran with none. Empty for entries
	// persisted before this field existed, for restores, and for runs that
	// failed before the key was resolved; the frontend shows those as unknown.
	Encryption string `json:"encryption,omitempty"`
	// ChangeDetection is the folder backup's mode for this run ("legacy",
	// "data" or "metadata"; empty for machine backups and older entries).
	// ReusedFiles/ReusedBytes count unchanged files metadata mode reused
	// without reading them (0 when not known for this run).
	ChangeDetection string `json:"changeDetection,omitempty"`
	ReusedFiles     uint64 `json:"reusedFiles,omitempty"`
	ReusedBytes     uint64 `json:"reusedBytes,omitempty"`

	// BackupType is "directory" or "machine" — which top-level pipeline ran.
	// Always set for entries written from now on (each OnComplete closure in
	// main.go is specific to one type, so it's a literal, not derived from
	// anything). Empty for entries persisted before this field existed.
	BackupType string `json:"backupType,omitempty"`

	// Trigger says how this run started: "scheduled" (automatic, interval
	// fired), "startup" (automatic, RunAtStartup fired), "manual" (user
	// clicked "Run Now" on a Backup Set), or "oneoff" (a genuine one-off
	// backup from the Backup tab, with no Backup Set involved at all).
	// Empty for entries persisted before this field existed — the frontend
	// shows those as unknown rather than guessing.
	Trigger string `json:"trigger,omitempty"`

	// MessageKey/MessageParams — see BackupStatus's fields of the same name
	// in backup_status.go and msgcodes.go's doc comment. Empty on every entry
	// persisted before this field existed; the frontend falls back to
	// Message for those.
	MessageKey    MessageKey `json:"message_key,omitempty"`
	MessageParams msgParams  `json:"message_params,omitempty"`

	// Notes are further lines Reports shows with the run, each translated
	// like the message (a run held back for the battery that ran late, the
	// charge at its start and end). Empty for most runs and older entries.
	Notes []HistoryNote `json:"notes,omitempty"`
	// logLine, when set, is what AddJobHistory writes to the Message Log in
	// place of the entry's own message (a short line for a long Reports
	// text). Never stored.
	logLine *MessageLogEntry

	// NameKey/NameParams let Reports show a restore's Name ("Restore of ...",
	// "Undelete: ...", "Roll back: ...") in the user's language. Name stays
	// the English text for older readers.
	NameKey    string    `json:"name_key,omitempty"`
	NameParams msgParams `json:"name_params,omitempty"`

	// Kind is "restore" for a restore run; empty means a backup (every entry
	// written before restores were recorded). Status for a restore is
	// "success", "failed" or "cancelled" (the user pressed Stop).
	Kind string `json:"kind,omitempty"`

	// Restore-only details, empty/zero for backups. BackupID above holds the
	// restored backup ID; RestoreSnapshot is the snapshot's UTC timestamp.
	// RestoreKind is "undelete" or "rollback" for those restores, empty for
	// an ordinary one.
	RestoreKind     string   `json:"restoreKind,omitempty"`
	RestoreSnapshot string   `json:"restoreSnapshot,omitempty"`
	RestoreDest     string   `json:"restoreDest,omitempty"`
	RestorePaths    []string `json:"restorePaths,omitempty"` // selected paths; empty = whole snapshot
	RestoreFiles    int      `json:"restoreFiles,omitempty"`
	RestoreBytes    int64    `json:"restoreBytes,omitempty"`
	// Verify-after-restore outcome; RestoreVerifyRan distinguishes "checked,
	// 0 mismatches" from "not checked".
	RestoreVerifyRan    bool `json:"restoreVerifyRan,omitempty"`
	RestoreVerified     int  `json:"restoreVerified,omitempty"`
	RestoreVerifyFailed int  `json:"restoreVerifyFailed,omitempty"`
	DurationSec         int  `json:"durationSec,omitempty"`
}

// jobsFileMu and historyFileMu keep this process's read-change-write of
// scheduled_jobs.json and job_history.json one at a time. With several runs
// at once, two finishing together each read the file, changed their own part
// and wrote it back, and the second write lost the first's (a Last run, a
// full-read count, a Reports entry). Readers need no lock: a file is
// replaced whole (atomicWriteFile).
var (
	jobsFileMu    sync.Mutex
	historyFileMu sync.Mutex
)

func getScheduledJobsPath() (string, error) {
	dir, err := getConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "scheduled_jobs.json"), nil
}

func getJobHistoryPath() (string, error) {
	dir, err := getConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "job_history.json"), nil
}

// SaveScheduledJob saves a new scheduled job
func (a *App) SaveScheduledJob(job ScheduledJob) error {
	// A folder on a mapped drive letter is kept as the share's path: the
	// service does not see the letters a user maps (resolveMappedDrives).
	job.BackupDirs = resolveMappedDrives(job.BackupDirs)
	if a.delegated() {
		return a.apiClient.Call("SaveScheduledJob", nil, job)
	}
	if err := a.checkJobsChangeAllowed(); err != nil {
		return err
	}
	writeDebugLog(fmt.Sprintf("SaveScheduledJob called for: %s", job.Name))
	if err := validateScheduledJob(job); err != nil {
		return err
	}
	normalizeJobSchedules(&job)
	pinJobDisks(&job)

	jobsFileMu.Lock()
	defer jobsFileMu.Unlock()
	// Load existing jobs
	jobs, err := a.GetScheduledJobs()
	if err != nil {
		// Refuse to continue: appending to an empty list and writing it back
		// would delete every existing job if the read/parse failed transiently.
		return fmt.Errorf("failed to load existing jobs (refusing to overwrite): %w", err)
	}
	if err := checkUniqueJobName(jobs, job); err != nil {
		return err
	}

	// Set enabled by default
	job.Enabled = true
	job.WaitingForServer = ""

	// Calculate next run
	setNextRun(&job, time.Now())

	// Add new job
	jobs = append(jobs, job)

	// Save to file
	jobsPath, err := getScheduledJobsPath()
	if err != nil {
		return fmt.Errorf("failed to get jobs path: %w", err)
	}

	data, err := json.MarshalIndent(jobs, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal jobs: %w", err)
	}

	if err := atomicWriteFile(jobsPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write jobs file: %w", err)
	}

	writeDebugLog(fmt.Sprintf("Scheduled job saved: %s (next run: %s)", job.Name, job.NextRun))
	a.wakeTimerAfterSetsChanged()

	// Note: For automatic execution after reboot, use the MSI installer
	// which installs the app's service as a Windows Service

	return nil
}

// GetScheduledJobs returns all scheduled jobs
func (a *App) GetScheduledJobs() ([]ScheduledJob, error) {
	if a.delegated() {
		maps, err := a.apiClient.GetScheduledJobsFull()
		if err != nil {
			return []ScheduledJob{}, err
		}
		jobs := []ScheduledJob{}
		if err := remarshal(maps, &jobs); err != nil {
			return nil, err
		}
		return jobs, nil
	}
	jobsPath, err := getScheduledJobsPath()
	if err != nil {
		return []ScheduledJob{}, err
	}

	data, err := os.ReadFile(jobsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []ScheduledJob{}, nil // No jobs yet
		}
		return nil, err
	}
	data = stripUTF8BOM(data)

	var jobs []ScheduledJob
	if err := json.Unmarshal(data, &jobs); err != nil {
		return nil, err
	}
	var pins map[string]string
	if a.config != nil {
		pins = a.config.UpstreamPinnedDisks
	}
	for i := range jobs {
		normalizeJobSchedules(&jobs[i])
		migrateUpstreamDiskPins(&jobs[i], pins)
	}

	return jobs, nil
}

// GetScheduledJobsForAPI returns scheduled jobs as map[string]interface{} for API compatibility
// This method is used by the BackupHandler interface for HTTP API
func (a *App) GetScheduledJobsForAPI() []map[string]interface{} {
	jobs, err := a.GetScheduledJobs()
	if err != nil {
		writeDebugLog(fmt.Sprintf("GetScheduledJobsForAPI error: %v", err))
		return []map[string]interface{}{}
	}

	result := make([]map[string]interface{}, len(jobs))
	for i, job := range jobs {
		result[i] = map[string]interface{}{
			"id":           job.ID,
			"name":         job.Name,
			"backup_type":  job.BackupType,
			"backup_id":    job.BackupID,
			"schedule":     job.ScheduleTime,
			"use_vss":      job.UseVSS,
			"backup_dirs":  job.BackupDirs,
			"exclude_list": job.ExcludeList,
			"last_run":     job.LastRun,
			"next_run":     job.NextRun,
			"enabled":      job.Enabled,
		}
	}
	return result
}

// checkUniqueJobName refuses a Backup Set name another set already has
// (ignoring case and surrounding spaces): the name is how sets are told apart
// in the list, Reports, notifications and the snapshot comment on PBS.
func checkUniqueJobName(jobs []ScheduledJob, job ScheduledJob) error {
	name := strings.TrimSpace(job.Name)
	if name == "" {
		return nil
	}
	for _, j := range jobs {
		if j.ID != job.ID && strings.EqualFold(strings.TrimSpace(j.Name), name) {
			return fmt.Errorf("a Backup Set named %q already exists: each set needs its own name", name)
		}
	}
	return nil
}

// UpdateScheduledJob updates an existing scheduled job
func (a *App) UpdateScheduledJob(job ScheduledJob) error {
	job.BackupDirs = resolveMappedDrives(job.BackupDirs) // as SaveScheduledJob
	if a.delegated() {
		return a.apiClient.Call("UpdateScheduledJob", nil, job)
	}
	if err := a.checkJobsChangeAllowed(); err != nil {
		return err
	}
	writeDebugLog(fmt.Sprintf("UpdateScheduledJob called for: %s", job.Name))
	if err := validateScheduledJob(job); err != nil {
		return err
	}
	normalizeJobSchedules(&job)
	pinJobDisks(&job)

	jobsFileMu.Lock()
	defer jobsFileMu.Unlock()
	// Load existing jobs
	jobs, err := a.GetScheduledJobs()
	if err != nil {
		return fmt.Errorf("failed to load jobs: %w", err)
	}
	if err := checkUniqueJobName(jobs, job); err != nil {
		return err
	}

	// Find and update the job
	found := false
	for i, j := range jobs {
		if j.ID == job.ID {
			// Preserve enabled state and run state
			job.Enabled = j.Enabled
			job.MetadataRunsSinceFullRead = j.MetadataRunsSinceFullRead
			job.FullReadDay = j.FullReadDay
			job.LastFullRead = j.LastFullRead
			// The editor does not send the last run: keep it, or the card
			// says "Never run yet" after every edit.
			if job.LastRun == "" {
				job.LastRun = j.LastRun
			}
			// Recalculate next run with the new schedules; a run waiting
			// for its server is no longer due (dropStaleServerHolds).
			job.WaitingForServer = ""
			setNextRun(&job, time.Now())
			jobs[i] = job
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("job with ID %s not found", job.ID)
	}

	// Save to file
	jobsPath, err := getScheduledJobsPath()
	if err != nil {
		return fmt.Errorf("failed to get jobs path: %w", err)
	}

	data, err := json.MarshalIndent(jobs, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal jobs: %w", err)
	}

	if err := atomicWriteFile(jobsPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write jobs file: %w", err)
	}

	writeDebugLog(fmt.Sprintf("Scheduled job updated: %s (next run: %s)", job.Name, job.NextRun))
	a.wakeTimerAfterSetsChanged()
	return nil
}

// SetScheduledJobEnabled enables or disables a backup set. A disabled set
// keeps its settings and history but the scheduler leaves it alone (Run Now
// still works). Enabling works out the next run from now, so a set that was
// disabled for a while does not fire at once for the runs it missed.
func (a *App) SetScheduledJobEnabled(jobID string, enabled bool) error {
	if a.delegated() {
		return a.apiClient.SetJobEnabled(jobID, enabled)
	}
	if err := a.checkJobsChangeAllowed(); err != nil {
		return err
	}
	jobsFileMu.Lock()
	defer jobsFileMu.Unlock()
	jobs, err := a.GetScheduledJobs()
	if err != nil {
		return fmt.Errorf("failed to load jobs: %w", err)
	}
	found := false
	for i := range jobs {
		if jobs[i].ID == jobID {
			jobs[i].Enabled = enabled
			jobs[i].WaitingForServer = ""
			if enabled {
				setNextRun(&jobs[i], time.Now())
			}
			found = true
			writeDebugLog(fmt.Sprintf("Backup set %s %s (next run: %s)", jobs[i].Name, map[bool]string{true: "enabled", false: "disabled"}[enabled], jobs[i].NextRun))
			break
		}
	}
	if !found {
		return fmt.Errorf("job with ID %s not found", jobID)
	}
	jobsPath, err := getScheduledJobsPath()
	if err != nil {
		return fmt.Errorf("failed to get jobs path: %w", err)
	}
	data, err := json.MarshalIndent(jobs, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal jobs: %w", err)
	}
	if err := atomicWriteFile(jobsPath, data, 0600); err != nil {
		return err
	}
	a.wakeTimerAfterSetsChanged()
	return nil
}

// DeleteScheduledJob removes a scheduled job by ID
func (a *App) DeleteScheduledJob(jobID string) error {
	if a.delegated() {
		return a.apiClient.Call("DeleteScheduledJob", nil, jobID)
	}
	if err := a.checkJobsChangeAllowed(); err != nil {
		return err
	}
	writeDebugLog(fmt.Sprintf("DeleteScheduledJob called for ID: %s", jobID))

	jobsFileMu.Lock()
	defer jobsFileMu.Unlock()
	jobs, err := a.GetScheduledJobs()
	if err != nil {
		return err
	}

	// Filter out the job to delete
	filtered := []ScheduledJob{}
	for _, job := range jobs {
		if job.ID != jobID {
			filtered = append(filtered, job)
		}
	}

	// Save updated list
	jobsPath, err := getScheduledJobsPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(filtered, "", "  ")
	if err != nil {
		return err
	}

	if err := atomicWriteFile(jobsPath, data, 0600); err != nil {
		return err
	}
	a.wakeTimerAfterSetsChanged()
	return nil
}

// RunScheduledJobNow immediately runs the given backup set regardless of its
// TriggerMode or NextRun — the "Run Now" action every backup set has, and the
// ONLY way a "manual" trigger-mode set ever runs. It goes through the exact
// same executeScheduledJob path a real scheduled fire takes, so history and
// LastRun/NextRun bookkeeping behave identically to an automatic run.
func (a *App) RunScheduledJobNow(jobID string) error {
	return a.RunScheduledJobNowRead(jobID, "")
}

// RunScheduledJobNowRead runs a Backup Set now. read only matters for a
// metadata-mode set: "full" reads every file this time (a full read, which
// resets the "full read every N runs" count), "quick" reuses unchanged files
// even when the next run is due a full read (which then stays due), "" does
// what the set's schedule would.
func (a *App) RunScheduledJobNowRead(jobID, read string) error {
	if read != "" && read != "full" && read != "quick" {
		return fmt.Errorf("unknown read choice %q", read)
	}
	// With the service, the set runs there: it records the history and the
	// last run and does the post-backup actions, as for its scheduled runs.
	if a.delegated() {
		return a.apiClient.Call("RunScheduledJobNowRead", nil, jobID, read)
	}
	jobs, err := a.GetScheduledJobs()
	if err != nil {
		return fmt.Errorf("failed to load jobs: %w", err)
	}

	var target *ScheduledJob
	for i := range jobs {
		if jobs[i].ID == jobID {
			target = &jobs[i]
			break
		}
	}
	if target == nil {
		return fmt.Errorf("backup set with ID %s not found", jobID)
	}

	runningJobsMutex.Lock()
	alreadyRunning := runningJobs[target.ID]
	runningJobsMutex.Unlock()
	if alreadyRunning {
		return fmt.Errorf("backup set %q is already running", target.Name)
	}
	if rollbackHeld(target.ID) {
		return fmt.Errorf("backup set %q is being rolled back; run it when the roll back has finished", target.Name)
	}

	writeDebugLog(fmt.Sprintf("RunScheduledJobNow: manually triggering %s (read %q)", target.Name, read))
	run := *target
	run.RequestedRead = read
	go a.executeScheduledJob(run, "manual")
	return nil
}

// GetJobHistory returns job history
func (a *App) GetJobHistory() ([]JobHistory, error) {
	if a.delegated() {
		maps, err := a.apiClient.GetJobHistory()
		if err != nil {
			return []JobHistory{}, err
		}
		history := []JobHistory{}
		if err := remarshal(maps, &history); err != nil {
			return nil, err
		}
		return history, nil
	}
	historyPath, err := getJobHistoryPath()
	if err != nil {
		return []JobHistory{}, err
	}

	data, err := os.ReadFile(historyPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []JobHistory{}, nil
		}
		return nil, err
	}
	data = stripUTF8BOM(data)

	var history []JobHistory
	if err := json.Unmarshal(data, &history); err != nil {
		return nil, err
	}

	return history, nil
}

// useRunSummary puts the run's own summary ("Machine backup completed in
// 5m 23s: 128.0 GB read ..."), with its message code so it is translated,
// on a history entry the scheduler wrote with the generic text, as the app
// writes it without the service (main.go's OnComplete). Found live
// 2026-11-03: every run with the service said only "Backup completed" in
// Reports, the Message Log and the e-mail. A cancelled run keeps its own
// wording, and an outcome that disagrees with the run's result is ignored.
func useRunSummary(entry *JobHistory, outcome *serviceRunOutcome, succeeded bool) {
	if outcome == nil || outcome.message == "" || outcome.success != succeeded || entry.Status == "cancelled" {
		return
	}
	entry.Message = outcome.message
	entry.MessageKey = outcome.key
	entry.MessageParams = outcome.params
	if entry.BackupType != "machine" && entry.ReusedFiles == 0 {
		entry.ReusedFiles = paramUint(outcome.params, "reusedFiles")
		entry.ReusedBytes = paramUint(outcome.params, "reusedBytes")
	}
}

// AddJobHistory adds a job to history. Every backup completion (scheduled or
// interactive one-shot) funnels through here, which makes it the single safe
// place to also feed the Message Log — no need to touch the backup control
// flow itself at each of its several completion sites.
func (a *App) AddJobHistory(entry JobHistory) error {
	if l := entry.logLine; l != nil {
		LogMessage("Backup", l.Level, fmt.Sprintf("%s: %s", entry.Name, l.Message), entry.Name, l.MessageKey, l.MessageParams)
		return a.appendJobHistory(entry)
	}
	level := "info"
	if entry.Status == "failed" {
		level = "error"
	}
	LogMessage("Backup", level, fmt.Sprintf("%s: %s", entry.Name, entry.Message), entry.Name, entry.MessageKey, entry.MessageParams)
	return a.appendJobHistory(entry)
}

// appendJobHistory persists entry at the top of the Reports history without
// touching the Message Log. Restores use it directly because they already
// write their own "Restore" Message Log line.
func (a *App) appendJobHistory(entry JobHistory) error {
	if a.delegated() {
		return a.apiClient.AppendJobHistory(entry)
	}
	historyFileMu.Lock()
	defer historyFileMu.Unlock()
	history, err := a.GetJobHistory()
	if err != nil {
		// Refuse to overwrite the history file with just this entry when the
		// existing history could not be read — that would destroy the record.
		return fmt.Errorf("failed to load job history (refusing to overwrite): %w", err)
	}

	// Add new entry at the beginning
	history = append([]JobHistory{entry}, history...)

	// Keep only last 50 entries
	if len(history) > 50 {
		history = history[:50]
	}

	// Save
	historyPath, err := getJobHistoryPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		return err
	}

	return atomicWriteFile(historyPath, data, 0600)
}

// weekdayAbbrev maps time.Weekday (Sunday=0) to the 3-letter form
// ScheduledJob.DaysOfWeek and the frontend's day checkboxes use.
var weekdayAbbrev = [...]string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}

// dayAllowed reports whether t's weekday is in days. An empty/nil days list
// means every day is allowed — the behavior every job had before DaysOfWeek
// existed, kept as the default so old jobs are unaffected.
func dayAllowed(t time.Time, days []string) bool {
	if len(days) == 0 {
		return true
	}
	abbr := weekdayAbbrev[t.Weekday()]
	for _, d := range days {
		if strings.EqualFold(d, abbr) {
			return true
		}
	}
	return false
}

// parseHHMM parses an "HH:MM" string into its hour/minute components.
func parseHHMM(s string) (hour, min int, err error) {
	if _, err = fmt.Sscanf(s, "%d:%d", &hour, &min); err != nil {
		return 0, 0, err
	}
	return hour, min, nil
}

// calculateNextRunAt calculates a set's next run after now: the earliest time
// any of its enabled schedules gives (schedules.go), "" for a manual set.
// Taking the reference time keeps the scheduling maths testable without the
// wall clock.
func calculateNextRunAt(job ScheduledJob, now time.Time) string {
	next, _ := nextRunFor(job, now)
	return next
}

// RecalculateNextRuns repairs jobs whose nextRun is MISSING or unparseable by
// giving them a valid future run time. It deliberately leaves an overdue but
// valid nextRun in the past: the scheduler now catches missed runs up on its
// next tick (see checkAndRunScheduledJobs). Pushing overdue runs forward here —
// as this used to — silently dropped the run a rebooted/off machine had missed.
func (a *App) RecalculateNextRuns() {
	writeDebugLog("RecalculateNextRuns called - repairing missing/invalid nextRun values")

	jobsFileMu.Lock()
	defer jobsFileMu.Unlock()
	jobs, err := a.GetScheduledJobs()
	if err != nil {
		writeDebugLog(fmt.Sprintf("Error loading scheduled jobs: %v", err))
		return
	}

	modified := false

	for i, job := range jobs {
		if !job.Enabled || job.ScheduleMode == ScheduleModeManual {
			continue
		}

		// A present, parseable nextRun is left alone — even if it is in the past,
		// so the scheduler's catch-up can run the missed backup.
		if job.NextRun != "" {
			if _, perr := time.Parse(time.RFC3339, job.NextRun); perr == nil {
				continue
			}
		}

		newNextRun, by := nextRunFor(job, time.Now())
		writeDebugLog(fmt.Sprintf("[RecalculateNextRuns] Job %s: nextRun was missing/invalid (%q), set to %s",
			job.Name, job.NextRun, newNextRun))
		jobs[i].NextRun, jobs[i].NextRunBy = newNextRun, by
		modified = true
	}

	if modified {
		jobsPath, err := getScheduledJobsPath()
		if err != nil {
			writeDebugLog(fmt.Sprintf("Error getting jobs path: %v", err))
			return
		}

		data, err := json.MarshalIndent(jobs, "", "  ")
		if err != nil {
			writeDebugLog(fmt.Sprintf("Error marshaling jobs: %v", err))
			return
		}

		if err := atomicWriteFile(jobsPath, data, 0600); err != nil {
			writeDebugLog(fmt.Sprintf("Error saving recalculated jobs: %v", err))
		} else {
			writeDebugLog("Successfully recalculated stale nextRun values")
		}
	}
}

// schedulerMu guards starting and stopping the scheduler, which can now
// happen more than once in a GUI session: Run as Service hands scheduling to
// the service and takes it back when the service is removed.
var (
	schedulerMu      sync.Mutex
	schedulerRunning bool
)

// StartScheduler starts the background job scheduler (no-op when running).
func (a *App) StartScheduler() {
	schedulerMu.Lock()
	defer schedulerMu.Unlock()
	if schedulerRunning {
		return
	}
	writeDebugLog("Starting background job scheduler")
	if a.stopScheduler == nil {
		a.stopScheduler = make(chan struct{})
	}
	select {
	case <-a.stopScheduler: // closed by an earlier stop
		a.stopScheduler = make(chan struct{})
	default:
	}
	stop := a.stopScheduler
	schedulerRunning = true

	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		// Only a running scheduler starts the backup a wake is for.
		a.refreshWakeTimer()
		defer dropWakeTimer()
		// The tick before, so a tick after a long gap tells that the computer
		// slept, and woke for a set's run (battery_rule.go).
		lastTick := time.Now()
		// Started with the computer, the network may not be up yet: runs
		// starting soon wait for their server (network_wait.go).
		started := "after the app started"
		if a.isServiceProcess {
			started = "after the service started"
		}
		noteResume(lastTick, started)

		for {
			select {
			case <-ticker.C:
				now := time.Now()
				resumed := noteSchedulerTick(lastTick, now)
				lastTick = now
				a.checkAndRunScheduledJobs(resumed)
				a.RefreshIdleTooltip()
				a.refreshWakeTimer()
			case <-stop:
				writeDebugLog("Scheduler stopped")
				return
			}
		}
	}()
}

// StopScheduler stops the background job scheduler (no-op when stopped).
func (a *App) StopScheduler() {
	schedulerMu.Lock()
	defer schedulerMu.Unlock()
	if !schedulerRunning {
		return
	}
	writeDebugLog("Stopping background job scheduler")
	close(a.stopScheduler)
	schedulerRunning = false
}

// CleanupAbandonedJobs marks any "running" jobs as abandoned on app startup,
// and ties runs an earlier version recorded to their Backup Sets.
func (a *App) CleanupAbandonedJobs() {
	writeDebugLog("CleanupAbandonedJobs called - cleaning up stale running jobs")

	historyFileMu.Lock()
	defer historyFileMu.Unlock()
	history, err := a.GetJobHistory()
	if err != nil {
		writeDebugLog(fmt.Sprintf("Error loading job history: %v", err))
		return
	}

	modified := false
	for i, entry := range history {
		if entry.Status == "running" {
			writeDebugLog(fmt.Sprintf("Marking abandoned job as failed: %s", entry.Name))
			history[i].Status = "failed"
			history[i].Message = "Abandoned (application interrupted)"
			history[i].MessageKey = MsgJobAbandoned
			history[i].MessageParams = nil
			history[i].Timestamp = time.Now().Format(time.RFC3339)
			modified = true
		}
	}

	// Tie runs recorded before entries held their Backup Set's ID to their
	// sets (history_set_id.go). Left for the next start when the sets cannot
	// be read: with no sets to go on every run would be marked as no set's.
	if jobs, jerr := a.GetScheduledJobs(); jerr != nil {
		writeDebugLog(fmt.Sprintf("Run history not tied to Backup Sets this start, the sets could not be read: %v", jerr))
	} else if stampHistorySetIDs(history, jobs) {
		writeDebugLog("Run history: tied earlier runs to their Backup Sets")
		modified = true
	}

	if modified {
		// Save updated history
		historyPath, err := getJobHistoryPath()
		if err != nil {
			writeDebugLog(fmt.Sprintf("Error getting history path: %v", err))
			return
		}

		data, err := json.MarshalIndent(history, "", "  ")
		if err != nil {
			writeDebugLog(fmt.Sprintf("Error marshaling history: %v", err))
			return
		}

		if err := atomicWriteFile(historyPath, data, 0600); err != nil {
			writeDebugLog(fmt.Sprintf("Error saving updated history: %v", err))
		} else {
			writeDebugLog("Successfully cleaned up abandoned jobs")
		}
	}
}

// HandleStartupRun runs the sets whose "When something happens on this
// computer" trigger has Starts ticked (computer_events.go).
func (a *App) HandleStartupRun() {
	writeDebugLog("HandleStartupRun called - checking for startup jobs")

	// Wait a bit to avoid conflict with scheduler if app starts at scheduled time
	time.Sleep(5 * time.Second)

	jobs, err := a.GetScheduledJobs()
	if err != nil {
		writeDebugLog(fmt.Sprintf("Error loading scheduled jobs: %v", err))
		return
	}

	for _, job := range jobs {
		if !computerEventsOn(job) || !job.ComputerEvents.Starts {
			continue
		}
		eventState.Lock()
		eventState.started[job.ID] = time.Now() // a sign-in right after the start starts no second run
		eventState.Unlock()

		// Check if this job is already running (mutex protection)
		// If scheduler already started it, the mutex will prevent duplicate execution
		writeDebugLog(fmt.Sprintf("Executing startup job: %s", job.Name))
		go a.executeScheduledJob(job, "startup")
	}
}

// schedulerTickCount tracks ticks for periodic verbose logging
var schedulerTickCount int

// checkAndRunScheduledJobs checks if any jobs need to run. resumed: the
// computer woke since the last tick (the "wakes" event).
func (a *App) checkAndRunScheduledJobs(resumed bool) {
	jobs, err := a.GetScheduledJobs()
	if err != nil {
		writeDebugLog(fmt.Sprintf("[Scheduler] Error loading scheduled jobs: %v", err))
		return
	}

	schedulerTickCount++
	// Log status every 15 minutes (15 ticks at 1 tick/min) instead of every minute
	verbose := schedulerTickCount%15 == 1
	dropStaleBatteryHolds(jobs, time.Now())

	if len(jobs) == 0 {
		if verbose {
			writeDebugLog("[Scheduler] No scheduled jobs found")
		}
		return
	}

	now := time.Now()
	if verbose {
		writeDebugLog(fmt.Sprintf("[Scheduler] Checking %d jobs at %s", len(jobs), now.Format("15:04:05")))
	}

	for _, job := range jobs {
		if !job.Enabled {
			continue
		}

		// Parse next run time
		if job.NextRun == "" {
			a.retryHeldStartupRun(job)
			continue
		}

		nextRun, err := time.Parse(time.RFC3339, job.NextRun)
		if err != nil {
			writeDebugLog(fmt.Sprintf("[Scheduler] Error parsing next run time for %s: %v", job.Name, err))
			a.retryHeldStartupRun(job)
			continue
		}

		// Fire as soon as we are at or past NextRun, with NO upper bound. The old
		// 2-minute window meant a missed tick (machine asleep/hibernating, heavy
		// load, the process not running at NextRun) left now permanently past the
		// window, so the job never ran again until the next process restart —
		// silent loss of scheduled protection. A late run is caught up instead.
		// Repeated firing is prevented by the runningJobs guard and by
		// executeScheduledJob advancing NextRun on completion.
		shouldRun := now.After(nextRun)

		if verbose {
			writeDebugLog(fmt.Sprintf("[Scheduler] Job %s: NextRun=%s, Now=%s, ShouldRun=%v",
				job.Name, nextRun.Format("15:04:05"), now.Format("15:04:05"), shouldRun))
		}

		if shouldRun {
			// A run held back for the battery is due on every tick until the
			// computer is plugged in: said once a quarter of an hour, not
			// every minute.
			if _, held := batteryHeld(job.ID); (!held && job.WaitingForServer == "") || verbose {
				writeDebugLog(fmt.Sprintf("[Scheduler] Executing scheduled job: %s", job.Name))
			}
			go a.executeScheduledJob(job, "scheduled")
			continue
		}
		a.retryHeldStartupRun(job)
	}
	// "When the server can be reached" (server_trigger.go).
	a.checkServerTriggers(jobs, now)
	// "When something happens on this computer" (computer_events.go).
	a.checkComputerEvents(jobs, now, resumed)
}

// retryHeldStartupRun tries a run at startup held back for the battery
// again, as a due run is on every tick, so it starts when the computer is
// plugged in. (A set's due scheduled run takes the hold instead.)
func (a *App) retryHeldStartupRun(job ScheduledJob) {
	if trigger, held := batteryHeld(job.ID); held && trigger == "startup" {
		go a.executeScheduledJob(job, "startup")
	}
}

// executeScheduledJob executes a scheduled job. trigger records how this run
// started ("scheduled", "startup", or "manual") — passed straight through to
// StartBackup/StartMachineBackup as a real parameter (see app_types.go's
// removal comment for the shared field this used to be).
func (a *App) executeScheduledJob(job ScheduledJob, trigger string) {
	// A roll back of this set is putting its folders back: backing them up
	// half way through would record a mix of old and new.
	if rollbackHeld(job.ID) {
		writeDebugLog(fmt.Sprintf("Job %s skipped: a roll back of this set is running", job.Name))
		return
	}
	// Check if job is already running
	runningJobsMutex.Lock()
	if runningJobs[job.ID] {
		writeDebugLog(fmt.Sprintf("Job %s is already running, skipping", job.Name))
		runningJobsMutex.Unlock()
		return
	}
	runningJobs[job.ID] = true
	runningJobsMutex.Unlock()

	// Ensure we mark as not running when done
	defer func() {
		runningJobsMutex.Lock()
		delete(runningJobs, job.ID)
		runningJobsMutex.Unlock()
	}()

	// The set's "On battery" rule: a scheduled or startup run on battery may
	// be held back until the computer is plugged in (battery_rule.go). Its
	// next run is left as it is, so the run is not lost.
	battery, mayStart := a.batteryGate(job, trigger, time.Now())
	if !mayStart {
		return
	}
	if trigger == "scheduled" {
		if due, err := time.Parse(time.RFC3339, job.NextRun); err == nil {
			battery.dueAt = due
		}
		// The schedule's server and network choices: the run may wait for
		// its server, or be skipped on another network (server_trigger.go).
		if !a.serverGate(job, battery, time.Now()) {
			return
		}
	}
	// Keep the computer awake until the run has started (#142): a computer
	// woken by a timer goes back to sleep within minutes when left alone, and
	// a run waiting for its server, or for its turn, then waited until
	// something next woke the computer. Found 2026-11-17: a 2 minute wait for
	// the server took 93 minutes. The run takes its own hold once it starts
	// (RunBackupInline); this one ends when the run has been started, and
	// before anything this run does afterwards (put the computer to sleep).
	endAwake, awakeErr := preventSleep(sleepReason())
	defer endAwake()
	if awakeErr != nil {
		writeDebugLog(fmt.Sprintf("Backup Set %s: %v", job.Name, awakeErr))
	}
	// This hold takes over from the one the wake took for this set's run
	// (wake_hold.go).
	releaseWakeHold(job.ID)

	// Just after the computer woke or this program started, the network may
	// not be back yet: an automatic run waits for its server first, here in
	// its own goroutine (network_wait.go). A server still out of reach then:
	// the run is not started.
	waited, werr := a.waitForServerAfterWake(job, trigger)
	if werr != nil {
		writeDebugLog(fmt.Sprintf("Backup Set %s: %v", job.Name, werr))
		endAwake()
		a.recordRunNotStarted(job, trigger, battery, werr)
		a.recordJobRan(job)
		return
	}
	if waited >= wokeWaitNoted && battery != nil {
		secs := int(waited.Round(time.Second) / time.Second)
		battery.notes = append(battery.notes, HistoryNote{
			Key: NoteWokeWaited, Params: msgParams{"secs": secs},
			Text: fmt.Sprintf("Waited %d s for the server after the computer woke or started.", secs),
		})
	}

	writeDebugLog(fmt.Sprintf("Executing scheduled job: %s", job.Name))

	// Prepare history entry (will be added at the end with final status)
	startTime := time.Now()

	// Use StartBackup to route through mode detection (service or direct)
	writeDebugLog(fmt.Sprintf("[Scheduled Job] Executing via StartBackup (mode: %s)", a.mode.String()))

	// The run takes the Backup Options compression setting: a Backup Set's
	// own "compression" (never shown in the editor, and "fastest" in sets
	// saved by older builds) is not read any more.
	compression := ""

	// driveLetters carries the physical disks for machine backups (empty for
	// directory backups); it is persisted on the job so a scheduled run restores
	// the exact same selection.
	driveLetters := job.DriveLetters
	if len(driveLetters) == 0 {
		driveLetters = []string{}
	}

	// Route to the right top-level entry point by the job's own type — a
	// machine (whole-disk) backup set must call StartMachineBackup, not
	// StartBackup. Found 2026-09-23: this always called StartBackup
	// regardless of type, which for a machine job passed \\.\PhysicalDriveN
	// device paths into the DIRECTORY backup pipeline (RunBackupInline
	// decides machine vs. directory via BackupOptions.Kind, which
	// startBackupDirect never sets) — silently misrouted into treating a
	// raw device path as a folder to walk, panicking almost immediately
	// ("slice bounds out of range") rather than doing an actual disk
	// backup. Only the frontend's own one-off machine backup button ever
	// called StartMachineBackup correctly; every SCHEDULED machine backup
	// set has never actually worked.
	// The run's name (history, Reports, the PBS comment) is job.Name, passed
	// as the comment below.

	runPreBackupApp(job)

	// This run's change detection (metadata sets read every file on every
	// FullReadEvery-th run). The job copy carries the decision to the run
	// through registerPendingPostActions; the counter moves only when the run
	// completes (recordChangeDetectionOutcome), so a cancelled or failed run
	// neither counts towards the next full read nor stands in for one.
	runMode, _ := changeDetectionForRun(job)
	runMode, job.ScheduledFullRead, job.FullReadRequested = applyRequestedRead(job, runMode)
	if job.ScheduledFullRead && !a.claimFullReadDay(job.ID, time.Now()) {
		// Another set has today's full read: this one reads only changed
		// files and is due again on its next run.
		runMode, job.ScheduledFullRead = pbscommon.ChangeDetectionMetadata, false
		writeDebugLog(fmt.Sprintf("Backup Set %s: full read moved to the next run (another set has today's)", job.Name))
	}
	if job.ScheduledFullRead {
		writeDebugLog(fmt.Sprintf("Backup Set %s: full read run (every %d runs)", job.Name, job.FullReadEvery))
	} else if job.FullReadRequested {
		writeDebugLog(fmt.Sprintf("Backup Set %s: full read requested (Run now)", job.Name))
	}
	job.ChangeDetectionMode = runMode

	// job.Name passed directly, and postActions/trigger registered under a
	// fresh key (app_types.go) rather than a shared field — NOT the generic
	// fallback lookup a shared field would need. Found live 2026-09-27
	// testing the queue feature: clicking several Run Now buttons in quick
	// succession spawns that many concurrent executeScheduledJob goroutines
	// (RunScheduledJobNow fires each via `go`), and a single shared value has
	// no atomicity across its own set-then-read sequence — goroutine A's own
	// set can land in between goroutine B's set and read, so B's
	// history/comment/post-actions end up using A's values (or the generic
	// fallback, if a clear lands there instead). Fixed for the name
	// 2026-09-27; postActions/trigger used to be the exact same shared-field
	// pattern (currentScheduledJobPostActions/currentScheduledJobTrigger,
	// removed) but had no one-off use case at all, so a per-run key replaces
	// them outright instead of just moving where a shared value gets read.
	postActionsKey := a.registerPendingPostActions(job, trigger)
	// What the battery did, for the history entry; and its cancel level,
	// checked while the run goes on. This function lasts as long as the run
	// (it waits for it below), so the watch ends with it.
	a.setRunBattery(postActionsKey, battery)
	stopBatteryWatch := a.watchBatteryCancel(job, battery)
	defer stopBatteryWatch()
	// A run this app does itself goes on in a goroutine (startBackupDirect)
	// and writes its own history and post-backup actions; this closes when
	// it has ended. Taken now: the run drops its entry when it completes.
	localDone := a.localRunDone(postActionsKey)

	// In the service, the run gets an entry on the local API like a run a
	// GUI asked for, so an open window shows it and can pause or stop it.
	runID := ""
	if a.isServiceProcess && a.runReporter != nil {
		runID = a.runReporter.BeginRun(job.Name)
		if v, ok := a.pendingPostActions.Load(postActionsKey); ok {
			v.(*pendingPostActionsEntry).runID = runID
		}
	}

	var err error
	if job.BackupType == "machine" {
		err = a.StartMachineBackup(
			scheduledMachineBackupType(job),
			driveLetters,
			job.BackupID,
			job.UseVSS,
			compression,
			job.PBSServerID,
			job.Name,
			postActionsKey,
		)
	} else {
		err = a.StartBackup(
			job.BackupType,
			job.BackupDirs,
			driveLetters,
			job.ExcludeList,
			job.BackupID,
			job.UseVSS,
			compression,
			job.PBSServerID,
			job.Name,
			postActionsKey,
		)
	}
	// Started (or failed to): the run holds the computer awake itself while
	// it goes on, and what this set does afterwards must not find this hold
	// still standing (a Linux sleep inhibitor blocks the sleep it asks for).
	endAwake()
	// The app handed the run to the background service: the call above only
	// started it. Wait for its real outcome, so the history entry and the
	// post-backup actions (an e-mail saying it completed, the "after"
	// application, a shutdown) come when it has finished, not at once.
	// Found live 2026-10-17: a Run Now in service mode wrote "success" and
	// ran them at the start of a 5 minute machine backup.
	noter := a.runNoter(postActionsKey)
	// How the run ended in its own words, for Reports: from the service's
	// API entry here, or from the backup's completion in the service.
	outcome := a.runOutcome(postActionsKey)
	ranInService := false
	if err == nil && !a.isServiceProcess {
		if ch := a.serviceRunDone(postActionsKey); ch != nil {
			writeDebugLog(fmt.Sprintf("Backup Set %s: waiting for the service to finish the run", job.Name))
			o := <-ch
			outcome = &o
			ranInService = true
			if !o.success {
				err = fmt.Errorf("%s", o.message)
			} else {
				// The snapshot is the service's: find it on the server if
				// a failed e-mail has to be noted on it.
				noter = a.newGroupNoter(job, startTime)
			}
		} else if localDone != nil && a.mode != api.ModeService {
			// Hold the set as running until its run has ended, queue wait
			// included: with several backups at once a set waiting behind
			// its own earlier run must not be queued again by the next tick,
			// and Last run says when the run finished.
			<-localDone
		}
	}
	// A run that failed here before the backup started (its login refused, a
	// disk or the key not found) has no log of its own: it gets one, so its
	// "View log" does not open the run before it (run_log_not_started.go).
	// A run the service did keeps its logs there.
	if err != nil && !ranInService && !a.delegated() && !isCancelledRun(err.Error(), "") &&
		!runLogStartedSince(runLogIDFor(job.BackupID), startTime) {
		writeNotStartedRunLog(job.BackupID, job.Name, trigger, err)
	}
	// Close the API entry if the backup ended before its own completion
	// reported (a failure before it started); otherwise this changes nothing.
	if runID != "" {
		if err != nil {
			a.finishRun(runID, false, fmt.Sprintf("Backup failed: %v", err))
		} else {
			a.finishRun(runID, true, "Backup completed")
		}
	}

	// Add history entry derived from the REAL outcome, service mode only.
	// In service mode StartBackup runs synchronously (app_service_stubs.go
	// returns RunBackupInline's error), so err here genuinely reflects the
	// finished backup: nil is success, non-nil is partial/failed, and this
	// is the only history write that exists at all for service mode
	// (main.go's OnComplete-based writer above isn't even compiled into
	// service builds).
	//
	// In GUI-standalone mode, StartBackup is fire-and-forget (the backup
	// runs in a goroutine and its error is not awaited here), so err is
	// always nil at this point regardless of the real outcome, meaning a
	// history entry written here would always claim "success" whether or
	// not that were true. Found live 2026-09-24: this produced a second,
	// always-"success" entry for every standalone scheduled run, a few
	// seconds before the accurate one startBackupDirect's OnComplete wrote
	// once the backup actually finished (now correctly using this job's
	// real name too, via currentScheduledJobName), giving two rows in
	// Reports for one real run. Skip entirely in standalone mode; that
	// OnComplete write is now the single source of truth there.
	// The service process runs jobs synchronously too, but its App is in
	// ModeStandalone (service.go), so test for the process, not just the mode.
	if a.mode == api.ModeService || a.isServiceProcess {
		historyEntry := JobHistory{
			ID:         fmt.Sprintf("%d", startTime.Unix()),
			Name:       job.Name,
			JobID:      job.ID,
			Timestamp:  time.Now().Format(time.RFC3339),
			Status:     "success",
			Message:    "Backup completed",
			BackupDirs: job.BackupDirs,
			BackupID:   job.BackupID,
			UseVSS:     job.UseVSS,
			BackupType: job.BackupType,
			Trigger:    trigger,
			MessageKey: MsgBackupCompletedGeneric,
		}
		readEveryFile := false
		if job.BackupType != "machine" {
			historyEntry.ChangeDetection = changeDetectionLabel(job.ChangeDetectionMode)
			if r := a.takeRunResult(postActionsKey); r != nil {
				historyEntry.ReusedFiles, historyEntry.ReusedBytes = r.ReusedFiles, r.ReusedFileBytes
				readEveryFile = r.ReadEveryFile
			}
		}
		if job.BackupType != "machine" {
			a.recordChangeDetectionOutcome(job.ID, err == nil, readEveryFile)
		}

		if err != nil && isCancelledRun(err.Error(), "") {
			historyEntry.Status = "cancelled"
			historyEntry.Message = "Backup cancelled by user"
			historyEntry.MessageKey = MsgBackupCancelled
		} else if err != nil {
			writeDebugLog(fmt.Sprintf("Scheduled job error: %v", err))
			historyEntry.Status = "failed"
			historyEntry.Message = fmt.Sprintf("Error: %v", err)
			historyEntry.MessageKey = MsgScheduledJobError
			historyEntry.MessageParams = msgParams{"error": err.Error()}
		} else if pbsCfg, perr := a.resolvePBS(job.PBSServerID); perr == nil {
			// A finished run means the key (if any) unlocked, so the
			// configured key file is what the snapshot was written with.
			historyEntry.Encryption = encryptionLabel(pbsCfg.EncryptionKeyFile != "")
		}
		useRunSummary(&historyEntry, outcome, err == nil)
		a.applyBatteryToHistory(&historyEntry, battery)

		if err := a.AddJobHistory(historyEntry); err != nil {
			writeDebugLog(fmt.Sprintf("Warning: Failed to add job history: %v", err))
		}
		// Service mode: this synchronous err is the real, final outcome (see
		// the comment above), so this is the one place to fire post-backup
		// actions for it — startBackupDirect's OnComplete (main.go) isn't
		// even compiled into service builds, so it will never consume the
		// postActionsKey registered above; discard it here instead so the
		// pendingPostActions map entry doesn't leak for this run's lifetime
		// (`job` is this function's own local variable throughout, so the
		// discarded value itself isn't needed — only the cleanup is).
		a.runPostBackupActions(job, err == nil, historyEntry.Message, noter, battery)
		a.takePendingPostActions(postActionsKey)
	} else if err != nil {
		// Standalone mode: err here is only ever non-nil for an immediate,
		// synchronous failure (e.g. StartBackup rejecting bad parameters
		// before ever launching the async goroutine). A real mid-backup
		// failure is still only visible via the OnComplete-based write. Log
		// it so it's not silently lost, but don't write a competing history
		// entry for it. Post-actions still fire here for this specific
		// immediate-failure case, since OnComplete never runs at all when
		// the backup never actually started — same leak-prevention reasoning
		// as the service-mode branch above for the takePendingPostActions call.
		writeDebugLog(fmt.Sprintf("Scheduled job %s failed to start: %v", job.Name, err))
		// OnComplete never runs for a run that never started, so without this
		// the failure (e.g. a passphrase-protected key with no stored passphrase)
		// would only ever reach the debug log.
		if herr := a.AddJobHistory(JobHistory{
			ID:            fmt.Sprintf("%d", startTime.Unix()),
			Name:          job.Name,
			JobID:         job.ID,
			Timestamp:     time.Now().Format(time.RFC3339),
			Status:        "failed",
			Message:       fmt.Sprintf("Error: %v", err),
			BackupDirs:    job.BackupDirs,
			BackupID:      job.BackupID,
			UseVSS:        job.UseVSS,
			BackupType:    job.BackupType,
			Trigger:       trigger,
			MessageKey:    MsgScheduledJobError,
			MessageParams: msgParams{"error": err.Error()},
		}); herr != nil {
			writeDebugLog(fmt.Sprintf("Warning: Failed to add job history: %v", herr))
		}
		a.runPostBackupActions(job, false, err.Error(), nil, battery)
		a.takePendingPostActions(postActionsKey)
	}

	a.recordJobRan(job)
}

// recordRunNotStarted records a run of job that was not started because its
// PBS server could not be reached after the computer woke (network_wait.go):
// its own log, a failed Reports entry, and what the set does after a run (an
// e-mail, putting a computer woken for it back to sleep). The set's "before"
// application has not run: there was no backup for it to come before.
func (a *App) recordRunNotStarted(job ScheduledJob, trigger string, battery *batteryRunInfo, cause error) {
	writeNotStartedRunLog(job.BackupID, job.Name, trigger, cause)
	entry := JobHistory{
		ID:            fmt.Sprintf("%d", time.Now().Unix()),
		Name:          job.Name,
		JobID:         job.ID,
		Timestamp:     time.Now().Format(time.RFC3339),
		Status:        "failed",
		Message:       fmt.Sprintf("Error: %v", cause),
		BackupDirs:    job.BackupDirs,
		BackupID:      job.BackupID,
		UseVSS:        job.UseVSS,
		BackupType:    job.BackupType,
		Trigger:       trigger,
		MessageKey:    MsgScheduledJobError,
		MessageParams: msgParams{"error": cause.Error()},
	}
	a.applyBatteryToHistory(&entry, battery)
	if err := a.AddJobHistory(entry); err != nil {
		writeDebugLog(fmt.Sprintf("Warning: Failed to add job history: %v", err))
	}
	a.runPostBackupActions(job, false, entry.Message, nil, battery)
}

// recordJobRan sets job's last run to now and works out its next run, once a
// run of it has ended, however it ended.
func (a *App) recordJobRan(job ScheduledJob) {
	// Re-read from file to pick up any changes made by the GUI while backup was running.
	// Never in an app working through the service: its sets are the
	// service's, which records their runs itself.
	if a.delegated() {
		return
	}
	jobsFileMu.Lock()
	defer jobsFileMu.Unlock()
	jobs, err := a.GetScheduledJobs()
	if err != nil {
		// Never rewrite the jobs file from a failed read — it would wipe every job.
		writeDebugLog(fmt.Sprintf("Warning: skipping LastRun/NextRun update, could not load jobs: %v", err))
		return
	}
	found := false
	for i, j := range jobs {
		if j.ID == job.ID {
			jobs[i].LastRun = time.Now().Format(time.RFC3339)
			jobs[i].WaitingForServer = "" // any run ends a wait for the server
			setNextRun(&jobs[i], time.Now())
			found = true
			break
		}
	}
	if !found {
		// Job was deleted while running; nothing to update.
		return
	}

	// Save updated jobs
	jobsPath, err := getScheduledJobsPath()
	if err != nil {
		writeDebugLog(fmt.Sprintf("Warning: cannot resolve jobs path: %v", err))
		return
	}
	data, err := json.MarshalIndent(jobs, "", "  ")
	if err != nil {
		writeDebugLog(fmt.Sprintf("Warning: cannot marshal jobs: %v", err))
		return
	}
	if err := atomicWriteFile(jobsPath, data, 0600); err != nil {
		writeDebugLog(fmt.Sprintf("Warning: Failed to save updated jobs: %v", err))
	}
}
