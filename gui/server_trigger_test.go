package main

import (
	"testing"
	"time"
)

// fakeNetworks stands in for the system's network list.
func fakeNetworks(t *testing.T, info NetworkInfo) {
	t.Helper()
	old := readNetworkInfo
	readNetworkInfo = func() NetworkInfo { return info }
	resetNetInfoCache()
	t.Cleanup(func() {
		readNetworkInfo = old
		resetNetInfoCache()
	})
}

func resetNetInfoCache() {
	netInfoCache.Lock()
	netInfoCache.at = time.Time{}
	netInfoCache.Unlock()
}

func homeWifi() NetworkInfo {
	return NetworkInfo{Supported: true, MeteredKnown: true, Current: []NetConn{{Name: "Home", Kind: connWifi}}}
}

// waitingSet is a set with one daily schedule at 02:00 that waits for its
// server, due now, saved in the (temporary) settings folder.
func waitingSet(t *testing.T, s Schedule, due time.Time) ScheduledJob {
	t.Helper()
	s.ID, s.Enabled, s.Unit, s.Every, s.At = "s1", true, "days", 1, "02:00"
	job := batterySet("", 0, 0)
	job.BackupID = "docs-laptop"
	job.Schedules = []Schedule{s}
	job.NextRun, job.NextRunBy = due.Format(time.RFC3339), "s1"
	if err := writeJobs([]ScheduledJob{job}); err != nil {
		t.Fatal(err)
	}
	return job
}

func savedSet(t *testing.T, a *App) ScheduledJob {
	t.Helper()
	jobs, err := a.GetScheduledJobs()
	if err != nil || len(jobs) != 1 {
		t.Fatalf("sets %v, %v", jobs, err)
	}
	return jobs[0]
}

// A run whose server cannot be reached at its time waits, recorded once; it
// starts when the server answers, and says it waited.
func TestServerWaitHoldsTheRunUntilTheServerAnswers(t *testing.T) {
	newBatteryTest(t)
	due := time.Date(2026, 11, 20, 2, 0, 0, 0, time.Local)
	f := newFakeNet(t, due, 1000)
	a := appWithServer()
	waitingSet(t, Schedule{IfUnreachable: ifUnreachableWait, WaitHours: 12}, due)

	if a.serverGate(savedSet(t, a), &batteryRunInfo{}, due.Add(30*time.Second)) {
		t.Fatal("the run started with its server out of reach")
	}
	job := savedSet(t, a)
	if job.WaitingForServer == "" {
		t.Fatal("the set does not show it is waiting for the server")
	}
	if a.serverGate(job, &batteryRunInfo{}, due.Add(90*time.Second)) {
		t.Fatal("the run started on the next tick, the server still out of reach")
	}
	h := historyOf(t, a)
	if len(h) != 1 || h[0].Status != "held" || h[0].MessageKey != MsgServerWaiting {
		t.Fatalf("want one waiting entry, got %+v", h)
	}

	f.failures = len(f.dials) // the next try answers
	info := &batteryRunInfo{}
	if !a.serverGate(savedSet(t, a), info, due.Add(3*time.Hour)) {
		t.Fatal("the server answered, but the run did not start")
	}
	if len(info.notes) != 1 || info.notes[0].Key != NoteServerWaited {
		t.Fatalf("the run does not say it waited for the server: %+v", info.notes)
	}
	if savedSet(t, a).WaitingForServer != "" {
		t.Error("the set still shows it is waiting")
	}
}

// A run still without its server when the wait runs out is recorded as failed
// then, and the set's next run is worked out.
func TestServerWaitRunsOutAndTheRunFails(t *testing.T) {
	newBatteryTest(t)
	due := time.Date(2026, 11, 20, 2, 0, 0, 0, time.Local)
	newFakeNet(t, due, 1000)
	a := appWithServer()
	waitingSet(t, Schedule{IfUnreachable: ifUnreachableWait, WaitHours: 2}, due)

	a.serverGate(savedSet(t, a), &batteryRunInfo{}, due.Add(time.Minute))
	if a.serverGate(savedSet(t, a), &batteryRunInfo{}, due.Add(2*time.Hour+time.Minute)) {
		t.Fatal("the run started after its wait ran out")
	}
	h := historyOf(t, a)
	if len(h) != 2 || h[0].Status != "failed" || h[0].MessageKey != MsgServerGaveUp {
		t.Fatalf("want a failed entry after the waiting one, got %+v", h)
	}
	job := savedSet(t, a)
	if job.WaitingForServer != "" {
		t.Error("the set still shows it is waiting")
	}
	if next, _ := time.Parse(time.RFC3339, job.NextRun); !next.After(due) {
		t.Errorf("next run %s was not moved on", job.NextRun)
	}
}

// A schedule limited to some networks skips a run on another one, and a run
// with no choices goes ahead without trying the server.
func TestServerNetworkChoiceSkipsTheRun(t *testing.T) {
	newBatteryTest(t)
	due := time.Date(2026, 11, 20, 2, 0, 0, 0, time.Local)
	f := newFakeNet(t, due, 0)
	fakeNetworks(t, NetworkInfo{Supported: true, Current: []NetConn{{Name: "Hotel", Kind: connWifi}}})
	a := appWithServer()

	plain := waitingSet(t, Schedule{}, due)
	if !a.serverGate(plain, &batteryRunInfo{}, due) || len(f.dials) != 0 {
		t.Fatalf("a schedule with no server or network choice was held (%d tries)", len(f.dials))
	}

	waitingSet(t, Schedule{Networks: []string{"Home"}}, due)
	if a.serverGate(savedSet(t, a), &batteryRunInfo{}, due) {
		t.Fatal("the run started on a network the schedule does not run on")
	}
	h := historyOf(t, a)
	if len(h) != 1 || h[0].Status != "skipped" || h[0].MessageKey != MsgNetworkSkipped {
		t.Fatalf("want one skipped entry, got %+v", h)
	}
	if next, _ := time.Parse(time.RFC3339, savedSet(t, a).NextRun); !next.After(due) {
		t.Error("the skipped run's next run was not moved on")
	}
}

func TestNetworkAllows(t *testing.T) {
	metered := homeWifi()
	metered.Metered = true
	cases := []struct {
		name        string
		info        NetworkInfo
		nets, conns []string
		notMetered  bool
		ok          bool
		why         string
	}{
		{"no choices", homeWifi(), nil, nil, false, true, ""},
		{"on a chosen network, any case", homeWifi(), []string{"home"}, nil, false, true, ""},
		{"on another network", homeWifi(), []string{"Office"}, nil, false, false, netBlockNetwork},
		{"a chosen kind", homeWifi(), nil, []string{connWired, connWifi}, false, true, ""},
		{"not a chosen kind", homeWifi(), nil, []string{connWired}, false, false, netBlockConnection},
		{"metered", metered, nil, nil, true, false, netBlockMetered},
		{"metered allowed", metered, nil, nil, false, true, ""},
		{"networks not known", NetworkInfo{}, []string{"Home"}, []string{connWired}, true, true, ""},
	}
	for _, c := range cases {
		ok, why := networkAllows(c.info, c.nets, c.conns, c.notMetered)
		if ok != c.ok || why != c.why {
			t.Errorf("%s: got %v %q, want %v %q", c.name, ok, why, c.ok, c.why)
		}
	}
}

func TestServerTriggerCountsFromTheLastGoodBackup(t *testing.T) {
	good := time.Date(2026, 11, 19, 7, 40, 0, 0, time.Local)
	history := []JobHistory{ // newest first
		{JobID: "docs", Status: "failed", Timestamp: good.Add(5 * time.Hour).Format(time.RFC3339)},
		{JobID: "other", Status: "success", Timestamp: good.Add(4 * time.Hour).Format(time.RFC3339)},
		{JobID: "docs", Status: "success", Kind: "restore", Timestamp: good.Add(3 * time.Hour).Format(time.RFC3339)},
		{JobID: "docs", Status: "success", Timestamp: good.Format(time.RFC3339)},
	}
	if got := lastGoodRun(history, "docs"); !got.Equal(good) {
		t.Errorf("last good run %s, want %s", got, good)
	}
	if !lastGoodRun(history, "new").IsZero() {
		t.Error("a set with no runs has a last good run")
	}
	if d := (&ServerTrigger{}).every(); d != 24*time.Hour {
		t.Errorf("default every %s, want 24h", d)
	}
}

func TestTriggerChoicesAreValidated(t *testing.T) {
	ok := ScheduledJob{ScheduleMode: "scheduled", ServerTrigger: &ServerTrigger{Enabled: true, EveryHours: 24, Connections: []string{connWifi}},
		Schedules: []Schedule{{ID: "s1", Enabled: true, Unit: "days", Every: 1, At: "02:00", IfUnreachable: ifUnreachableWait, WaitHours: 12}}}
	if err := validateSchedules(ok); err != nil {
		t.Fatalf("a good set was refused: %v", err)
	}
	for name, bad := range map[string]func(*ScheduledJob){
		"unknown connection": func(j *ScheduledJob) { j.ServerTrigger = &ServerTrigger{Connections: []string{"carrier pigeon"}} },
		"every too long":     func(j *ScheduledJob) { j.ServerTrigger = &ServerTrigger{EveryHours: maxServerEveryHours + 1} },
		"unknown choice":     func(j *ScheduledJob) { j.Schedules[0].IfUnreachable = "panic" },
		"wait too long":      func(j *ScheduledJob) { j.Schedules[0].WaitHours = maxWaitHours + 1 },
	} {
		j := ok
		j.Schedules = append([]Schedule(nil), ok.Schedules...)
		bad(&j)
		if err := validateSchedules(j); err == nil {
			t.Errorf("%s: saved without an error", name)
		}
	}
}
