package llm

import (
	"context"
	"strings"
	"testing"

	"github.com/orieken/testsmith/internal/domain"
)

// ── ResultCache.Stats ─────────────────────────────────────────────────────────

func TestResultCache_Stats(t *testing.T) {
	t.Parallel()
	c := NewResultCache()

	// Initial state.
	h, m, sz := c.Stats()
	if h != 0 || m != 0 || sz != 0 {
		t.Errorf("new cache: want (0,0,0) got (%d,%d,%d)", h, m, sz)
	}

	// Miss.
	c.Get("no-such-key")
	_, _, _ = c.Stats()
	_, m, _ = c.Stats()
	if m < 1 {
		t.Error("expected at least 1 miss after Get of absent key")
	}

	// Set then hit.
	c.Set("k", []domain.BodyGenResult{{MemberName: "fn"}})
	c.Get("k")
	h, _, sz = c.Stats()
	if h < 1 {
		t.Error("expected at least 1 hit after Get of present key")
	}
	if sz != 1 {
		t.Errorf("cache size = %d, want 1", sz)
	}
}

// ── CacheKey ─────────────────────────────────────────────────────────────────

func TestCacheKey_Deterministic(t *testing.T) {
	t.Parallel()
	req := domain.BodyGenRequest{
		Language:   "go",
		MemberName: "Foo",
		MemberKind: domain.KindFunction,
		SourceCode: "func Foo() {}",
	}
	k1 := CacheKey(req)
	k2 := CacheKey(req)
	if k1 != k2 {
		t.Error("CacheKey must be deterministic for identical requests")
	}
}

func TestCacheKey_DifferentOnFieldChange(t *testing.T) {
	t.Parallel()
	base := domain.BodyGenRequest{Language: "go", MemberName: "Foo", SourceCode: "func Foo() {}"}
	modified := base
	modified.MemberName = "Bar"
	if CacheKey(base) == CacheKey(modified) {
		t.Error("CacheKey should differ when MemberName changes")
	}
}

// ── buildBatchPrompt ──────────────────────────────────────────────────────────

func TestBuildBatchPrompt_ContainsExpectedSections(t *testing.T) {
	t.Parallel()
	d := batchPromptData{
		Language:   "go",
		Framework:  "testing",
		ModulePath: "mymod/util",
		SourceCode: "func Add(a, b int) int { return a + b }",
		MemberList: "1. function `Add`\n",
	}
	prompt := buildBatchPrompt(d)
	for _, want := range []string{"mymod/util", "Add", "go", "testing"} {
		if !strings.Contains(prompt, want) {
			t.Errorf("batch prompt missing %q:\n%s", want, prompt)
		}
	}
}

func TestBuildBatchPrompt_JSONOutput(t *testing.T) {
	t.Parallel()
	d := batchPromptData{UseJSONOutput: true, MemberList: "1. function `X`\n"}
	prompt := buildBatchPrompt(d)
	if !strings.Contains(prompt, `"tests"`) {
		t.Errorf("JSON output prompt should reference 'tests' schema, got:\n%s", prompt)
	}
}

func TestBuildBatchPrompt_DelimiterOutput(t *testing.T) {
	t.Parallel()
	d := batchPromptData{UseJSONOutput: false, MemberList: "1. function `X`\n"}
	prompt := buildBatchPrompt(d)
	if !strings.Contains(prompt, "<!-- TEST:") {
		t.Errorf("delimiter output prompt should reference <!-- TEST: marker, got:\n%s", prompt)
	}
}

func TestBuildBatchPrompt_IncludesOptionalFields(t *testing.T) {
	t.Parallel()
	d := batchPromptData{
		DepsSignatures:      "func Dep() {}",
		ExistingTestSnippet: "func TestOld() {}",
		MockLibrary:         "testify/mock",
		AssertStyle:         "require",
		MemberList:          "1. function `X`\n",
	}
	prompt := buildBatchPrompt(d)
	for _, want := range []string{"Dep", "TestOld", "testify/mock", "require"} {
		if !strings.Contains(prompt, want) {
			t.Errorf("batch prompt missing optional field %q", want)
		}
	}
}

// ── parseBatchResponse ────────────────────────────────────────────────────────

func TestParseBatchResponse_ExtractsBlocks(t *testing.T) {
	t.Parallel()
	response := `<!-- TEST: Foo -->
` + "```go" + `
func TestFoo(t *testing.T) { assert.True(t, true) }
` + "```" + `
<!-- TEST: Bar -->
` + "```go" + `
func TestBar(t *testing.T) {}
` + "```"

	reqs := []domain.BodyGenRequest{
		{MemberName: "Foo"},
		{MemberName: "Bar"},
	}
	results := parseBatchResponse(response, reqs)
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if len(results[0].CodeLines) == 0 {
		t.Error("Foo block not extracted")
	}
	if len(results[1].CodeLines) == 0 {
		t.Error("Bar block not extracted")
	}
}

func TestParseBatchResponse_MissingBlock_NilLines(t *testing.T) {
	t.Parallel()
	reqs := []domain.BodyGenRequest{{MemberName: "Missing"}}
	results := parseBatchResponse("no blocks here", reqs)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].CodeLines != nil {
		t.Error("absent block should produce nil CodeLines")
	}
}

// ── parseBatchJSON ────────────────────────────────────────────────────────────

func TestParseBatchJSON_ValidJSON(t *testing.T) {
	t.Parallel()
	json := `{"tests":[{"name":"Foo","code":"func TestFoo() {}"},{"name":"Bar","code":"func TestBar() {}"}]}`
	reqs := []domain.BodyGenRequest{{MemberName: "Foo"}, {MemberName: "Bar"}}
	results := parseBatchJSON(json, reqs)
	if results == nil {
		t.Fatal("parseBatchJSON returned nil for valid JSON")
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if len(results[0].CodeLines) == 0 {
		t.Error("Foo code lines should not be empty")
	}
}

func TestParseBatchJSON_InvalidJSON_ReturnsNil(t *testing.T) {
	t.Parallel()
	reqs := []domain.BodyGenRequest{{MemberName: "X"}}
	if got := parseBatchJSON("not json at all", reqs); got != nil {
		t.Errorf("expected nil for invalid JSON, got %v", got)
	}
}

func TestParseBatchJSON_EmptyTestsArray_ReturnsNil(t *testing.T) {
	t.Parallel()
	reqs := []domain.BodyGenRequest{{MemberName: "X"}}
	if got := parseBatchJSON(`{"tests":[]}`, reqs); got != nil {
		t.Errorf("expected nil for empty tests array, got %v", got)
	}
}

func TestParseBatchJSON_StripsMdWrapper(t *testing.T) {
	t.Parallel()
	// Model wraps JSON in a fenced code block.
	wrapped := "```json\n" + `{"tests":[{"name":"X","code":"func TestX() {}"}]}` + "\n```"
	reqs := []domain.BodyGenRequest{{MemberName: "X"}}
	results := parseBatchJSON(wrapped, reqs)
	if results == nil {
		t.Fatal("parseBatchJSON should handle markdown-wrapped JSON")
	}
}

// ── GenerateBatchBodies ───────────────────────────────────────────────────────

type recordingProv struct {
	response string
}

func (r *recordingProv) Complete(_ context.Context, req CompletionRequest) (CompletionResponse, error) {
	return CompletionResponse{Content: r.response, TokensUsed: 10}, nil
}

func TestGenerateBatchBodies_EmptyReturnsNil(t *testing.T) {
	t.Parallel()
	gen := New(&recordingProv{}, nil, "m", 100, 0)
	results, err := gen.GenerateBatchBodies(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if results != nil {
		t.Errorf("expected nil for empty input, got %v", results)
	}
}

func TestGenerateBatchBodies_SingleDelegatesToGenerateBodies(t *testing.T) {
	t.Parallel()
	raw := "```go\nfunc TestX(t *testing.T) {}\n```"
	gen := New(&recordingProv{response: raw}, nil, "m", 100, 0)
	reqs := []domain.BodyGenRequest{{Language: "go", MemberName: "X", MemberKind: domain.KindFunction}}
	results, err := gen.GenerateBatchBodies(context.Background(), reqs)
	if err != nil {
		t.Fatalf("single-req batch: %v", err)
	}
	if len(results) != 1 || len(results[0].CodeLines) == 0 {
		t.Error("single-req batch should return one non-empty result")
	}
}

func TestGenerateBatchBodies_MultipleViaJSON(t *testing.T) {
	t.Parallel()
	jsonResp := `{"tests":[{"name":"Foo","code":"func TestFoo() {}"},{"name":"Bar","code":"func TestBar() {}"}]}`
	gen := New(&recordingProv{response: jsonResp}, nil, "m", 100, 0)
	reqs := []domain.BodyGenRequest{
		{Language: "go", MemberName: "Foo", MemberKind: domain.KindFunction, SourceCode: "func Foo() {}"},
		{Language: "go", MemberName: "Bar", MemberKind: domain.KindFunction, SourceCode: "func Bar() {}"},
	}
	results, err := gen.GenerateBatchBodies(context.Background(), reqs)
	if err != nil {
		t.Fatalf("batch via JSON: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if len(results[0].CodeLines) == 0 || len(results[1].CodeLines) == 0 {
		t.Error("both members should have non-empty code lines")
	}
}

func TestGenerateBatchBodies_CachesResultsForSingleCalls(t *testing.T) {
	t.Parallel()
	// GenerateBatchBodies caches each member's result individually so that
	// subsequent single-member GenerateBodies calls are served from cache.
	jsonResp := `{"tests":[{"name":"Foo","code":"func TestFoo() {}"},{"name":"Bar","code":"func TestBar() {}"}]}`
	gen := New(&recordingProv{response: jsonResp}, nil, "m", 100, 0)
	reqs := []domain.BodyGenRequest{
		{Language: "go", MemberName: "Foo", MemberKind: domain.KindFunction, SourceCode: "func Foo() {}"},
		{Language: "go", MemberName: "Bar", MemberKind: domain.KindFunction, SourceCode: "func Bar() {}"},
	}
	_, _ = gen.GenerateBatchBodies(context.Background(), reqs)

	// Individual single-member calls should now be served from cache.
	h0, _, _ := gen.CacheStats()
	_, _ = gen.GenerateBodies(context.Background(), reqs[0])
	_, _ = gen.GenerateBodies(context.Background(), reqs[1])
	h1, _, _ := gen.CacheStats()
	if h1-h0 < 2 {
		t.Errorf("expected 2 cache hits after batch pre-population, got %d", h1-h0)
	}
}

// ── CacheStats ────────────────────────────────────────────────────────────────

func TestCacheStats_DelegatesToCache(t *testing.T) {
	t.Parallel()
	raw := "```go\nfunc TestFoo(t *testing.T) {}\n```"
	gen := New(&recordingProv{response: raw}, nil, "m", 100, 0)

	req := domain.BodyGenRequest{Language: "go", MemberName: "Foo", MemberKind: domain.KindFunction}
	_, _ = gen.GenerateBodies(context.Background(), req)
	_, _ = gen.GenerateBodies(context.Background(), req) // second call → cache hit

	h, m, sz := gen.CacheStats()
	if h < 1 {
		t.Errorf("CacheStats hits = %d, want >= 1", h)
	}
	if m < 1 {
		t.Errorf("CacheStats misses = %d, want >= 1", m)
	}
	if sz < 1 {
		t.Errorf("CacheStats size = %d, want >= 1", sz)
	}
}
