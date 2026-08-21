// Package anthropic provides an Anthropic Messages API provider using plain net/http.
package anthropic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/orieken/assay/internal/llm"
)

const defaultBaseURL = "https://api.anthropic.com/v1"
const anthropicVersion = "2023-06-01"

// sharedTransport is reused across all Provider instances to enable connection pooling.
var sharedTransport = &http.Transport{
	MaxIdleConns:        10,
	MaxIdleConnsPerHost: 10,
	IdleConnTimeout:     30 * time.Second,
}

// Provider calls the Anthropic Messages API.
type Provider struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

// New returns a Provider. Pass "" for baseURL to use the default Anthropic endpoint.
func New(apiKey, baseURL string) *Provider {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	return &Provider{
		apiKey:  apiKey,
		baseURL: baseURL,
		client: &http.Client{
			Timeout:   90 * time.Second,
			Transport: sharedTransport,
		},
	}
}

func (p *Provider) Complete(ctx context.Context, req llm.CompletionRequest) (llm.CompletionResponse, error) {
	// Use the structured content format so we can attach cache_control blocks.
	// The system prompt and user prompt are both marked ephemeral — Anthropic
	// caches these for up to 5 minutes, cutting repeat token costs by ~90%.
	systemBlock := map[string]any{
		"type":          "text",
		"text":          req.SystemPrompt,
		"cache_control": map[string]string{"type": "ephemeral"},
	}
	userBlock := map[string]any{
		"type":          "text",
		"text":          req.UserPrompt,
		"cache_control": map[string]string{"type": "ephemeral"},
	}
	body := map[string]any{
		"model":      req.Model,
		"max_tokens": req.MaxTokens,
		"system":     []map[string]any{systemBlock},
		"messages": []map[string]any{
			{"role": "user", "content": []map[string]any{userBlock}},
		},
	}

	data, err := json.Marshal(body)
	if err != nil {
		return llm.CompletionResponse{}, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/messages", bytes.NewReader(data))
	if err != nil {
		return llm.CompletionResponse{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.apiKey)
	httpReq.Header.Set("anthropic-version", anthropicVersion)
	httpReq.Header.Set("anthropic-beta", "prompt-caching-2024-07-31")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return llm.CompletionResponse{}, err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return llm.CompletionResponse{}, err
	}

	if resp.StatusCode != http.StatusOK {
		return llm.CompletionResponse{}, fmt.Errorf("anthropic API error %d: %s", resp.StatusCode, string(raw))
	}

	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
		Usage struct {
			InputTokens              int `json:"input_tokens"`
			OutputTokens             int `json:"output_tokens"`
			CacheReadInputTokens     int `json:"cache_read_input_tokens"`
			CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return llm.CompletionResponse{}, err
	}

	text := ""
	if len(result.Content) > 0 {
		text = result.Content[0].Text
	}

	// Input tokens: include cache-related tokens so the reported total is accurate.
	inputTokens := result.Usage.InputTokens +
		result.Usage.CacheReadInputTokens + result.Usage.CacheCreationInputTokens

	return llm.CompletionResponse{
		Content:      text,
		InputTokens:  inputTokens,
		OutputTokens: result.Usage.OutputTokens,
		TokensUsed:   inputTokens + result.Usage.OutputTokens,
	}, nil
}
