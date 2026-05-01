// Package openai provides an OpenAI-compatible Chat Completions provider using plain net/http.
// Works with OpenAI, Azure OpenAI, and Ollama (which exposes an OpenAI-compatible endpoint).
package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/orieken/testsmith/internal/llm"
)

const defaultBaseURL = "https://api.openai.com/v1"

// Provider calls the OpenAI Chat Completions API (or any compatible endpoint).
type Provider struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

func New(apiKey, baseURL string) *Provider {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	return &Provider{apiKey: apiKey, baseURL: baseURL, client: &http.Client{}}
}

func (p *Provider) Complete(ctx context.Context, req llm.CompletionRequest) (llm.CompletionResponse, error) {
	body := map[string]any{
		"model":       req.Model,
		"max_tokens":  req.MaxTokens,
		"temperature": req.Temperature,
		"messages": []map[string]string{
			{"role": "system", "content": req.SystemPrompt},
			{"role": "user", "content": req.UserPrompt},
		},
	}

	data, err := json.Marshal(body)
	if err != nil {
		return llm.CompletionResponse{}, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/chat/completions", bytes.NewReader(data))
	if err != nil {
		return llm.CompletionResponse{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

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
		return llm.CompletionResponse{}, fmt.Errorf("openai API error %d: %s", resp.StatusCode, string(raw))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			TotalTokens int `json:"total_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return llm.CompletionResponse{}, err
	}

	text := ""
	if len(result.Choices) > 0 {
		text = result.Choices[0].Message.Content
	}

	return llm.CompletionResponse{
		Content:    text,
		TokensUsed: result.Usage.TotalTokens,
	}, nil
}
