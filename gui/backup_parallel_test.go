package main

import (
	"context"
	"strings"
	"testing"
)

// A batch cancelled before any folder started reports a failure naming every
// skipped folder, and never contacts PBS (no server is configured here).
func TestRunFoldersParallelCancelledBeforeStart(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	opts := BackupOptions{
		Ctx:           ctx,
		BaseURL:       "https://pbs.invalid:8007",
		Datastore:     "test-parallel",
		BackupObjects: []string{"/data/a", "/data/b", "/data/c"},
	}
	agg := &BackupStatus{Outcome: OutcomeVerifiedSuccess}

	errs := runFoldersParallel(opts, "host", 2, agg)

	if len(errs) != 1 || !strings.Contains(errs[0], "3/3 folder(s) not backed up") {
		t.Fatalf("errors = %q, want one cancellation error for 3/3 folders", errs)
	}
	if agg.Outcome != OutcomeFailed {
		t.Errorf("outcome = %v, want %v", agg.Outcome, OutcomeFailed)
	}
}

func TestRecommendedParallelAtLeastOne(t *testing.T) {
	if n := RecommendedParallel(); n < 1 {
		t.Errorf("RecommendedParallel() = %d, want >= 1", n)
	}
}
