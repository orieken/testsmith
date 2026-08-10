package llm_test

import (
	"context"
	"testing"

	"github.com/orieken/assay/internal/domain"
	"github.com/orieken/assay/internal/llm"
)

// stubProvider returns a fixed completion string.
type stubProvider struct {
	content    string
	tokensUsed int
	err        error
}

func (s *stubProvider) Complete(_ context.Context, _ llm.CompletionRequest) (llm.CompletionResponse, error) {
	return llm.CompletionResponse{Content: s.content, TokensUsed: s.tokensUsed}, s.err
}

func TestGenerateBodies_ParsesCodeBlock(t *testing.T) {
	raw := "Here is the test:\n```python\ndef test_foo():\n    assert 1 == 1\n```"
	gen := llm.New(&stubProvider{content: raw, tokensUsed: 42},
		map[string]string{"python": "generate tests for {{.MemberName}}"},
		"claude-sonnet-4-6", 1500, 0.0)

	req := domain.BodyGenRequest{
		Language:   "python",
		MemberName: "foo",
		MemberKind: domain.KindFunction,
		SourceCode: "def foo(): pass",
	}
	results, err := gen.GenerateBodies(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	r := results[0]
	if r.MemberName != "foo" {
		t.Errorf("member name: got %q, want %q", r.MemberName, "foo")
	}
	if r.TokensUsed != 42 {
		t.Errorf("tokens: got %d, want 42", r.TokensUsed)
	}
	if len(r.CodeLines) == 0 {
		t.Error("code lines should not be empty")
	}
	if r.CodeLines[0] != "def test_foo():" {
		t.Errorf("first code line: got %q", r.CodeLines[0])
	}
}

func TestGenerateBodies_FallsBackToDefaultPrompt(t *testing.T) {
	raw := "```go\nfunc TestFoo(t *testing.T) {}\n```"
	gen := llm.New(&stubProvider{content: raw},
		nil, // no prompts map → uses defaultPrompt
		"claude-sonnet-4-6", 1500, 0.0)

	req := domain.BodyGenRequest{
		Language:   "go",
		MemberName: "Foo",
		MemberKind: domain.KindFunction,
		SourceCode: "func Foo() {}",
	}
	results, err := gen.GenerateBodies(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) == 0 || len(results[0].CodeLines) == 0 {
		t.Error("expected non-empty code lines from default prompt path")
	}
}

func TestGenerateBodies_NoCodeBlock_ReturnsNilLines(t *testing.T) {
	gen := llm.New(&stubProvider{content: "sorry, I cannot help with that"},
		nil, "model", 100, 0.0)

	req := domain.BodyGenRequest{Language: "python", MemberName: "bar", MemberKind: domain.KindFunction}
	results, err := gen.GenerateBodies(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results[0].CodeLines) != 0 {
		t.Errorf("expected nil code lines when no code block, got %v", results[0].CodeLines)
	}
}

func TestGenerateBodies_InvalidPromptTemplate_ReturnsError(t *testing.T) {
	gen := llm.New(&stubProvider{content: "anything"},
		map[string]string{"go": "{{.Unclosed"},
		"model", 100, 0.0)

	req := domain.BodyGenRequest{Language: "go", MemberName: "Foo", MemberKind: domain.KindFunction}
	_, err := gen.GenerateBodies(context.Background(), req)
	if err == nil {
		t.Error("expected error for invalid template, got nil")
	}
}

func TestGenerateBodies_ProviderError_Propagates(t *testing.T) {
	gen := llm.New(&stubProvider{err: context.DeadlineExceeded},
		nil, "model", 100, 0.0)

	req := domain.BodyGenRequest{Language: "python", MemberName: "baz", MemberKind: domain.KindFunction}
	_, err := gen.GenerateBodies(context.Background(), req)
	if err == nil {
		t.Error("expected error from provider, got nil")
	}
}
