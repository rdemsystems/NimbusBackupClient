//go:build unix

package pbscommon

import "testing"

// A Unix restore of an archive with placeholder permissions gives files 0644
// when it was made on Windows and 0755 otherwise (an old Linux client), and
// leaves an archive with real permissions alone.
func TestPlaceholderFileMode(t *testing.T) {
	placeholder := placeholderArchive(t, 0)
	if got := NewPXARReader(placeholder).PlaceholderFileMode(&BackupMeta{OS: "windows"}); got != 0o644 {
		t.Errorf("Windows backup: %04o, want 0644", got)
	}
	if got := NewPXARReader(placeholder).PlaceholderFileMode(&BackupMeta{OS: "linux"}); got != 0o755 {
		t.Errorf("old Linux backup: %04o, want 0755", got)
	}
	if got := NewPXARReader(placeholder).PlaceholderFileMode(nil); got != 0o755 {
		t.Errorf("no meta: %04o, want 0755", got)
	}
	if got := NewPXARReader(placeholderArchive(t, 7)).PlaceholderFileMode(&BackupMeta{OS: "linux"}); got != 0 {
		t.Errorf("real permissions: %04o, want 0 (use them)", got)
	}
}
