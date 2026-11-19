package main

import (
	"encoding/xml"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// What woke the computer, as the system recorded it. On Windows each wake
// from sleep, hibernate or a Fast Startup shutdown writes System event 1 of
// Microsoft-Windows-Power-Troubleshooter, a few seconds after the wake: its
// WakeSourceType is 6 for a wake timer, with the timer's program in
// WakeTimerOwner, 1 for the power button, 0 when Windows could not tell. A
// start in the wake timer's window is the timer's only if that record says
// so: found on a laptop 2026-11-19, switched on with the power button a few
// minutes after its wake timer was due (it could not wake from a Fast Startup
// shutdown), it was taken as woken for the set, so an "only if this backup
// woke the computer" sleep afterwards would have put it to sleep under the
// person who had just switched it on.

// wakeEvent is one record of a wake.
type wakeEvent struct {
	WakeTime   time.Time
	SourceType int
	SourceText string // the timer's program, "salvanza-svc.exe"
	TimerOwner string // its full path, "\Device\HarddiskVolume2\...\salvanza-svc.exe"
}

// errNoWakeRecord: this system keeps no record of what woke it.
var errNoWakeRecord = errors.New("no record of what woke the computer on this system")

// latestWakeEvent reads the newest record of a wake (wake_source_windows.go);
// tests replace it.
var latestWakeEvent = platformLatestWakeEvent

// wakeSourceWait is how long to look for the record of a wake: Windows writes
// it a few seconds after the computer wakes. wakeSourceEvery is how often.
var (
	wakeSourceWait  = 20 * time.Second
	wakeSourceEvery = time.Second
	wakeSourceNow   = time.Now
	wakeSourceSleep = time.Sleep
)

// wakeSourceTimer is WakeSourceType for a wake timer.
const wakeSourceTimer = 6

// wakeVerdict is what the record says about a wake.
type wakeVerdict struct {
	known bool   // a record of this wake was found
	ours  bool   // this program's wake timer woke the computer
	what  string // what woke it, for the log
}

// whatWokeTheComputer finds the record of the wake after asleepSince (the
// last tick before the computer slept) and says whether this program's timer
// woke it. Not known when the system keeps no such record, or none turned up
// within wakeSourceWait: the wake is then taken as the timer's, as before.
func whatWokeTheComputer(asleepSince time.Time) wakeVerdict {
	deadline := wakeSourceNow().Add(wakeSourceWait)
	for {
		ev, err := latestWakeEvent()
		if errors.Is(err, errNoWakeRecord) {
			return wakeVerdict{}
		}
		if err == nil && ev.WakeTime.After(asleepSince) {
			return judgeWake(ev, ownExeName())
		}
		if !wakeSourceNow().Before(deadline) {
			return wakeVerdict{}
		}
		wakeSourceSleep(wakeSourceEvery)
	}
}

// judgeWake: the wake is ours when a timer woke the computer and the timer's
// program is this one (exe, compared without case: "salvanza-svc.exe").
func judgeWake(ev wakeEvent, exe string) wakeVerdict {
	owner := baseName(ev.TimerOwner)
	if owner == "" {
		owner = baseName(ev.SourceText)
	}
	ours := ev.SourceType == wakeSourceTimer && exe != "" && strings.EqualFold(owner, exe)
	return wakeVerdict{known: true, ours: ours, what: describeWakeSource(ev)}
}

func describeWakeSource(ev wakeEvent) string {
	switch {
	case ev.SourceType == wakeSourceTimer && ev.SourceText != "":
		return "a wake timer of " + ev.SourceText
	case ev.SourceType == wakeSourceTimer:
		return "a wake timer"
	case ev.SourceType == 1:
		return "the power button"
	case ev.SourceType == 0:
		return "something Windows did not record (the power button, after a Fast Startup shutdown or hibernate)"
	case ev.SourceText != "":
		return fmt.Sprintf("%s (wake source type %d)", ev.SourceText, ev.SourceType)
	}
	return fmt.Sprintf("wake source type %d", ev.SourceType)
}

// baseName is the last part of a Windows or Unix path, on any system.
func baseName(p string) string {
	p = strings.TrimSpace(p)
	if i := strings.LastIndexAny(p, `\/`); i >= 0 {
		p = p[i+1:]
	}
	return p
}

var (
	ownExeOnce sync.Once
	ownExe     string
)

// ownExeName is this program's file name ("salvanza-svc.exe"), whatever its
// brand calls it.
func ownExeName() string {
	ownExeOnce.Do(func() {
		if p, err := os.Executable(); err == nil {
			ownExe = baseName(p)
		}
	})
	return ownExe
}

// parseWakeEventXML reads a Power-Troubleshooter event 1 as Windows renders
// it to XML (EvtRender, or wevtutil /f:xml).
func parseWakeEventXML(data []byte) (wakeEvent, error) {
	var doc struct {
		Data []struct {
			Name  string `xml:"Name,attr"`
			Value string `xml:",chardata"`
		} `xml:"EventData>Data"`
	}
	if err := xml.Unmarshal(data, &doc); err != nil {
		return wakeEvent{}, fmt.Errorf("reading the wake event: %w", err)
	}
	var ev wakeEvent
	seen := false
	for _, d := range doc.Data {
		v := strings.TrimSpace(d.Value)
		switch d.Name {
		case "WakeTime":
			t, err := time.Parse(time.RFC3339Nano, v)
			if err != nil {
				return wakeEvent{}, fmt.Errorf("the wake event's time %q: %w", v, err)
			}
			ev.WakeTime, seen = t, true
		case "WakeSourceType":
			n, err := strconv.Atoi(v)
			if err != nil {
				return wakeEvent{}, fmt.Errorf("the wake event's source type %q: %w", v, err)
			}
			ev.SourceType = n
		case "WakeSourceText":
			ev.SourceText = v
		case "WakeTimerOwner":
			ev.TimerOwner = v
		}
	}
	if !seen {
		return wakeEvent{}, errors.New("the wake event has no wake time")
	}
	return ev, nil
}
