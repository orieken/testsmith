package openai_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/orieken/testsmith/internal/llm"
	"github.com/orieken/testsmith/internal/llm/openai"
)

func TestComplete_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("expected Bearer token, got %q", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{"message": map[string]string{"content": "```go\nfunc TestFoo(t *testing.T) {}\n```"}},
			},
			"usage": map[string]int{"total_tokens": 20},
		})
	}))
	defer srv.Close()

	p := openai.New("test-key", srv.URL)
	resp, err := p.Complete(context.Background(), llm.CompletionRequest{
		SystemPrompt: "system",
		UserPrompt:   "user",
		Model:        "gpt-4o",
		MaxTokens:    100,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.TokensUsed != 20 {
		t.Errorf("tokens: got %d, want 20", resp.TokensUsed)
	}
	if resp.Content == "" {
		t.Error("expected non-empty content")
	}
}

func TestComplete_Non200_ReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":{"message":"rate limit"}}`, http.StatusTooManyRequests)
	}))
	defer srv.Close()

	p := openai.New("key", srv.URL)
	_, err := p.Complete(context.Background(), llm.CompletionRequest{Model: "m", MaxTokens: 10})
	if err == nil {
		t.Error("expected error for 429 response")
	}
}
