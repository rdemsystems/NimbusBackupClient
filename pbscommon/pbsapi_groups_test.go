package pbscommon

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// ListGroups reads the groups endpoint of the datastore, in the namespace,
// and keeps each group's comment (the bare metal restore wizard shows it next
// to the group, for example "vm/9000198  Deepthought").
func TestListGroupsReadsComments(t *testing.T) {
	var path, ns string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		ns = r.URL.Query().Get("ns")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[` +
			`{"backup-type":"vm","backup-id":"9000198","comment":"Deepthought","backup-count":3},` +
			`{"backup-type":"host","backup-id":"laptop"}]}`))
	}))
	defer srv.Close()

	pbs := newTestClient(srv.URL, nil)
	groups, err := pbs.ListGroups()
	if err != nil {
		t.Fatalf("ListGroups: %v", err)
	}
	if want := "/api2/json/admin/datastore/teststore/groups"; path != want {
		t.Errorf("request path = %q, want %q", path, want)
	}
	if ns != "testns" {
		t.Errorf("ns query parameter = %q, want %q", ns, "testns")
	}
	want := []BackupGroup{
		{BackupType: "vm", BackupID: "9000198", Comment: "Deepthought"},
		{BackupType: "host", BackupID: "laptop"},
	}
	if len(groups) != len(want) {
		t.Fatalf("got %d groups, want %d: %+v", len(groups), len(want), groups)
	}
	for i := range want {
		if groups[i] != want[i] {
			t.Errorf("group %d = %+v, want %+v", i, groups[i], want[i])
		}
	}
}

func TestListGroupsReportsHTTPErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "permission check failed", http.StatusForbidden)
	}))
	defer srv.Close()

	pbs := newTestClient(srv.URL, nil)
	if _, err := pbs.ListGroups(); err == nil {
		t.Fatal("expected an error for HTTP 403")
	}
}

// Salvanza writes the machine's name on each snapshot, not on the group: the
// newest snapshot with a comment gives it, blank comments are skipped.
func TestSnapshotCommentsNewestPerGroup(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[` +
			`{"backup-type":"vm","backup-id":"9000198","backup-time":100,"comment":"Old name"},` +
			`{"backup-type":"vm","backup-id":"9000198","backup-time":300,"comment":"Deepthought - Full Machine"},` +
			`{"backup-type":"vm","backup-id":"9000198","backup-time":400,"comment":"  "},` +
			`{"backup-type":"host","backup-id":"laptop","backup-time":200}]}`))
	}))
	defer srv.Close()
	pbs := newTestClient(srv.URL, nil)
	got, err := pbs.SnapshotComments()
	if err != nil {
		t.Fatal(err)
	}
	if got["vm/9000198"] != "Deepthought - Full Machine" {
		t.Errorf("vm/9000198 = %q", got["vm/9000198"])
	}
	if _, ok := got["host/laptop"]; ok {
		t.Errorf("a group with no snapshot comment got %q", got["host/laptop"])
	}
}
