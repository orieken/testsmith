package llm

import (
	"context"
	"errors"
	"testing"
	"time"
)

// ── helpers ───────────────────────────────────────────────────────────────────

type countProv struct {
	calls   int
	failFor int // fail on the first N calls with a retryable error
}

func (c *countProv) Complete(_ context.Context, _ CompletionRequest) (CompletionResponse, error) {
	c.calls++
	if c.calls <= c.failFor {
		return CompletionResponse{}, errors.New("API error 500: internal server error")
	}
	return CompletionResponse{Content: "ok", TokensUsed: 1}, nil
}

type alwaysFailProv struct{ code string }

func (a *alwaysFailProv) Complete(_ context.Context, _ CompletionRequest) (CompletionResponse, error) {
	return CompletionResponse{}, errors.New(a.code)
}

// ── DefaultRetryStrategy ─────────────────────────────────────────────────────

func TestDefaultRetryStrategy(t *testing.T) {
	t.Parallel()
	s := DefaultRetryStrategy()
	if s.MaxAttempts != 3 {
		t.Errorf("MaxAttempts = %d, want 3", s.MaxAttempts)
	}
	if s.BaseDelay != 500*time.Millisecond {
		t.Errorf("BaseDelay = %v, want 500ms", s.BaseDelay)
	}
	if s.MaxDelay != 30*time.Second {
		t.Errorf("MaxDelay = %v, want 30s", s.MaxDelay)
	}
	if s.Multiplier != 2.0 {
		t.Errorf("Multiplier = %f, want 2.0", s.Multiplier)
	}
}

// ── delayFor ─────────────────────────────────────────────────────────────────

func TestDelayFor_ExponentialGrowthAndCap(t *testing.T) {
	t.Parallel()
	s := RetryStrategy{BaseDelay: 100 * time.Millisecond, MaxDelay: 500 * time.Millisecond, Multiplier: 2.0}

	tests := []struct {
		attempt int
		want    time.Duration
	}{
		{0, 100 * time.Millisecond}, // 100 * 2^0 = 100
		{1, 200 * time.Millisecond}, // 100 * 2^1 = 200
		{2, 400 * time.Millisecond}, // 100 * 2^2 = 400
		{3, 500 * time.Millisecond}, // 100 * 2^3 = 800 → capped at 500
	}
	for _, tt := range tests {
		if got := s.delayFor(tt.attempt); got != tt.want {
			t.Errorf("delayFor(%d) = %v, want %v", tt.attempt, got, tt.want)
		}
	}
}

func TestDelayFor_ZeroMultiplierFallsBackToTwo(t *testing.T) {
	t.Parallel()
	s := RetryStrategy{BaseDelay: 100 * time.Millisecond, MaxDelay: time.Minute, Multiplier: 0}
	// fallback multiplier 2.0: 100ms * 2^1 = 200ms
	if got := s.delayFor(1); got != 200*time.Millisecond {
		t.Errorf("delayFor(1) with zero multiplier = %v, want 200ms", got)
	}
}

// ── isRetryable ──────────────────────────────────────────────────────────────

func TestIsRetryable(t *testing.T) {
	t.Parallel()
	tests := []struct {
		msg  string
		want bool
	}{
		{"API error 429: rate limited", true},
		{"API error 500: internal", true},
		{"error 502: bad gateway", true},
		{"error 503: service unavailable", true},
		{"API error 504: gateway timeout", true},
		{"API error 401: unauthorized", false},
		{"API error 403: forbidden", false},
		{"API error 404: not found", false},
		{"connection refused", false},
	}
	for _, tt := range tests {
		if got := isRetryable(errors.New(tt.msg)); got != tt.want {
			t.Errorf("isRetryable(%q) = %v, want %v", tt.msg, got, tt.want)
		}
	}
	if isRetryable(nil) {
		t.Error("isRetryable(nil) = true, want false")
	}
}

// ── parseRetryAfter ──────────────────────────────────────────────────────────

func TestParseRetryAfter(t *testing.T) {
	t.Parallel()
	tests := []struct {
		msg  string
		want time.Duration
	}{
		{"rate limit retry-after: 5", 5 * time.Second},
		{"retry-after: 30", 30 * time.Second},
		{"RETRY-AFTER: 10 extra ignored", 10 * time.Second},
		{"retry-after: 999999", 30 * time.Second}, // capped at 30 s
		{"retry-after: abc", 0},
		{"retry-after: 0", 0},
		{"retry-after: -1", 0},
		{"no header here", 0},
	}
	for _, tt := range tests {
		got := parseRetryAfter(errors.New(tt.msg))
		if got != tt.want {
			t.Errorf("parseRetryAfter(%q) = %v, want %v", tt.msg, got, tt.want)
		}
	}
	if parseRetryAfter(nil) != 0 {
		t.Error("parseRetryAfter(nil) should return 0")
	}
}

// ── WithRetry ────────────────────────────────────────────────────────────────

func TestWithRetry_MaxAttemptsOneReturnsInner(t *testing.T) {
	t.Parallel()
	inner := &countProv{}
	wrapped := WithRetry(inner, RetryStrategy{MaxAttempts: 1})
	// WithRetry must return the inner provider unwrapped for MaxAttempts <= 1.
	if wrapped != inner {
		t.Error("WithRetry(MaxAttempts=1) should return the inner provider unchanged")
	}
}

func TestWithRetry_SucceedsOnSecondAttempt(t *testing.T) {
	t.Parallel()
	inner := &countProv{failFor: 1}
	s := RetryStrategy{MaxAttempts: 3, BaseDelay: 0, MaxDelay: 0, Multiplier: 2.0}
	wrapped := WithRetry(inner, s)

	_, err := wrapped.Complete(context.Background(), CompletionRequest{})
	if err != nil {
		t.Fatalf("expected success on second attempt, got: %v", err)
	}
	if inner.calls != 2 {
		t.Errorf("expected 2 calls, got %d", inner.calls)
	}
}

func TestWithRetry_GivesUpAfterMaxAttempts(t *testing.T) {
	t.Parallel()
	inner := &countProv{failFor: 99}
	s := RetryStrategy{MaxAttempts: 3, BaseDelay: 0, MaxDelay: 0, Multiplier: 2.0}
	wrapped := WithRetry(inner, s)

	_, err := wrapped.Complete(context.Background(), CompletionRequest{})
	if err == nil {
		t.Fatal("expected error after exhausting retries, got nil")
	}
	if inner.calls != 3 {
		t.Errorf("expected exactly 3 calls, got %d", inner.calls)
	}
}

func TestWithRetry_DoesNotRetryNonRetryableError(t *testing.T) {
	t.Parallel()
	inner := &alwaysFailProv{code: "API error 401: unauthorized"}
	s := RetryStrategy{MaxAttempts: 3, BaseDelay: 0, MaxDelay: 0, Multiplier: 2.0}
	wrapped := WithRetry(inner, s)

	_, err := wrapped.Complete(context.Background(), CompletionRequest{})
	if err == nil {
		t.Fatal("expected error for non-retryable, got nil")
	}
}

func TestWithRetry_ContextCancelledDuringWait(t *testing.T) {
	t.Parallel()
	// Set a long delay so the test exits via context cancellation, not timer.
	inner := &countProv{failFor: 99}
	s := RetryStrategy{MaxAttempts: 3, BaseDelay: time.Hour, MaxDelay: time.Hour, Multiplier: 1.0}
	wrapped := WithRetry(inner, s)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before the first retry delay

	_, err := wrapped.Complete(ctx, CompletionRequest{})
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

// ── WithSemaphore ─────────────────────────────────────────────────────────────

func TestWithSemaphore_ZeroOrNegativeReturnsInner(t *testing.T) {
	t.Parallel()
	inner := &countProv{}
	for _, n := range []int{0, -1, -100} {
		if wrapped := WithSemaphore(inner, n); wrapped != inner {
			t.Errorf("WithSemaphore(%d) should return inner unwrapped", n)
		}
	}
}

func TestWithSemaphore_AllowsCallsWithinLimit(t *testing.T) {
	t.Parallel()
	inner := &countProv{}
	wrapped := WithSemaphore(inner, 2)

	for i := 0; i < 5; i++ {
		if _, err := wrapped.Complete(context.Background(), CompletionRequest{}); err != nil {
			t.Fatalf("call %d: unexpected error: %v", i, err)
		}
	}
	if inner.calls != 5 {
		t.Errorf("expected 5 calls through semaphore, got %d", inner.calls)
	}
}

func TestWithSemaphore_ContextCancelledWhileWaiting(t *testing.T) {
	t.Parallel()
	inner := &countProv{}
	// Build a semaphore with 1 slot and pre-fill it to simulate a saturated semaphore.
	sp := &semaphoreProvider{inner: inner, sem: make(chan struct{}, 1)}
	sp.sem <- struct{}{} // occupy the only slot

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := sp.Complete(ctx, CompletionRequest{})
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled while waiting for semaphore slot, got %v", err)
	}
	// The pre-filled slot must still be in the channel (we never acquired it).
	if len(sp.sem) != 1 {
		t.Error("semaphore slot should still be filled after cancelled call")
	}
}
