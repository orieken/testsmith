package ollama_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/orieken/assay/internal/llm"
	"github.com/orieken/assay/internal/llm/ollama"
)

func openAIResponse(content string, tokens int) map[string]any {
	return map[string]any{
		"choices": []map[string]any{
			{"message": map[string]string{"content": content}},
		},
		"usage": map[string]int{"total_tokens": tokens},
	}
}

func TestComplete_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/chat/completions") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(openAIResponse("```go\nfunc TestFoo(t *testing.T) {}\n```", 42))
	}))
	defer srv.Close()

	p := ollama.New(srv.URL + "/v1")
	resp, err := p.Complete(context.Background(), llm.CompletionRequest{
		Model:        "llama3",
		SystemPrompt: "You are a test assistant.",
		UserPrompt:   "Write tests for Foo.",
		MaxTokens:    200,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.TokensUsed != 42 {
		t.Errorf("tokens: got %d, want 42", resp.TokensUsed)
	}
	if resp.Content == "" {
		t.Error("expected non-empty content")
	}
}

func TestComplete_UsesProvidedBaseURL(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(openAIResponse("ok", 1))
	}))
	defer srv.Close()

	p := ollama.New(srv.URL + "/v1")
	_, _ = p.Complete(context.Background(), llm.CompletionRequest{Model: "llama3", MaxTokens: 10})

	if gotPath != "/v1/chat/completions" {
		t.Errorf("path: got %q, want /v1/chat/completions", gotPath)
	}
}

func TestComplete_DefaultBaseURL_NonEmpty(t *testing.T) {
	// Passing empty string should fall back to the default Ollama URL without panic.
	// We can't actually connect, so just verify New("") doesn't panic.
	p := ollama.New("")
	if p == nil {
		t.Error("expected non-nil provider with default base URL")
	}
}

func TestComplete_Non200_ReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"model not found"}`, http.StatusNotFound)
	}))
	defer srv.Close()

	p := ollama.New(srv.URL + "/v1")
	_, err := p.Complete(context.Background(), llm.CompletionRequest{Model: "missing", MaxTokens: 10})
	if err == nil {
		t.Error("expected error for 404 response")
	}
}

func TestComplete_SendsAuthHeader(t *testing.T) {
	// Ollama uses "ollama" as the API key which becomes "Bearer ollama".
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(openAIResponse("ok", 1))
	}))
	defer srv.Close()

	p := ollama.New(srv.URL + "/v1")
	_, _ = p.Complete(context.Background(), llm.CompletionRequest{Model: "llama3", MaxTokens: 10})

	if gotAuth != "Bearer ollama" {
		t.Errorf("Authorization: got %q, want %q", gotAuth, "Bearer ollama")
	}
}
