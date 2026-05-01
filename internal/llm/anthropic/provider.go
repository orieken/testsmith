// Package anthropic provides an Anthropic Messages API provider using plain net/http.
package anthropic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/orieken/testsmith/internal/llm"
)

const defaultBaseURL = "https://api.anthropic.com/v1"
const anthropicVersion = "2023-06-01"

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
	return &Provider{apiKey: apiKey, baseURL: baseURL, client: &http.Client{}}
}

func (p *Provider) Complete(ctx context.Context, req llm.CompletionRequest) (llm.CompletionResponse, error) {
	body := map[string]any{
		"model":      req.Model,
		"max_tokens": req.MaxTokens,
		"system":     req.SystemPrompt,
		"messages": []map[string]string{
			{"role": "user", "content": req.UserPrompt},
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
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return llm.CompletionResponse{}, err
	}

	text := ""
	if len(result.Content) > 0 {
		text = result.Content[0].Text
	}

	return llm.CompletionResponse{
		Content:    text,
		TokensUsed: result.Usage.InputTokens + result.Usage.OutputTokens,
	}, nil
}
