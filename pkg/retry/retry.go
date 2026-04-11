package retry

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"
)

// ErrMaxRetriesExceeded is returned when all retry attempts have been exhausted
var ErrMaxRetriesExceeded = errors.New("maximum retry attempts exceeded")

// Config holds retry configuration
type Config struct {
	MaxAttempts int           // Maximum number of retry attempts
	InitialDelay time.Duration // Initial delay between retries
	MaxDelay    time.Duration // Maximum delay between retries
	Multiplier  float64       // Backoff multiplier (e.g., 2.0 for exponential)
}

// DefaultConfig returns a sensible default retry configuration
func DefaultConfig() Config {
	return Config{
		MaxAttempts:  3,
		InitialDelay: 1 * time.Second,
		MaxDelay:     30 * time.Second,
		Multiplier:   2.0,
	}
}

// IsRetryable determines if an error should trigger a retry
type IsRetryable func(error) bool

// DefaultRetryable retries on network and temporary errors
func DefaultRetryable(err error) bool {
	if err == nil {
		return false
	}
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "connection reset") ||
		strings.Contains(errStr, "temporary failure") ||
		strings.Contains(errStr, "unexpected eof") ||
		strings.Contains(errStr, "broken pipe") ||
		strings.Contains(errStr, "i/o timeout")
}

// Do executes fn with retry logic
func Do(ctx context.Context, cfg Config, isRetryable IsRetryable, fn func() error) error {
	if isRetryable == nil {
		isRetryable = DefaultRetryable
	}

	var lastErr error
	delay := cfg.InitialDelay

	for attempt := 0; attempt < cfg.MaxAttempts; attempt++ {
		// Check context before attempting
		select {
		case <-ctx.Done():
			return fmt.Errorf("context cancelled: %w", ctx.Err())
		default:
		}

		// Execute function
		err := fn()
		if err == nil {
			return nil // Success!
		}

		lastErr = err

		// Check if error is retryable
		if !isRetryable(err) {
			return fmt.Errorf("non-retryable error: %w", err)
		}

		// Last attempt, don't wait
		if attempt == cfg.MaxAttempts-1 {
			break
		}

		// Wait with exponential backoff
		select {
		case <-ctx.Done():
			return fmt.Errorf("context cancelled during retry wait: %w", ctx.Err())
		case <-time.After(delay):
			// Calculate next delay with exponential backoff
			delay = time.Duration(float64(delay) * cfg.Multiplier)
			if delay > cfg.MaxDelay {
				delay = cfg.MaxDelay
			}
		}
	}

	return fmt.Errorf("%w after %d attempts: %v", ErrMaxRetriesExceeded, cfg.MaxAttempts, lastErr)
}

// DoWithJitter adds jitter to retry delays to prevent thundering herd
func DoWithJitter(ctx context.Context, cfg Config, isRetryable IsRetryable, fn func() error) error {
	originalDelay := cfg.InitialDelay

	// Add up to 20% jitter
	jitter := float64(originalDelay) * 0.2
	cfg.InitialDelay = time.Duration(float64(originalDelay) + (math.Mod(float64(time.Now().UnixNano()), jitter)))

	return Do(ctx, cfg, isRetryable, fn)
}
