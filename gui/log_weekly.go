package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Weekly run logs (16/11/2026). Each backup run writes its own log, and nothing
// removed them: an hourly Backup Set left about 8,700 files a year. A run's log
// is still written to its own file and compressed when the run ends, as before;
// it is then added to the end of one file for its week,
// backup-logs-<date of the week's Monday>.log.gz, and its own file removed.
//
// A gzip file may hold several compressed parts one after another and is still
// one valid gzip file: any gzip reader reads them all as one text, so the week
// can be opened as a whole (a log viewer, zcat). Each part here is one run's
// log, and its gzip header carries the run log's file name and the time it
// ended. The log viewer lists each part under that name, and reads only that
// part, so Reports, the log viewer and the text-mode window show one run's log
// as they did when it was a file of its own. Nothing else had to change for
// them.
//
// Retention is by whole weeks: a week file older than the weeks kept
// (Preferences > Message Log, 52 unless set, 0 for all) is removed when the
// settings are read or changed and after a run.

// DefaultLogRetentionWeeks is how many weeks of run logs are kept unless the
// settings say otherwise (Preferences > Message Log): this week and the 51
// before it. 0 in the settings keeps them all.
const DefaultLogRetentionWeeks = 52

// MaxLogRetentionWeeks is the most the setting may be (ten years).
const MaxLogRetentionWeeks = 520

// logRetention is the weeks of run logs kept, from the settings: -1 until
// the settings are read, and then nothing is removed (a week file is never
// removed on the default when the settings say keep all).
var logRetention atomic.Int64

func init() { logRetention.Store(-1) }

// logRetentionSetting is the setting as the app shows it: 52 when unset.
func logRetentionSetting(c *Config) int {
	if c == nil || c.LogRetentionWeeks == nil {
		return DefaultLogRetentionWeeks
	}
	w := *c.LogRetentionWeeks
	if w < 0 || w > MaxLogRetentionWeeks { // hand-edited: the default
		return DefaultLogRetentionWeeks
	}
	return w
}

// applyLogRetention takes the setting from c (applySpeedLimits, whenever the
// settings are loaded or changed). When it changes, the week files past it
// are removed at once.
func applyLogRetention(c *Config) {
	w := int64(logRetentionSetting(c))
	if logRetention.Swap(w) != w && logDir != "" {
		go foldRunLogs(logDir, "")
	}
}

// SetLogRetention saves Preferences > Message Log "Keep backup run logs": how
// many weeks, 1 to MaxLogRetentionWeeks, or 0 to keep them all.
func (a *App) SetLogRetention(weeks int) error {
	if weeks < 0 || weeks > MaxLogRetentionWeeks {
		return fmt.Errorf("run logs are kept for 1 to %d weeks, or 0 for all", MaxLogRetentionWeeks)
	}
	if a.delegated() {
		return a.serviceCall("SetLogRetention", weeks)
	}
	a.config.LogRetentionWeeks = &weeks
	if err := a.config.Save(); err != nil {
		return fmt.Errorf("failed to save how long run logs are kept: %w", err)
	}
	applyLogRetention(a.config)
	writeDebugLog(fmt.Sprintf("Run logs kept for %d weeks (0 = all)", weeks))
	return nil
}

// weeklyLogName matches a week's file of run logs.
var weeklyLogName = regexp.MustCompile(`^backup-logs-(\d{8})\.log\.gz$`)

// weeklyMu: one fold or prune at a time in this process. A folder of logs is
// written by one process only (the service's by the service, the app's by the
// app).
var weeklyMu sync.Mutex

// weekStart is midnight at the start of the Monday of t's week, in t's zone.
func weekStart(t time.Time) time.Time {
	d := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	back := (int(d.Weekday()) + 6) % 7 // Monday 0 ... Sunday 6
	return d.AddDate(0, 0, -back)
}

// weeklyLogFile is the name of the week file a run that started at t goes in.
func weeklyLogFile(t time.Time) string {
	return "backup-logs-" + weekStart(t).Format("20060102") + ".log.gz"
}

// runLogStart is the start of the run whose log (or rotated part of it, or its
// compressed copy) is called name, from the name.
func runLogStart(name string) (time.Time, bool) {
	m := runLogName.FindStringSubmatch(name)
	if m == nil {
		return time.Time{}, false
	}
	t, err := time.ParseInLocation("20060102150405", m[1]+m[2], time.Local)
	return t, err == nil
}

// weeklyMember is one run log inside a week file.
type weeklyMember struct {
	Name     string    // the run log's own file name (backup-...-<id>.log.gz)
	Modified time.Time // when it was written (the run's end)
	Offset   int64     // where its gzip part starts in the week file
	Length   int64     // its compressed length
}

type weeklyScan struct {
	size    int64
	mod     time.Time
	members []weeklyMember
	goodEnd int64 // the end of the last whole part (the file's size unless its end was cut off)
}

var (
	weeklyCacheMu sync.Mutex
	weeklyCache   = map[string]weeklyScan{}
)

// countingReader counts the bytes the gzip reader takes. It is an
// io.ByteReader, so the gzip reader reads no further than the part it is in
// and the count is where the next part starts.
type countingReader struct {
	r *bufio.Reader
	n int64
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.n += int64(n)
	return n, err
}

func (c *countingReader) ReadByte() (byte, error) {
	b, err := c.r.ReadByte()
	if err == nil {
		c.n++
	}
	return b, err
}

// scanWeeklyLog lists the run logs in a week file. A part whose end is cut off
// (the computer stopped while one was being added) and anything after it are
// not listed; goodEnd says where the whole parts end. The result is kept until
// the file changes.
func scanWeeklyLog(path string) (weeklyScan, error) {
	info, err := os.Stat(path)
	if err != nil {
		return weeklyScan{}, err
	}
	weeklyCacheMu.Lock()
	if c, ok := weeklyCache[path]; ok && c.size == info.Size() && c.mod.Equal(info.ModTime()) {
		weeklyCacheMu.Unlock()
		return c, nil
	}
	weeklyCacheMu.Unlock()

	f, err := os.Open(path)
	if err != nil {
		return weeklyScan{}, err
	}
	defer func() { _ = f.Close() }()
	cr := &countingReader{r: bufio.NewReader(f)}
	scan := weeklyScan{size: info.Size(), mod: info.ModTime()}
	var z *gzip.Reader
	for {
		start := cr.n
		if z == nil {
			z, err = gzip.NewReader(cr)
		} else {
			err = z.Reset(cr)
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			break // a part cut off in its header
		}
		z.Multistream(false)
		if _, err := io.Copy(io.Discard, z); err != nil {
			break // a part cut off in its data
		}
		scan.members = append(scan.members, weeklyMember{Name: z.Name, Modified: z.ModTime, Offset: start, Length: cr.n - start})
		scan.goodEnd = cr.n
	}
	weeklyCacheMu.Lock()
	weeklyCache[path] = scan
	weeklyCacheMu.Unlock()
	return scan, nil
}

// openWeeklyRunLog opens the run log called name from its week file in dir,
// decompressed; ok is false when the week file does not hold it.
func openWeeklyRunLog(dir, name string) (io.ReadCloser, bool) {
	start, ok := runLogStart(name)
	if !ok {
		return nil, false
	}
	path := filepath.Join(dir, weeklyLogFile(start))
	scan, err := scanWeeklyLog(path)
	if err != nil {
		return nil, false
	}
	for _, m := range scan.members {
		if m.Name != name {
			continue
		}
		f, err := os.Open(path)
		if err != nil {
			return nil, false
		}
		z, err := gzip.NewReader(io.NewSectionReader(f, m.Offset, m.Length))
		if err != nil {
			_ = f.Close()
			return nil, false
		}
		z.Multistream(false)
		return struct {
			io.Reader
			io.Closer
		}{z, f}, true
	}
	return nil, false
}

// weeklyRunLogs lists the run logs in the week files in dir, as log files of
// their own.
func weeklyRunLogs(dir string, entries []os.DirEntry) []weeklyMember {
	var out []weeklyMember
	for _, e := range entries {
		if e.IsDir() || !weeklyLogName.MatchString(e.Name()) {
			continue
		}
		scan, err := scanWeeklyLog(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		out = append(out, scan.members...)
	}
	return out
}

// foldRunLog adds the compressed run log called name in dir to its week file
// and removes it. A file that does not decompress whole (one still being
// written) is left as it is. A run log the week file already holds (the
// computer stopped after it was added, before the file was removed) is only
// removed.
func foldRunLog(dir, name string) error {
	start, ok := runLogStart(name)
	if !ok || !strings.HasSuffix(name, ".gz") {
		return fmt.Errorf("%s is not a compressed run log", name)
	}
	src := filepath.Join(dir, name)
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	// #nosec G304 -- a run log in the log folder, by a name runLogName admits
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	zr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	plain, err := io.ReadAll(zr)
	if err != nil {
		return fmt.Errorf("%s does not decompress whole: %w", name, err)
	}
	var part bytes.Buffer
	zw := gzip.NewWriter(&part)
	zw.Name = name
	zw.ModTime = info.ModTime()
	if _, err := zw.Write(plain); err != nil {
		return err
	}
	if err := zw.Close(); err != nil {
		return err
	}

	week := filepath.Join(dir, weeklyLogFile(start))
	if scan, err := scanWeeklyLog(week); err == nil {
		for _, m := range scan.members {
			if m.Name == name {
				return removeFolded(src)
			}
		}
		// A part cut off at the end is dropped before the next is added, or
		// everything after it could not be read.
		if scan.goodEnd < scan.size {
			if err := os.Truncate(week, scan.goodEnd); err != nil {
				return fmt.Errorf("could not cut the broken end off %s: %w", filepath.Base(week), err)
			}
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	f, err := os.OpenFile(week, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return err
	}
	_, werr := f.Write(part.Bytes())
	serr := f.Sync()
	cerr := f.Close()
	if werr != nil || serr != nil || cerr != nil {
		return fmt.Errorf("could not add %s to %s: %v", name, filepath.Base(week), errors.Join(werr, serr, cerr))
	}
	return removeFolded(src)
}

func removeFolded(path string) error {
	if err := os.Remove(path); err != nil {
		time.Sleep(500 * time.Millisecond) // Windows: a scanner may hold it a moment
		return os.Remove(path)
	}
	return nil
}

// foldRunLogs adds the compressed run logs in dir to their week files: all of
// them, or, with prefix, those of the one run whose log file is called prefix
// (its main file and its rotated parts). Then it removes the week files older
// than the setting keeps (none before the settings are read, or when they
// keep all).
func foldRunLogs(dir, prefix string) {
	weeklyMu.Lock()
	defer weeklyMu.Unlock()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".gz") || (prefix != "" && !strings.HasPrefix(name, prefix+".")) {
			continue
		}
		if _, ok := runLogStart(name); !ok {
			continue
		}
		if err := foldRunLog(dir, name); err != nil {
			fmt.Fprintf(os.Stderr, "weekly logs: %v\n", err)
		}
	}
	if keep := int(logRetention.Load()); keep > 0 {
		if entries, err = os.ReadDir(dir); err == nil {
			pruneWeeklyLogs(dir, entries, keep, time.Now())
		}
	}
}

// pruneWeeklyLogs removes the week files of weeks that started more than
// keepWeeks-1 weeks before this one.
func pruneWeeklyLogs(dir string, entries []os.DirEntry, keepWeeks int, now time.Time) {
	if keepWeeks <= 0 {
		return
	}
	oldest := weekStart(now).AddDate(0, 0, -7*(keepWeeks-1))
	var gone []string
	for _, e := range entries {
		m := weeklyLogName.FindStringSubmatch(e.Name())
		if m == nil {
			continue
		}
		t, err := time.ParseInLocation("20060102", m[1], now.Location())
		if err != nil || !t.Before(oldest) {
			continue
		}
		if err := os.Remove(filepath.Join(dir, e.Name())); err == nil {
			gone = append(gone, e.Name())
		}
	}
	if len(gone) > 0 {
		sort.Strings(gone)
		fmt.Fprintf(os.Stderr, "weekly logs: removed %d week files older than %d weeks: %s\n", len(gone), keepWeeks, strings.Join(gone, ", "))
	}
}
