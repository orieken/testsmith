// Package factory builds a domain.BodyGenerator from application config.
// It lives in its own package to avoid an import cycle between the llm parent
// package and its provider subpackages (anthropic, openai, ollama).
package factory

import (
	"fmt"
	"os"
	"time"

	"github.com/orieken/assay/internal/config"
	"github.com/orieken/assay/internal/domain"
	"github.com/orieken/assay/internal/llm"
	"github.com/orieken/assay/internal/llm/anthropic"
	"github.com/orieken/assay/internal/llm/ollama"
	"github.com/orieken/assay/internal/llm/openai"
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
	limited := llm.WithSemaphore(rawProvider, cfg.MaxConcurrentCalls)
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

// BuildProvider constructs a middleware-wrapped Provider without the BodyGenerator
// abstraction. Use this when you need direct LLM access with a custom prompt
// (e.g. the learn subcommand's pattern-extraction call).
// Returns an error when LLM is disabled or the API key is missing.
func BuildProvider(cfg config.LLMConfig) (llm.Provider, error) {
	if !cfg.Enabled {
		return nil, fmt.Errorf("LLM is disabled — set llm.enabled: true in .assay.yaml")
	}
	apiKey := os.Getenv(cfg.APIKeyEnvVar)
	if apiKey == "" && cfg.Provider != "ollama" {
		return nil, fmt.Errorf("LLM provider %q requires env var %s to be set", cfg.Provider, cfg.APIKeyEnvVar)
	}
	raw, err := buildProvider(cfg, apiKey)
	if err != nil {
		return nil, err
	}
	limited := llm.WithSemaphore(raw, 1)
	return llm.WithRetry(limited, llm.RetryStrategy{
		MaxAttempts: cfg.MaxRetryAttempts,
		BaseDelay:   500 * time.Millisecond,
		MaxDelay:    30 * time.Second,
		Multiplier:  2.0,
	}), nil
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
