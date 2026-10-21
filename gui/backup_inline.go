package main

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/alphadose/haxmap"
	"machinebackuplib"
	"pbscommon"
	"retry"
	"snapshot"
)

// BackupOptions contains all parameters for a backup operation
type BackupOptions struct {
	Ctx             context.Context // Cancel to request a graceful stop between backup steps
	BaseURL         string
	AuthID          string
	Secret          string
	Ticket          string // PBS session ticket (u/p login); preferred over AuthID/Secret when set
	CSRFToken       string
	Datastore       string
	Namespace       string
	CertFingerprint string
	BackupObjects   []string // Multiple directories or drives to backup
	BackupID        string
	BackupType      string // "host" for GUI backups; "vm" additionally uploads a VM config and needs a numeric BackupID
	Kind            string // "disk", "directory", or "machine"
	UseVSS          bool
	Compression     string   // Compression level: "fastest", "default", "better", "best"
	ExcludeList     []string // User-configured exclusion patterns applied by the PXAR writer (H-04)
	DisableSplit    bool     // When true, never auto-split regardless of size
	SplitSizeBytes  uint64   // Auto-split threshold and per-bin target; 0 = default (SplitThreshold)
	// Crypt encrypts every chunk and signs the manifest. Nil means a plain
	// snapshot. Populated from Config.Crypt (Config.EncryptionKeyFile).
	Crypt      *pbscommon.CryptConfig
	OnProgress func(percent float64, message string)
	OnComplete func(success bool, message string)
	// OnResult delivers the full structured result (Group 0 contract). It is
	// additive: OnComplete keeps firing with the success bool for existing
	// consumers. OnResult is the source the sidecar (Group 1) and rich history read.
	OnResult func(*BackupStatus)
	// OnStats delivers structured live progress so the GUI can show real
	// statistics instead of parsing them out of the progress message string.
	OnStats func(*BackupProgressStats)
	// Parallel is how many folders of a multi-folder backup run at the same
	// time (0 or 1: one after the other). No upper limit; the GUI recommends
	// CPUs / 4 (RecommendedParallel).
	Parallel int
	// OneSnapshot writes every folder into one snapshot, one archive per
	// folder (runOneSnapshotBackup), instead of one backup group per folder.
	// Parallel does not apply: the folders share one session.
	OneSnapshot bool

	// Set by runFoldersParallel on each folder's options: the batch already
	// holds the destination lock, and under VSS (batchVSS) the folder reads
	// snapshotPath, taken once for the whole batch.
	lockHeld     bool
	batchVSS     bool
	snapshotPath string
}

// RecommendedParallel is the suggested number of folders backed up at the same
// time: CPUs / 4, at least 1.
func RecommendedParallel() int {
	return max(runtime.NumCPU()/4, 1)
}

// isFatalSessionError returns true for errors that make the current PBS session
// unusable. These indicate the H2 connection was lost; the session state on the
// server is gone, and all subsequent operations on this session will fail.
// The only recovery is a fresh Connect() with a new backup-time.
func isFatalSessionError(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "PBS session cannot be resumed") ||
		strings.Contains(s, "connection lost") ||
		strings.Contains(s, "unexpected EOF") ||
		strings.Contains(s, "writer '") && strings.Contains(s, "not registered")
}

// Global backup locks per destination (BaseURL + Datastore)
var (
	backupLocks      = make(map[string]*sync.Mutex)
	backupLocksMutex sync.Mutex
)

// Current in-flight backup cancellation. The GUI's Stop button calls
// CancelBackup() which cancels this context. The archive writer checks it at
// every write, so the folder being backed up stops at once and its session is
// dropped without committing a partial snapshot.
var (
	currentBackupCancelMutex sync.Mutex
	currentBackupCancel      context.CancelFunc
)

// newBackupContext returns a fresh cancellable context and registers its
// cancel function as the current in-flight backup. Callers should not defer
// Cancel() (that would also cancel the shared token); instead call
// doneBackupContext() when the run finishes.
func newBackupContext() (context.Context, context.CancelFunc) {
	currentBackupCancelMutex.Lock()
	defer currentBackupCancelMutex.Unlock()
	if currentBackupCancel != nil {
		currentBackupCancel() // cancel any previous run before replacing
	}
	ctx, cancel := context.WithCancel(context.Background())
	currentBackupCancel = cancel
	return ctx, cancel
}

// doneBackupContext clears the shared cancellation token when a run finishes.
func doneBackupContext() {
	currentBackupCancelMutex.Lock()
	defer currentBackupCancelMutex.Unlock()
	currentBackupCancel = nil
}

// getBackupLock returns a mutex for the given backup destination
func getBackupLock(baseURL, datastore string) *sync.Mutex {
	key := baseURL + "|" + datastore
	backupLocksMutex.Lock()
	defer backupLocksMutex.Unlock()

	if _, exists := backupLocks[key]; !exists {
		backupLocks[key] = &sync.Mutex{}
	}
	return backupLocks[key]
}

// calculateDirSizeCtx scans a directory recursively and returns total size in bytes.
//
// It is hardened for whole-drive roots like C:\ in two ways:
//   - It never descends into directory junctions / reparse points. On a system
//     drive these loop (e.g. C:\ProgramData\Application Data -> C:\ProgramData,
//     C:\Documents and Settings -> C:\Users), which can make a naive walk run
//     effectively forever. The pxar writer already skips them (pxar.go WriteDir),
//     so they don't belong in the size estimate either.
//   - It honors ctx, so a pathologically large or slow drive (e.g. one where
//     antivirus scans every file open) can't stall the caller indefinitely; on
//     deadline it returns the partial size gathered so far.
//
// Returns size and error if access was denied at the root (needs VSS) or if ctx
// fired (partial size is still returned alongside the context error).
func calculateDirSizeCtx(ctx context.Context, path string) (uint64, error) {
	var totalSize uint64
	var accessDenied bool

	err := filepath.WalkDir(path, func(filePath string, d os.DirEntry, err error) error {
		if ctx.Err() != nil {
			// Deadline/cancel: stop walking and report what we have so far.
			return ctx.Err()
		}
		if err != nil {
			// Check if it's an access denied error
			if strings.Contains(err.Error(), "Access is denied") ||
				strings.Contains(err.Error(), "permission denied") {
				accessDenied = true
				// Stop walking this path, but continue with others
				if filePath == path {
					// Root path denied - cannot scan at all
					return fmt.Errorf("access denied to root path")
				}
				return filepath.SkipDir
			}
			return nil // Skip other errors
		}
		// Never follow junctions / symlinks (see doc comment): on Windows
		// ReadDir reports mount points with ModeSymlink, so skip the whole
		// subtree rather than recursing into a loop.
		if d.Type()&os.ModeSymlink != 0 {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.IsDir() {
			info, err := d.Info()
			if err != nil {
				return nil // file vanished or became unreadable: ignore for sizing
			}
			totalSize += uint64(info.Size())
		}
		return nil
	})

	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		// Partial size is still useful for the split heuristic.
		return totalSize, err
	}
	if accessDenied && totalSize == 0 {
		return 0, fmt.Errorf("access denied: %s", path)
	}

	return totalSize, err
}

type ChunkState struct {
	assignments         []string
	assignmentsOffset   []uint64
	pos                 uint64
	wrid                uint64
	chunkcount          uint64
	chunkdigests        hash.Hash
	currentChunk        []byte
	C                   pbscommon.Chunker
	newchunk            *atomic.Uint64
	reusechunk          *atomic.Uint64
	failedchunk         *atomic.Uint64 // Track failed chunk uploads
	knownChunks         *haxmap.Map[string, bool]
	onProgress          func(float64, string)
	onStats             func(*BackupProgressStats) // Structured live stats for the GUI (nil for the catalog stream)
	currentDir          string                     // Directory currently being archived, for the stats payload
	lastProgressReport  uint64
	lastProgressPercent float64        // Track last reported percentage to prevent backwards progress
	totalSize           *atomic.Uint64 // Total size, updated by background scan
	uploadErrors        []string       // Collect upload errors to report at the end
	errorsMutex         sync.Mutex     // Protect uploadErrors slice

	// cancelCtx is the Stop button's context: it ends the upload retries.
	// nil = never cancelled.
	cancelCtx context.Context
}

func (c *ChunkState) Init(newchunk *atomic.Uint64, reusechunk *atomic.Uint64, failedchunk *atomic.Uint64, knownChunks *haxmap.Map[string, bool], onProgress func(float64, string), totalSize *atomic.Uint64, onStats func(*BackupProgressStats), currentDir string) {
	c.assignments = make([]string, 0)
	c.assignmentsOffset = make([]uint64, 0)
	c.pos = 0
	c.chunkcount = 0
	c.chunkdigests = sha256.New()
	c.currentChunk = make([]byte, 0)
	c.C = pbscommon.Chunker{}
	// Chunk size avg = 4MB → max = 16MB (PBS hard limit on chunk size)
	// Was 8MB (max=32MB) as workaround for "Invalid string length" errors,
	// but the real cause was broken JSON encoding in CreateDynamicIndex (fixed in 16fbba8)
	c.C.New(1024 * 1024 * 4)
	c.reusechunk = reusechunk
	c.newchunk = newchunk
	c.failedchunk = failedchunk
	c.knownChunks = knownChunks
	c.onProgress = onProgress
	c.onStats = onStats
	c.currentDir = currentDir
	c.lastProgressReport = 0
	c.lastProgressPercent = 0.0
	c.totalSize = totalSize
	c.uploadErrors = make([]string, 0)
}

// retryParent is the context the chunk upload retries derive from: the Stop
// button's one when set, so a cancel ends the retries and their waits.
func (c *ChunkState) retryParent() context.Context {
	if c.cancelCtx != nil {
		return c.cancelCtx
	}
	return context.Background()
}

func (c *ChunkState) HandleData(b []byte, client *pbscommon.PBSClient) error {
	chunkpos := c.C.Scan(b)

	if chunkpos == 0 {
		c.currentChunk = append(c.currentChunk, b...)
	} else {
		for chunkpos > 0 {
			c.currentChunk = append(c.currentChunk, b[:chunkpos]...)

			// The digest has to be the one the chunk is published under:
			// sha256(plaintext) for a plain snapshot, but
			// sha256(plaintext || id_key) once a key file is in play. A bare
			// sha256 on an encrypted snapshot is rejected by
			// proxmox-backup-client ("wrong digest") and leaks the plaintext
			// fingerprint to the server.
			bindigest := client.ChunkDigest(c.currentChunk)
			shahash := hex.EncodeToString(bindigest[:])

			if _, known := c.knownChunks.Get(shahash); !known {
				writeBackupLog(fmt.Sprintf("New chunk[%s] %d bytes", shahash, len(c.currentChunk)))

				// Retry chunk upload with exponential backoff
				chunkData := c.currentChunk // Capture for closure
				retryConfig := retry.DefaultConfig()
				retryConfig.MaxAttempts = 5 // More retries for chunk uploads
				ctx, cancel := context.WithTimeout(c.retryParent(), 5*time.Minute)
				defer cancel()

				err := retry.DoWithJitter(ctx, retryConfig, retry.DefaultRetryable, func() error {
					return client.UploadDynamicCompressedChunk(c.wrid, shahash, chunkData)
				})
				if err != nil {
					errMsg := fmt.Sprintf("⚠️  Failed to upload chunk %s after %d retries: %v", shahash, retryConfig.MaxAttempts, err)
					writeBackupLog(errMsg)
					c.failedchunk.Add(1)

					// Collect error for final report
					c.errorsMutex.Lock()
					c.uploadErrors = append(c.uploadErrors, errMsg)
					c.errorsMutex.Unlock()

					// C3 fail-closed: a chunk we could not upload must NOT be indexed.
					// Indexing it would close a dynamic index referencing a digest that
					// is absent from the datastore — an unrestorable snapshot. Abort the
					// writer now (whether the error is fatal or transient); WriteDir's EOF,
					// CloseDynamicIndex, manifest and Finish are never reached for this dir.
					return fmt.Errorf("chunk upload failed, aborting to avoid committing a corrupt snapshot: %w", err)
				}
				// Mark the chunk known ONLY after a confirmed upload, so a failed chunk
				// can never make a later identical chunk be skipped as "reused".
				c.knownChunks.Set(shahash, true)
				c.newchunk.Add(1)
			} else {
				writeBackupLog(fmt.Sprintf("Reuse chunk[%s] %d bytes", shahash, len(c.currentChunk)))
				c.reusechunk.Add(1)
			}

			if err := binary.Write(c.chunkdigests, binary.LittleEndian, (c.pos + uint64(len(c.currentChunk)))); err != nil {
				return fmt.Errorf("failed to write chunk offset: %w", err)
			}
			if _, err := c.chunkdigests.Write(bindigest[:]); err != nil {
				return fmt.Errorf("failed to write chunk digest: %w", err)
			}

			c.assignmentsOffset = append(c.assignmentsOffset, c.pos)
			c.assignments = append(c.assignments, shahash)
			c.pos += uint64(len(c.currentChunk))
			c.chunkcount += 1

			// Report progress every 10 MB
			if c.onProgress != nil && c.pos-c.lastProgressReport > 10*1024*1024 {
				c.lastProgressReport = c.pos
				sizeMB := c.pos / (1024 * 1024)

				// Build progress message with chunk stats
				var msg string
				failed := c.failedchunk.Load()
				if failed > 0 {
					msg = fmt.Sprintf("Processed: %d MB (New: %d, Reused: %d, ⚠️ Failed: %d chunks)",
						sizeMB, c.newchunk.Load(), c.reusechunk.Load(), failed)
				} else {
					msg = fmt.Sprintf("Processed: %d MB (New: %d, Reused: %d chunks)",
						sizeMB, c.newchunk.Load(), c.reusechunk.Load())
				}

				// Calculate progress based on total size if available
				var progress float64
				totalSize := c.totalSize.Load()
				if totalSize > 0 {
					// Progress from 10% to 90% based on bytes processed
					progress = 0.1 + (float64(c.pos)/float64(totalSize))*0.8
					if progress > 0.9 {
						progress = 0.9
					}
					if failed > 0 {
						msg = fmt.Sprintf("Processed: %d / %d MB (New: %d, Reused: %d, ⚠️ Failed: %d chunks)",
							sizeMB, totalSize/(1024*1024), c.newchunk.Load(), c.reusechunk.Load(), failed)
					} else {
						msg = fmt.Sprintf("Processed: %d / %d MB (New: %d, Reused: %d chunks)",
							sizeMB, totalSize/(1024*1024), c.newchunk.Load(), c.reusechunk.Load())
					}
				} else {
					// No total size yet, show indeterminate progress
					progress = 0.1 + float64(sizeMB%100)/1000.0 // Slowly increment from 10%
					if progress > 0.5 {
						progress = 0.5
					}
				}

				// Never report backwards progress - totalSize can increase during backup
				if progress < c.lastProgressPercent {
					progress = c.lastProgressPercent
				}
				c.lastProgressPercent = progress

				c.onProgress(progress, msg)

				// Structured live stats for the GUI (same cadence as the message).
				if c.onStats != nil {
					c.onStats(&BackupProgressStats{
						Percent:      progress,
						BytesDone:    c.pos,
						BytesTotal:   totalSize,
						NewChunks:    c.newchunk.Load(),
						ReusedChunks: c.reusechunk.Load(),
						FailedChunks: failed,
						CurrentDir:   c.currentDir,
						Message:      msg,
					})
				}
			}

			c.currentChunk = make([]byte, 0)
			b = b[chunkpos:]
			chunkpos = c.C.Scan(b)
		}
		c.currentChunk = append(c.currentChunk, b...)
	}
	return nil
}

func (c *ChunkState) EOF(client *pbscommon.PBSClient) error {
	if len(c.currentChunk) > 0 {
		// Same digest rule as in HandleData.
		bindigest := client.ChunkDigest(c.currentChunk)
		shahash := hex.EncodeToString(bindigest[:])
		if err := binary.Write(c.chunkdigests, binary.LittleEndian, (c.pos + uint64(len(c.currentChunk)))); err != nil {
			return fmt.Errorf("failed to write final chunk offset: %w", err)
		}
		if _, err := c.chunkdigests.Write(bindigest[:]); err != nil {
			return fmt.Errorf("failed to write final chunk digest: %w", err)
		}

		if _, known := c.knownChunks.Get(shahash); !known {
			writeBackupLog(fmt.Sprintf("New chunk[%s] %d bytes", shahash, len(c.currentChunk)))

			// Retry final chunk upload with exponential backoff
			chunkData := c.currentChunk
			retryConfig := retry.DefaultConfig()
			retryConfig.MaxAttempts = 5
			ctx, cancel := context.WithTimeout(c.retryParent(), 5*time.Minute)
			defer cancel()

			err := retry.DoWithJitter(ctx, retryConfig, retry.DefaultRetryable, func() error {
				return client.UploadDynamicCompressedChunk(c.wrid, shahash, chunkData)
			})
			if err != nil {
				errMsg := fmt.Sprintf("⚠️  Failed to upload final chunk %s after %d retries: %v", shahash, retryConfig.MaxAttempts, err)
				writeBackupLog(errMsg)
				c.failedchunk.Add(1)

				c.errorsMutex.Lock()
				c.uploadErrors = append(c.uploadErrors, errMsg)
				c.errorsMutex.Unlock()

				// C3 fail-closed (see HandleData): never index an unconfirmed chunk —
				// abort before the chunk is appended to the index and before EOF closes it.
				return fmt.Errorf("final chunk upload failed, aborting to avoid committing a corrupt snapshot: %w", err)
			}
			// Mark known only after a confirmed upload.
			c.knownChunks.Set(shahash, true)
			c.newchunk.Add(1)
		} else {
			writeBackupLog(fmt.Sprintf("Reuse chunk[%s] %d bytes", shahash, len(c.currentChunk)))
			c.reusechunk.Add(1)
		}
		c.assignmentsOffset = append(c.assignmentsOffset, c.pos)
		c.assignments = append(c.assignments, shahash)
		c.pos += uint64(len(c.currentChunk))
		c.chunkcount += 1
	}

	// Assign chunks in batches with retry
	retryConfig := retry.DefaultConfig()
	retryConfig.MaxAttempts = 5
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	for k := 0; k < len(c.assignments); k += 128 {
		k2 := k + 128
		if k2 > len(c.assignments) {
			k2 = len(c.assignments)
		}

		// Capture loop variables for closure
		batchStart, batchEnd := k, k2
		assignments := c.assignments[batchStart:batchEnd]
		offsets := c.assignmentsOffset[batchStart:batchEnd]

		err := retry.DoWithJitter(ctx, retryConfig, retry.DefaultRetryable, func() error {
			return client.AssignDynamicChunks(c.wrid, assignments, offsets)
		})
		if err != nil {
			return fmt.Errorf("failed to assign chunks (batch %d-%d) after retries: %w", batchStart, batchEnd, err)
		}
	}

	// Close index with retry
	digest := hex.EncodeToString(c.chunkdigests.Sum(nil))
	err := retry.DoWithJitter(ctx, retryConfig, retry.DefaultRetryable, func() error {
		return client.CloseDynamicIndex(c.wrid, digest, c.pos, c.chunkcount)
	})
	if err != nil {
		return fmt.Errorf("failed to close dynamic index after retries: %w", err)
	}
	return nil
}

// formatDuration formats a duration in a human-readable format
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		mins := int(d.Minutes())
		secs := int(d.Seconds()) % 60
		return fmt.Sprintf("%dm %ds", mins, secs)
	}
	hours := int(d.Hours())
	mins := int(d.Minutes()) % 60
	return fmt.Sprintf("%dh %dm", hours, mins)
}

// RunBackupInline performs a backup without external binaries
func RunBackupInline(opts BackupOptions) (returnErr error) {
	// Per-run log file: each backup run gets its own dedicated log file.
	// Must be set up BEFORE any writeBackupLog call so logs land in the right place.
	runLogID := opts.BackupID
	if runLogID == "" {
		if h, err := os.Hostname(); err == nil {
			runLogID = h
		} else {
			runLogID = "backup"
		}
	}
	runLogger := StartBackupRunLog(runLogID)
	defer EndBackupRunLog(runLogger)

	// Wire a shared, user-cancellable context for this run (Stop button).
	ctx, cancel := newBackupContext()
	defer doneBackupContext()
	// Callers may supply their own context; otherwise use the shared one.
	if opts.Ctx == nil {
		opts.Ctx = ctx
	}
	_ = cancel

	// CRITICAL: Panic recovery to prevent silent goroutine death (scheduler launches backups in goroutines)
	defer func() {
		if r := recover(); r != nil {
			errMsg := fmt.Sprintf("CRITICAL: Backup panic in RunBackupInline: %v", r)
			writeBackupLog(errMsg)
			// Get stack trace
			buf := make([]byte, 4096)
			n := runtime.Stack(buf, false)
			writeBackupLog(fmt.Sprintf("Stack trace:\n%s", buf[:n]))
			returnErr = fmt.Errorf("backup panic: %v", r)
		}
	}()

	userName := "<unknown>"
	if cu, err := user.Current(); err == nil {
		userName = cu.Username
	}
	writeBackupLog(fmt.Sprintf("==== starting version %s - user %s - %s/%s ====",
		appVersion, userName, runtime.GOOS, runtime.GOARCH))

	// Validate options. Log the resolved target (secret presence only, never the
	// secret) so a misconfiguration is diagnosable from the backup log alone — a
	// prod report showed only "Validating backup options" then nothing because an
	// empty multi-PBS resolution failed here silently.
	writeBackupLog(fmt.Sprintf("[DEBUG] Validating backup options: target=%q datastore=%q ns=%q authid=%q secret=%v dirs=%d backupID=%q",
		opts.BaseURL, opts.Datastore, opts.Namespace, opts.AuthID, opts.Secret != "", len(opts.BackupObjects), opts.BackupID))
	hasToken := opts.AuthID != "" && opts.Secret != ""
	hasTicket := opts.Ticket != ""
	if opts.BaseURL == "" || (!hasToken && !hasTicket) {
		var missing []string
		if opts.BaseURL == "" {
			missing = append(missing, "BaseURL")
		}
		if !hasToken && !hasTicket {
			missing = append(missing, "API token (AuthID+Secret) or session ticket")
		}
		errMsg := fmt.Sprintf("PBS connection parameters required (missing: %s) — check the selected/default PBS server in the config",
			strings.Join(missing, ", "))
		writeBackupLog("[ERROR] " + errMsg)
		return fmt.Errorf("%s", errMsg)
	}
	writeBackupLog("[DEBUG] Options validated")

	if len(opts.BackupObjects) == 0 {
		return fmt.Errorf("at least one backup directory or drive required")
	}

	// Get hostname for backup-id generation
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unnamed-backup"
	}

	// Splitting a backup into smaller parts is now an explicit, opt-in choice the
	// user makes in the GUI (the split plan is built and orchestrated there via
	// CreateBackupSplitPlan, which issues one StartBackup per part). RunBackupInline
	// therefore no longer sizes the directories or auto-splits by total size: that
	// pre-pass was what made a whole-drive backup of C:\ hang before it ever began,
	// and it also ran on every scheduled run. A scheduled/recurring backup is always
	// a normal (unsplit) backup — exactly the intended "split the first seed, full
	// afterwards" behavior.

	// OneSnapshot (the default for new jobs): every folder is an archive of one
	// snapshot, see runOneSnapshotBackup. Otherwise (jobs made before 0.5):
	// each selected directory is its OWN backup group (its own
	// backup-id), so PBS retention treats successive runs of the same folder as one
	// series. Previously all selected folders shared one backup-id (derived from the
	// first), landing as separate snapshots in a single group — which makes prune
	// keep/drop the wrong folders. A single selected directory keeps the caller's
	// backup-id (which may have been set explicitly, e.g. by a scheduled job).
	// Machine backups are never split: all selected disks belong to ONE "vm"
	// snapshot whose backup-id is the numeric VM ID (see machinebackuplib).
	if opts.OneSnapshot && opts.Kind != "machine" {
		return runOneSnapshotBackup(opts)
	}
	if len(opts.BackupObjects) <= 1 || opts.Kind == "machine" {
		return runBackupInlineInternal(opts)
	}

	// Each folder runs as its own internal backup, but the per-folder terminal
	// callbacks are SUPPRESSED and aggregated into a single OnComplete/OnResult for
	// the whole multi-folder run. Otherwise the first folder's OnComplete would
	// finalize (and, in API mode, tear down) the job before the rest ran — losing
	// later folders' status and breaking the honest-result contract. Live progress
	// (OnProgress/OnStats) still flows per folder.
	aggStart := time.Now()
	agg := &BackupStatus{Outcome: OutcomeVerifiedSuccess, BackupID: opts.BackupID, BackupTime: aggStart.Unix()}
	var perDirErrors []string

	// Each folder gets a distinct backup-id DERIVED FROM the base id (the caller's
	// custom backup-id, or the hostname). Deriving it as "<base>_<path>" keeps the
	// base a substring of every child id, so restore's substring search on the
	// configured backup-id still discovers all of a multi-folder backup's groups.
	baseID := opts.BackupID
	if baseID == "" {
		baseID = hostname
	}

	// One after the other unless Parallel says otherwise; either way from one
	// VSS snapshot set taken for every folder (see runFoldersParallel).
	parallel := max(min(opts.Parallel, len(opts.BackupObjects)), 1)
	perDirErrors = runFoldersParallel(opts, baseID, parallel, agg)

	agg.DurationSec = time.Since(aggStart).Seconds()
	if len(perDirErrors) > 0 {
		agg.Message = fmt.Sprintf("%d/%d folders failed:\n%s", len(perDirErrors), len(opts.BackupObjects), strings.Join(perDirErrors, "\n"))
	} else {
		agg.Message = fmt.Sprintf("Backup of %d folders complete (%d new, %d reused chunks)", len(opts.BackupObjects), agg.NewChunks, agg.ReusedChunks)
	}

	if opts.OnComplete != nil {
		opts.OnComplete(agg.Success(), agg.Message)
	}
	if opts.OnResult != nil {
		opts.OnResult(agg)
	}
	if len(perDirErrors) > 0 {
		return fmt.Errorf("%s", agg.Message)
	}
	return nil
}

// errFoldersFailed tells runFoldersParallel's snapshot callback that some
// folders failed (already reported per folder).
var errFoldersFailed = errors.New("some folders failed")

// runFoldersParallel backs up up to parallel folders at the same time (1: one
// after the other), each as its own backup group and PBS session, merging
// every result into agg. It returns the per-folder errors.
//
// The batch takes the destination lock once (folders skip it, or they would
// run one by one). Under VSS, ONE snapshot set covering every folder is taken
// before any upload, so the folders are frozen together (one shadow copy per
// volume) rather than each at its own turn. It also keeps concurrent folders
// safe: concurrent snapshot creations collide ("VSS busy"), and the
// busy-recovery path deletes every shadow copy, pulling it from under the
// folders already running. Progress is the average of the folders,
// statistics their sum.
func runFoldersParallel(opts BackupOptions, baseID string, parallel int, agg *BackupStatus) []string {
	dirs := opts.BackupObjects
	writeBackupLog(fmt.Sprintf("[Parallel] Backing up %d folders, %d at a time", len(dirs), parallel))

	cancelledBeforeStart := func() []string {
		if opts.Ctx == nil || opts.Ctx.Err() == nil {
			return nil
		}
		agg.Outcome = OutcomeFailed
		errMsg := fmt.Sprintf("backup cancelled: %d/%d folder(s) not backed up", len(dirs), len(dirs))
		writeBackupLog("Cancellation requested — " + errMsg)
		return []string{errMsg}
	}
	if errs := cancelledBeforeStart(); errs != nil {
		return errs
	}

	backupLock := getBackupLock(opts.BaseURL, opts.Datastore)
	if opts.OnProgress != nil {
		opts.OnProgress(0, "Waiting for previous backup to complete...")
	}
	backupLock.Lock()
	defer backupLock.Unlock()
	if errs := cancelledBeforeStart(); errs != nil {
		return errs
	}

	var mu sync.Mutex
	var perDirErrors []string
	fractions := make([]float64, len(dirs))
	stats := make([]BackupProgressStats, len(dirs))
	started := 0

	reportProgress := func(i int, pct float64, msg string) {
		mu.Lock()
		fractions[i] = pct
		sum := 0.0
		for _, f := range fractions {
			sum += f
		}
		mu.Unlock()
		if opts.OnProgress != nil {
			opts.OnProgress(sum/float64(len(dirs)), fmt.Sprintf("[%s] %s", filepath.Base(dirs[i]), msg))
		}
	}
	reportStats := func(i int, s *BackupProgressStats) {
		mu.Lock()
		stats[i] = *s
		total := BackupProgressStats{CurrentDir: s.CurrentDir, Message: s.Message}
		for _, st := range stats {
			total.BytesDone += st.BytesDone
			total.BytesTotal += st.BytesTotal
			total.NewChunks += st.NewChunks
			total.ReusedChunks += st.ReusedChunks
			total.FailedChunks += st.FailedChunks
		}
		sum := 0.0
		for _, f := range fractions {
			sum += f
		}
		total.Percent = sum / float64(len(dirs))
		mu.Unlock()
		opts.OnStats(&total)
	}

	runAll := func(snapshotPaths map[string]string, parallel int) {
		sem := make(chan struct{}, parallel)
		var wg sync.WaitGroup
		for i, dir := range dirs {
			sem <- struct{}{}
			if opts.Ctx != nil && opts.Ctx.Err() != nil {
				<-sem
				// A cancelled run must never be reported as a success: the
				// folders not started were not backed up.
				errMsg := fmt.Sprintf("backup cancelled: %d/%d folder(s) not backed up", len(dirs)-i, len(dirs))
				writeBackupLog("Cancellation requested — skipping remaining folders (" + errMsg + ")")
				mu.Lock()
				perDirErrors = append(perDirErrors, errMsg)
				mu.Unlock()
				break
			}
			mu.Lock()
			started++
			mu.Unlock()
			wg.Add(1)
			go func() {
				defer wg.Done()
				defer func() { <-sem }()

				dirOpts := opts
				dirOpts.BackupObjects = []string{dir}
				dirOpts.BackupID = GenerateBackupID(baseID, dir)
				dirOpts.OnComplete = nil // aggregated by the caller
				var dirStatus *BackupStatus
				dirOpts.OnResult = func(s *BackupStatus) { dirStatus = s }
				dirOpts.OnProgress = func(pct float64, msg string) { reportProgress(i, pct, msg) }
				if opts.OnStats != nil {
					dirOpts.OnStats = func(s *BackupProgressStats) { reportStats(i, s) }
				}
				dirOpts.lockHeld = true
				if snapshotPaths != nil {
					dirOpts.batchVSS = true
					dirOpts.snapshotPath = snapshotPaths[dir]
				}

				writeBackupLog(fmt.Sprintf("[Parallel] Backing up %s as its own group %s", dir, dirOpts.BackupID))
				derr := runBackupInlineInternal(dirOpts)

				mu.Lock()
				defer mu.Unlock()
				if dirStatus != nil {
					agg.merge(dirStatus)
				} else if derr != nil {
					agg.Outcome = OutcomeFailed
				}
				if derr != nil {
					errMsg := fmt.Sprintf("backup of %s failed: %v", dir, derr)
					writeBackupLog(errMsg)
					perDirErrors = append(perDirErrors, errMsg)
				}
			}()
		}
		wg.Wait()
	}

	if opts.UseVSS {
		called := false
		err := snapshot.CreateVSSSnapshot(dirs, true, func(snaps map[string]snapshot.SnapShot) error {
			called = true
			paths := make(map[string]string, len(dirs))
			for _, dir := range dirs {
				if s, ok := snaps[dir]; ok {
					paths[dir] = s.FullPath
				} else if abs, aerr := filepath.Abs(dir); aerr == nil {
					if s, ok := snaps[abs]; ok {
						paths[dir] = s.FullPath
					}
				}
			}
			runAll(paths, parallel)
			if len(perDirErrors) > 0 {
				// Lets the snapshot code delete the shadow copies right away
				// (Windows only does it on an error); already reported.
				return errFoldersFailed
			}
			return nil
		})
		if err != nil && !called {
			// The set could not be taken (e.g. one volume refuses VSS): fall
			// back to one snapshot per folder, one folder at a time, so the
			// folders on healthy volumes are still backed up. Never in
			// parallel: concurrent snapshot creations collide.
			writeBackupLog(fmt.Sprintf("[VSS] One snapshot for every folder failed (%v): falling back to one snapshot per folder, one folder at a time", err))
			runAll(nil, 1)
		} else if err != nil && !errors.Is(err, errFoldersFailed) {
			// Releasing the snapshot failed: never a success, even when every
			// folder uploaded.
			errMsg := fmt.Sprintf("VSS snapshot of the folders failed: %v", err)
			writeBackupLog(errMsg)
			perDirErrors = append(perDirErrors, errMsg)
			if started == 0 {
				agg.Outcome = OutcomeFailed
			} else if outcomeRank(agg.Outcome) > outcomeRank(OutcomePartial) {
				agg.Outcome = OutcomePartial
			}
		}
	} else {
		runAll(nil, parallel)
	}

	// Cancelled or VSS-failed before every folder ran: never a success.
	if len(perDirErrors) > 0 && started < len(dirs) {
		if started == 0 {
			agg.Outcome = OutcomeFailed
		} else if outcomeRank(agg.Outcome) > outcomeRank(OutcomePartial) {
			agg.Outcome = OutcomePartial
		}
	}
	return perDirErrors
}

// runBackupInlineInternal is the actual backup implementation (called by RunBackupInline)
func runBackupInlineInternal(opts BackupOptions) (returnErr error) {
	// CRITICAL: Panic recovery for split jobs
	defer func() {
		if r := recover(); r != nil {
			errMsg := fmt.Sprintf("CRITICAL: Backup panic in runBackupInlineInternal: %v", r)
			writeBackupLog(errMsg)
			// Get stack trace
			buf := make([]byte, 4096)
			n := runtime.Stack(buf, false)
			writeBackupLog(fmt.Sprintf("Stack trace:\n%s", buf[:n]))
			returnErr = fmt.Errorf("backup panic: %v", r)
		}
	}()

	// Handle machine backup type (new Kind field)
	if opts.Kind == "machine" {
		return runMachineBackupInline(opts)
	}

	// Default to directory backup for "disk" or "directory" kinds
	// (existing logic handles these cases)

	startTime := time.Now()

	// Acquire backup lock for this destination to prevent concurrent backups.
	// A folder of a parallel batch runs under the batch's lock.
	if !opts.lockHeld {
		backupLock := getBackupLock(opts.BaseURL, opts.Datastore)
		writeBackupLog(fmt.Sprintf("[Backup Lock] Waiting for lock on %s/%s (prevents concurrent backups)", opts.BaseURL, opts.Datastore))

		// Notify that we're waiting if OnProgress is set
		if opts.OnProgress != nil {
			opts.OnProgress(0, "Waiting for previous backup to complete...")
		}

		backupLock.Lock()
		writeBackupLog(fmt.Sprintf("[Backup Lock] ✓ Lock acquired for %s/%s - starting backup", opts.BaseURL, opts.Datastore))
		defer func() {
			backupLock.Unlock()
			writeBackupLog(fmt.Sprintf("[Backup Lock] ✓ Lock released for %s/%s", opts.BaseURL, opts.Datastore))
		}()
	}

	// Generate backup ID from path if not specified
	writeBackupLog("[DEBUG] Generating backup ID if needed")
	if opts.BackupID == "" {
		hostname, err := os.Hostname()
		if err != nil {
			hostname = "unnamed-backup"
		}

		// Generate backup-id from first directory path: hostname_DRIVE_PATH
		if len(opts.BackupObjects) > 0 {
			opts.BackupID = GenerateBackupID(hostname, opts.BackupObjects[0])
		} else {
			opts.BackupID = hostname
		}
	}
	writeBackupLog(fmt.Sprintf("[DEBUG] BackupID set to: %s", opts.BackupID))

	// Default to "host" type for directory backups
	if opts.BackupType == "" {
		opts.BackupType = "host"
	}

	// Progress callback wrapper
	progress := func(pct float64, msg string) {
		writeBackupLog(fmt.Sprintf("Backup progress: %.1f%% - %s", pct*100, msg))
		if opts.OnProgress != nil {
			opts.OnProgress(pct, msg)
		}
	}

	// Check if all backup directories exist
	writeBackupLog(fmt.Sprintf("[DEBUG] Checking %d backup directories exist", len(opts.BackupObjects)))
	for idx, dir := range opts.BackupObjects {
		writeBackupLog(fmt.Sprintf("[DEBUG] Checking directory %d/%d: %s", idx+1, len(opts.BackupObjects), dir))
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			errMsg := fmt.Sprintf("Backup directory does not exist: %s", dir)
			writeBackupLog(errMsg)
			if opts.OnComplete != nil {
				opts.OnComplete(false, errMsg)
			}
			if opts.OnResult != nil {
				opts.OnResult(&BackupStatus{
					Outcome:     OutcomeFailed,
					BackupID:    opts.BackupID,
					DurationSec: time.Since(startTime).Seconds(),
					Message:     errMsg,
				})
			}
			return fmt.Errorf("%s", errMsg)
		}
	}
	writeBackupLog("[DEBUG] All directories checked, calling progress(0.05)")

	progress(0.05, "Connecting to PBS...")
	writeBackupLog("[DEBUG] After progress(0.05), before connection log")

	client := newDirBackupClient(opts)

	writeBackupLog("[DEBUG] PBS client created, starting directory backup loop")

	// Backup each directory
	run := &dirRun{}
	newchunk, reusechunk, failedchunk, totalSize := &run.newchunk, &run.reusechunk, &run.failedchunk, &run.totalSize

	for idx, dir := range opts.BackupObjects {
		if opts.Ctx != nil && opts.Ctx.Err() != nil {
			writeBackupLog(fmt.Sprintf("Cancellation requested — stopping before backup of %s", dir))
			if opts.OnComplete != nil {
				opts.OnComplete(false, "Backup cancelled by user")
			}
			return machinebackuplib.ErrCancelled
		}
		writeBackupLog(fmt.Sprintf("Starting backup of directory %d/%d: %s", idx+1, len(opts.BackupObjects), dir))

		// Each directory becomes its own PBS session (Connect → upload → Finish).
		// On session-lost we wait for PBS to release the group lock, then retry once.
		// PBS dedupes chunks, so retrying is cheap for the already-uploaded data.
		var err error
		var dirBytes uint64
		for attempt := 1; attempt <= maxDirAttempts; attempt++ {
			switch {
			case opts.batchVSS && opts.snapshotPath == "":
				err = fmt.Errorf("no VSS snapshot was taken for %s", dir)
			case opts.snapshotPath != "":
				// Folder of a multi-folder batch: read the batch's snapshot.
				dirBytes, err = backupReal(opts.Ctx, client, newchunk, reusechunk, failedchunk, opts.snapshotPath, dir, true, progress, opts.OnStats, opts.ExcludeList)
			default:
				dirBytes, err = backupDirectory(opts.Ctx, client, newchunk, reusechunk, failedchunk, dir, opts.UseVSS, progress, opts.OnStats, opts.ExcludeList)
			}
			if err == nil {
				break
			}
			// Tear down the abandoned PBS session before retrying or moving to the
			// next directory: leaving it open holds the backup-group lock (~16 min
			// until TCP keepalive reaps it) so every later directory in this run
			// then fails with "locked backup group". [B-4]
			client.Close()
			if !isFatalSessionError(err) {
				// Non-recoverable error (e.g. access denied) — don't retry
				break
			}
			if attempt < maxDirAttempts {
				writeBackupLog(fmt.Sprintf("Directory %s: session lost on attempt %d/%d: %v",
					dir, attempt, maxDirAttempts, err))
				if !waitForGroupLock(opts, progress) {
					return machinebackuplib.ErrCancelled
				}
				writeBackupLog(fmt.Sprintf("Wait complete, retrying directory %s with fresh connection", dir))
			}
		}

		// Cancelled in the middle of this directory: drop its session (already
		// done above on an error) instead of finishing it, so nothing of it is
		// committed. Stop here like a cancel between two directories does.
		if opts.Ctx != nil && opts.Ctx.Err() != nil {
			if err == nil {
				client.Close()
			}
			writeBackupLog(fmt.Sprintf("Cancellation requested during backup of %s — stopped, not committed", dir))
			if opts.OnComplete != nil {
				opts.OnComplete(false, "Backup cancelled by user")
			}
			return machinebackuplib.ErrCancelled
		}

		// Capture this directory's lists before the next directory's Connect() resets
		// them on the shared client (v2-H-02 / M-01).
		run.collect(client)

		if err != nil {
			run.fail(dir, fmt.Sprintf("Backup failed for %s: %v", dir, err), err)
			continue // Don't abort — try remaining directories
		}

		// Finalize this directory's PBS session immediately so it's committed on the server
		finishCfg := retry.DefaultConfig()
		finishCfg.MaxAttempts = 5
		finishCtx, finishCancel := context.WithTimeout(context.Background(), 5*time.Minute)
		finishErr := retry.DoWithJitter(finishCtx, finishCfg, retry.DefaultRetryable, func() error {
			return client.Finish()
		})
		finishCancel()
		if finishErr != nil {
			client.Close() // release the session/group lock on a failed finalize too [B-4]
			run.fail(dir, fmt.Sprintf("Failed to finalize backup for %s: %v", dir, finishErr), finishErr)
			continue
		}
		writeBackupLog(fmt.Sprintf("Directory %d/%d finalized: %s", idx+1, len(opts.BackupObjects), dir))
		// Accumulate the real archived byte count so TotalBytes / "X MB backed up"
		// and the [RESULT] support line are no longer always zero. [B-3]
		totalSize.Add(dirBytes)
		run.succeed(dir)
	}

	return run.report(opts, client.Manifest.BackupTime, startTime, progress)
}

// runMachineBackupInline handles machine backup using machinebackuplib
func runMachineBackupInline(opts BackupOptions) error {
	startTime := time.Now()

	// Create machine backup config from opts
	cfg := &machinebackuplib.Config{
		BaseURL:         opts.BaseURL,
		CertFingerprint: opts.CertFingerprint,
		AuthID:          opts.AuthID,
		Secret:          opts.Secret,
		Ticket:          opts.Ticket,
		CSRFToken:       opts.CSRFToken,
		Datastore:       opts.Datastore,
		Namespace:       opts.Namespace,
		BackupID:        opts.BackupID,
		BackupType:      opts.BackupType,
		BackupDevices:   opts.BackupObjects,
		Crypt:           opts.Crypt,
		UseSnapshot:     opts.UseVSS,
	}

	// Progress callback wrapper. Returning true (user pressed Stop, which
	// cancels opts.Ctx) makes the backup abort without committing the index.
	progress := func(pct float64, msg string) bool {
		writeBackupLog(fmt.Sprintf("Backup progress: %.1f%% - %s", pct*100, msg))
		if opts.OnProgress != nil {
			opts.OnProgress(pct, msg)
		}
		return opts.Ctx != nil && opts.Ctx.Err() != nil
	}

	// Perform machine backup
	_, err := machinebackuplib.Backup(cfg, progress)
	if err != nil {
		// A user Stop is not a failure: report it as a cancellation, and keep
		// the sentinel intact so callers can match it with errors.Is instead
		// of string-parsing the message.
		cancelled := errors.Is(err, machinebackuplib.ErrCancelled) ||
			(opts.Ctx != nil && opts.Ctx.Err() != nil)
		errMsg := fmt.Sprintf("Machine backup failed: %v", err)
		if cancelled {
			errMsg = "Machine backup cancelled by user"
		}
		writeBackupLog(errMsg)

		if opts.OnComplete != nil {
			opts.OnComplete(false, errMsg)
		}
		if opts.OnResult != nil {
			opts.OnResult(&BackupStatus{
				Outcome:     OutcomeFailed,
				BackupID:    opts.BackupID,
				DurationSec: time.Since(startTime).Seconds(),
				Message:     errMsg,
			})
		}
		if cancelled {
			return machinebackuplib.ErrCancelled
		}
		return errors.New(errMsg)
	}

	// Success case
	duration := time.Since(startTime)
	completionMsg := fmt.Sprintf("Machine backup completed in %s", formatDuration(duration))

	if opts.OnComplete != nil {
		opts.OnComplete(true, completionMsg)
	}
	if opts.OnResult != nil {
		opts.OnResult(&BackupStatus{
			Outcome:     OutcomeVerifiedSuccess,
			BackupID:    opts.BackupID,
			DurationSec: duration.Seconds(),
			Message:     completionMsg,
		})
	}

	return nil
}

func backupDirectory(ctx context.Context, client *pbscommon.PBSClient, newchunk, reusechunk, failedchunk *atomic.Uint64, backupdir string, usevss bool, progress func(float64, string), onStats func(*BackupProgressStats), excludeList []string) (uint64, error) {
	writeBackupLog(fmt.Sprintf("Starting backup of %s", backupdir))
	originalPath := backupdir

	if usevss {
		var bytesArchived uint64
		err := snapshot.CreateVSSSnapshot([]string{backupdir}, true, func(snaps map[string]snapshot.SnapShot) error {
			for _, snap := range snaps {
				backupdir = snap.FullPath
				break
			}
			var e error
			bytesArchived, e = backupReal(ctx, client, newchunk, reusechunk, failedchunk, backupdir, originalPath, usevss, progress, onStats, excludeList)
			return e
		})
		return bytesArchived, err
	}

	return backupReal(ctx, client, newchunk, reusechunk, failedchunk, backupdir, originalPath, usevss, progress, onStats, excludeList)
}

func backupReal(cancelCtx context.Context, client *pbscommon.PBSClient, newchunk, reusechunk, failedchunk *atomic.Uint64, backupdir string, originalPath string, vssUsed bool, progress func(float64, string), onStats func(*BackupProgressStats), excludeList []string) (returnBytes uint64, returnErr error) {
	// Panic recovery - critical to prevent silent crashes during backup
	defer func() {
		if r := recover(); r != nil {
			errMsg := fmt.Sprintf("CRITICAL: Backup panic occurred: %v", r)
			writeBackupLog(errMsg)
			// Get stack trace
			buf := make([]byte, 4096)
			n := runtime.Stack(buf, false)
			writeBackupLog(fmt.Sprintf("Stack trace:\n%s", buf[:n]))
			returnErr = fmt.Errorf("backup panic: %v", r)
		}
	}()

	// One folder, one snapshot of the original layout: backup.pxar.didx.
	session, err := openSnapshotSession(cancelCtx, client, newchunk, reusechunk, failedchunk, excludeList)
	if err != nil {
		return 0, err
	}
	n, err := session.writeArchive(legacyArchiveName, backupdir, originalPath, vssUsed, progress, onStats)
	if err != nil {
		return 0, err
	}
	if err := session.close(); err != nil {
		return 0, err
	}
	return n, nil
}
