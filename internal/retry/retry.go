package retry

import (
	"context"
	"errors"
	"math"
	"math/rand"
	"time"
)

// Config represents the retry configuration
type Config struct {
	MaxRetries      int
	InitialInterval time.Duration
	MaxInterval     time.Duration
	Multiplier      float64
	Randomization   float64
}

// RetryableError represents an error that can be retried
type RetryableError struct {
	Err error
}

func (e *RetryableError) Error() string {
	return e.Err.Error()
}

// NewRetryableError creates a new retryable error
func NewRetryableError(err error) error {
	return &RetryableError{Err: err}
}

// DefaultConfig returns a default retry configuration
func DefaultConfig() Config {
	return Config{
		MaxRetries:      3,
		InitialInterval: 100 * time.Millisecond,
		MaxInterval:     1 * time.Second,
		Multiplier:      2.0,
		Randomization:   0.1,
	}
}

// Do executes the given function with retries
func Do(ctx context.Context, config *Config, fn func() error) error {
	var err error
	interval := config.InitialInterval

	for i := 0; i <= config.MaxRetries; i++ {
		// Execute the function
		err = fn()
		if err == nil {
			return nil
		}

		// Check if the error is retryable
		var retryableErr *RetryableError
		if !errors.As(err, &retryableErr) {
			return err
		}

		// Check if context is done
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Calculate next interval with jitter
		interval = time.Duration(float64(interval) * config.Multiplier)
		if interval > config.MaxInterval {
			interval = config.MaxInterval
		}

		// Add jitter
		jitter := float64(interval) * config.Randomization
		interval = interval + time.Duration(rand.Float64()*jitter)

		// Wait for the next retry
		select {
		case <-time.After(interval):
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return err
}

// ExponentialBackoff calculates the next retry interval using exponential backoff
func ExponentialBackoff(attempt int, config Config) time.Duration {
	// Calculate base interval
	interval := float64(config.InitialInterval) * math.Pow(config.Multiplier, float64(attempt))

	// Add jitter
	jitter := interval * config.Randomization
	interval += jitter

	// Ensure we don't exceed max interval
	if interval > float64(config.MaxInterval) {
		interval = float64(config.MaxInterval)
	}

	return time.Duration(interval)
}
