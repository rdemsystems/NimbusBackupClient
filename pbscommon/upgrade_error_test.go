package pbscommon

import (
	"strings"
	"testing"
)

func TestUpgradeRejectionMessage(t *testing.T) {
	for _, tc := range []struct {
		name, statusLine, body, want string
	}{
		{
			name:       "owner check",
			statusLine: "HTTP/1.1 400 Bad Request\r",
			body:       "backup owner check failed (a@pbs!x != a@pbs)\n",
			want:       "PBS refused the backup (HTTP 400): backup owner check failed (a@pbs!x != a@pbs)",
		},
		{
			name:       "json body",
			statusLine: "HTTP/1.1 400 Bad Request",
			body:       `{"data":null,"message":"parameter verification errors"}`,
			want:       "PBS refused the backup (HTTP 400): parameter verification errors",
		},
		{
			name:       "authentication",
			statusLine: "HTTP/1.1 401 Unauthorized\r",
			body:       "permission check failed",
			want:       "PBS authentication failed (HTTP 401): permission check failed",
		},
		{
			name:       "access denied",
			statusLine: "HTTP/1.1 403 Forbidden\r",
			body:       "permission check failed - missing Datastore.Backup on /datastore/store1",
			want:       "PBS access denied (HTTP 403): permission check failed - missing Datastore.Backup on /datastore/store1",
		},
		{
			name:       "namespace",
			statusLine: "HTTP/1.1 404 Not Found\r",
			body:       "namespace not found",
			want:       "PBS refused the backup (HTTP 404): namespace not found",
		},
		{
			name:       "empty body",
			statusLine: "HTTP/1.1 400 Bad Request\r",
			body:       "",
			want:       "PBS refused the backup (HTTP 400): Bad Request",
		},
		{
			name:       "empty body, no reason phrase",
			statusLine: "HTTP/1.1 503\r",
			body:       "  \n",
			want:       "PBS refused the backup (HTTP 503): Service Unavailable",
		},
		{
			name:       "multi-line body",
			statusLine: "HTTP/1.1 400 Bad Request\r",
			body:       "line one\r\n  line two\n",
			want:       "PBS refused the backup (HTTP 400): line one line two",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := upgradeRejection(tc.statusLine, []byte(tc.body)).Error()
			if got != tc.want {
				t.Errorf("got  %q\nwant %q", got, tc.want)
			}
		})
	}
}

// The error carries the body only: the response headers stay in the debug log.
func TestUpgradeRejectionOmitsHeaders(t *testing.T) {
	err := upgradeRejection("HTTP/1.1 400 Bad Request\r", []byte("backup owner check failed"))
	if msg := err.Error(); strings.Contains(msg, "content-length") || strings.Contains(msg, "HTTP/1.1") {
		t.Errorf("headers leaked into the error: %q", msg)
	}
	if err.StatusCode != 400 || err.Status != "Bad Request" {
		t.Errorf("status = %d %q", err.StatusCode, err.Status)
	}
}
