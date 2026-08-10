package openai_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/orieken/assay/internal/llm"
	"github.com/orieken/assay/internal/llm/openai"
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
			"usage": map[string]int{
				"prompt_tokens":     14,
				"completion_tokens": 6,
				"total_tokens":      20,
			},
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
	if resp.InputTokens != 14 {
		t.Errorf("InputTokens: got %d, want 14", resp.InputTokens)
	}
	if resp.OutputTokens != 6 {
		t.Errorf("OutputTokens: got %d, want 6", resp.OutputTokens)
	}
	if resp.TokensUsed != 20 {
		t.Errorf("TokensUsed: got %d, want 20", resp.TokensUsed)
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

func TestNew_DefaultBaseURL(t *testing.T) {
	// Passing "" should use the default endpoint without panicking.
	p := openai.New("key", "")
	if p == nil {
		t.Fatal("New returned nil")
	}
}

func TestComplete_WithResponseFormat(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if _, ok := body["response_format"]; !ok {
			t.Errorf("expected response_format in request body")
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{"message": map[string]string{"content": "{}"}},
			},
			"usage": map[string]int{"total_tokens": 5},
		})
	}))
	defer srv.Close()

	p := openai.New("key", srv.URL)
	resp, err := p.Complete(context.Background(), llm.CompletionRequest{
		Model: "gpt-4o", MaxTokens: 10, ResponseFormat: "json_object",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content == "" {
		t.Error("expected non-empty content")
	}
}

func TestComplete_EmptyChoices_ReturnsEmptyContent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []any{},
			"usage":   map[string]int{"total_tokens": 0},
		})
	}))
	defer srv.Close()

	p := openai.New("key", srv.URL)
	resp, err := p.Complete(context.Background(), llm.CompletionRequest{Model: "m", MaxTokens: 10})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != "" {
		t.Errorf("expected empty content for empty choices, got %q", resp.Content)
	}
}

func TestComplete_NetworkError(t *testing.T) {
	// Point at a port nothing is listening on.
	p := openai.New("key", "http://127.0.0.1:1")
	_, err := p.Complete(context.Background(), llm.CompletionRequest{Model: "m", MaxTokens: 10})
	if err == nil {
		t.Error("expected error for unreachable host")
	}
}
