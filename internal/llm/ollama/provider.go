// Package ollama provides a provider for locally-running Ollama models.
// Ollama exposes an OpenAI-compatible /v1/chat/completions endpoint,
// so this is a thin wrapper around the openai provider with a local base URL.
package ollama

import (
	"context"

	"github.com/orieken/assay/internal/llm"
	"github.com/orieken/assay/internal/llm/openai"
)

const defaultBaseURL = "http://localhost:11434/v1"

// Provider wraps the OpenAI-compatible provider pointed at a local Ollama instance.
type Provider struct {
	inner *openai.Provider
}

func New(baseURL string) *Provider {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	return &Provider{inner: openai.New("ollama", baseURL)}
}

func (p *Provider) Complete(ctx context.Context, req llm.CompletionRequest) (llm.CompletionResponse, error) {
	return p.inner.Complete(ctx, req)
}
