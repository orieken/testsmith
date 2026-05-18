package llm

import "context"

// semaphoreProvider wraps a Provider and limits the number of concurrent
// in-flight calls using a buffered channel as a counting semaphore.
// This prevents burst traffic to the LLM API when many files are processed
// in parallel (--all with multiple workers × goroutine fan-out per file).
type semaphoreProvider struct {
	inner Provider
	sem   chan struct{}
}

// WithSemaphore wraps p so that at most maxConcurrent calls are in-flight at
// the same time. Callers block until a slot is available or ctx is cancelled.
// Pass maxConcurrent ≤ 0 to return p unwrapped (no limit).
func WithSemaphore(p Provider, maxConcurrent int) Provider {
	if maxConcurrent <= 0 {
		return p
	}
	return &semaphoreProvider{
		inner: p,
		sem:   make(chan struct{}, maxConcurrent),
	}
}

func (s *semaphoreProvider) Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error) {
	// Acquire slot.
	select {
	case s.sem <- struct{}{}:
	case <-ctx.Done():
		return CompletionResponse{}, ctx.Err()
	}
	// Release slot when done.
	defer func() { <-s.sem }()

	return s.inner.Complete(ctx, req)
}
