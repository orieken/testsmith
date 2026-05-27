package llm

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// RetryStrategy defines the parameters for exponential-backoff retries.
// Using a named strategy satisfies the architecture guardrail against raw retry
// loops: callers instantiate a strategy and delegate; they do not write loops.
type RetryStrategy struct {
	// MaxAttempts is the total number of attempts (first call + retries).
	// A value of 1 means no retries.
	MaxAttempts int
	// BaseDelay is the wait before the second attempt.
	BaseDelay time.Duration
	// MaxDelay caps the computed delay at each step.
	MaxDelay time.Duration
	// Multiplier is the growth factor applied per attempt (default 2.0).
	Multiplier float64
}

// DefaultRetryStrategy returns a reasonable production strategy:
// 3 attempts, starting at 500 ms, capped at 30 s, doubling each time.
func DefaultRetryStrategy() RetryStrategy {
	return RetryStrategy{
		MaxAttempts: 3,
		BaseDelay:   500 * time.Millisecond,
		MaxDelay:    30 * time.Second,
		Multiplier:  2.0,
	}
}

// delayFor returns the backoff duration for a given attempt index (0-based).
func (s RetryStrategy) delayFor(attempt int) time.Duration {
	multiplier := s.Multiplier
	if multiplier <= 0 {
		multiplier = 2.0
	}
	d := float64(s.BaseDelay) * math.Pow(multiplier, float64(attempt))
	if d > float64(s.MaxDelay) {
		d = float64(s.MaxDelay)
	}
	return time.Duration(d)
}

// retryProvider wraps a Provider and retries transient failures using the
// configured RetryStrategy. It is the only place in the codebase that contains
// a retry loop — all other callers delegate here.
type retryProvider struct {
	inner    Provider
	strategy RetryStrategy
}

// WithRetry wraps p with exponential-backoff retry logic for transient HTTP
// errors (429, 500, 502, 503, 504). Non-retryable errors are returned immediately.
func WithRetry(p Provider, s RetryStrategy) Provider {
	if s.MaxAttempts <= 1 {
		return p // no-op wrapper when retries are disabled
	}
	return &retryProvider{inner: p, strategy: s}
}

func (r *retryProvider) Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error) {
	var lastErr error
	for attempt := 0; attempt < r.strategy.MaxAttempts; attempt++ {
		if attempt > 0 {
			delay := r.strategy.delayFor(attempt - 1)
			// Honour a Retry-After hint embedded in the error message when present.
			if retryAfter := parseRetryAfter(lastErr); retryAfter > 0 {
				delay = retryAfter
			}
			select {
			case <-ctx.Done():
				return CompletionResponse{}, ctx.Err()
			case <-time.After(delay):
			}
		}

		resp, err := r.inner.Complete(ctx, req)
		if err == nil {
			return resp, nil
		}

		lastErr = err
		if !isRetryable(err) {
			return CompletionResponse{}, err
		}
	}
	return CompletionResponse{}, fmt.Errorf("all %d attempts failed: %w", r.strategy.MaxAttempts, lastErr)
}

// isRetryable returns true for transient HTTP status codes worth retrying.
func isRetryable(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	for _, code := range []string{"429", "500", "502", "503", "504"} {
		if strings.Contains(msg, "API error "+code) ||
			strings.Contains(msg, "error "+code) {
			return true
		}
	}
	return false
}

// parseRetryAfter extracts a Retry-After seconds value that some providers
// embed in the error message as "retry-after: N".
func parseRetryAfter(err error) time.Duration {
	if err == nil {
		return 0
	}
	msg := strings.ToLower(err.Error())
	idx := strings.Index(msg, "retry-after:")
	if idx < 0 {
		return 0
	}
	rest := strings.TrimSpace(msg[idx+len("retry-after:"):])
	// Value may be a number of seconds or an HTTP-date; handle seconds only.
	fields := strings.Fields(rest)
	if len(fields) == 0 {
		return 0
	}
	secs, parseErr := strconv.Atoi(fields[0])
	if parseErr != nil || secs <= 0 {
		return 0
	}
	max := int(30 * time.Second / time.Second)
	if secs > max {
		secs = max
	}
	return time.Duration(secs) * time.Second
}
