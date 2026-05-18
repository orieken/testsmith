// Package factory builds a domain.BodyGenerator from application config.
// It lives in its own package to avoid an import cycle between the llm parent
// package and its provider subpackages (anthropic, openai, ollama).
package factory

import (
	"fmt"
	"os"
	"time"

	"github.com/orieken/testsmith/internal/config"
	"github.com/orieken/testsmith/internal/domain"
	"github.com/orieken/testsmith/internal/llm"
	"github.com/orieken/testsmith/internal/llm/anthropic"
	"github.com/orieken/testsmith/internal/llm/ollama"
	"github.com/orieken/testsmith/internal/llm/openai"
)

// Build constructs a BodyGenerator for the given driver and config.
// Returns nil, nil when LLM is disabled — callers pass nil to generation.NewPipeline.
func Build(cfg config.LLMConfig, driver domain.LanguageDriver) (domain.BodyGenerator, error) {
	if !cfg.Enabled {
		return nil, nil
	}

	apiKey := os.Getenv(cfg.APIKeyEnvVar)
	if apiKey == "" && cfg.Provider != "ollama" {
		return nil, fmt.Errorf("LLM provider %q requires env var %s to be set", cfg.Provider, cfg.APIKeyEnvVar)
	}

	rawProvider, err := buildProvider(cfg, apiKey)
	if err != nil {
		return nil, err
	}

	// Layer middleware: semaphore caps concurrency, retry handles transient errors.
	// Order: retry wraps semaphore wraps raw provider — so each retry attempt
	// waits for a semaphore slot independently, preventing slot starvation.
	limited  := llm.WithSemaphore(rawProvider, cfg.MaxConcurrentCalls)
	provider := llm.WithRetry(limited, llm.RetryStrategy{
		MaxAttempts: cfg.MaxRetryAttempts,
		BaseDelay:   500 * time.Millisecond,
		MaxDelay:    30 * time.Second,
		Multiplier:  2.0,
	})

	prompts := map[string]string{
		driver.Language(): driver.BodyGenerationPrompt(),
	}

	return llm.New(provider, prompts, cfg.Model, cfg.MaxTokensPerFunction, cfg.Temperature), nil
}

func buildProvider(cfg config.LLMConfig, apiKey string) (llm.Provider, error) {
	switch cfg.Provider {
	case "anthropic", "":
		return anthropic.New(apiKey, cfg.BaseURL), nil
	case "openai":
		return openai.New(apiKey, cfg.BaseURL), nil
	case "ollama":
		return ollama.New(cfg.BaseURL), nil
	default:
		return nil, fmt.Errorf("unknown LLM provider %q (supported: anthropic, openai, ollama)", cfg.Provider)
	}
}
