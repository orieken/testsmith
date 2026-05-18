package llm

import (
	"crypto/sha256"
	"fmt"
	"sync"

	"github.com/orieken/testsmith/internal/domain"
)

// ResultCache is a thread-safe in-process cache for LLM body generation results.
// Cache keys are derived from the request content — language, member identity,
// source code, and framework config — so results are automatically invalidated
// when any of those change (e.g. between watch-mode saves).
//
// The cache is intentionally in-memory and per-run. Cross-run persistence would
// require a file-backed store and is a future enhancement.
type ResultCache struct {
	mu    sync.RWMutex
	store map[string][]domain.BodyGenResult
	hits  int
	misses int
}

// NewResultCache returns an empty, ready-to-use ResultCache.
func NewResultCache() *ResultCache {
	return &ResultCache{store: make(map[string][]domain.BodyGenResult)}
}

// Get returns cached results and true when a matching entry exists.
func (c *ResultCache) Get(key string) ([]domain.BodyGenResult, bool) {
	c.mu.RLock()
	v, ok := c.store[key]
	c.mu.RUnlock()
	if ok {
		c.mu.Lock()
		c.hits++
		c.mu.Unlock()
	} else {
		c.mu.Lock()
		c.misses++
		c.mu.Unlock()
	}
	return v, ok
}

// Set stores results under key, overwriting any existing entry.
func (c *ResultCache) Set(key string, results []domain.BodyGenResult) {
	c.mu.Lock()
	c.store[key] = results
	c.mu.Unlock()
}

// Stats returns (hits, misses, size) for observability / verbose logging.
func (c *ResultCache) Stats() (hits, misses, size int) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.hits, c.misses, len(c.store)
}

// CacheKey derives a stable cache key from the parts of a BodyGenRequest that
// determine the LLM output. Fields that only affect routing (context.Context,
// Extra map ordering) are excluded deliberately.
func CacheKey(req domain.BodyGenRequest) string {
	h := sha256.New()
	// Write each deterministic field separated by a null byte to prevent
	// accidental collisions between e.g. ("ab", "c") and ("a", "bc").
	for _, s := range []string{
		req.Language,
		string(req.MemberKind),
		req.MemberName,
		req.SourceCode,
		req.ModulePath,
		req.DepsSignatures,
		req.ExistingTestSnippet,
		req.Framework.Name,
		req.ProjectKnowledge,
	} {
		_, _ = fmt.Fprintf(h, "%s\x00", s)
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}
