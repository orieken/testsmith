package anthropic_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/orieken/testsmith/internal/llm"
	"github.com/orieken/testsmith/internal/llm/anthropic"
)

func TestComplete_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("x-api-key") != "test-key" {
			t.Errorf("missing x-api-key header")
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"content": []map[string]string{{"text": "```python\ndef test(): pass\n```"}},
			"usage":   map[string]int{"input_tokens": 10, "output_tokens": 5},
		})
	}))
	defer srv.Close()

	p := anthropic.New("test-key", srv.URL)
	resp, err := p.Complete(context.Background(), llm.CompletionRequest{
		SystemPrompt: "system",
		UserPrompt:   "user",
		Model:        "claude-sonnet-4-6",
		MaxTokens:    100,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.TokensUsed != 15 {
		t.Errorf("tokens: got %d, want 15", resp.TokensUsed)
	}
	if resp.Content == "" {
		t.Error("expected non-empty content")
	}
}

func TestComplete_Non200_ReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"invalid_api_key"}`, http.StatusUnauthorized)
	}))
	defer srv.Close()

	p := anthropic.New("bad-key", srv.URL)
	_, err := p.Complete(context.Background(), llm.CompletionRequest{Model: "m", MaxTokens: 10})
	if err == nil {
		t.Error("expected error for 401 response")
	}
}
