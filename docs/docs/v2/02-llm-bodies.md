# Feature: LLM Test Body Generation

## What It Does

By default, `testsmith generate` produces test scaffolds with `// TODO: implement` stubs. Passing `--llm` activates the optional `BodyGenerator` adapter, which sends each public member's context to a configured LLM provider (Anthropic, OpenAI, or a local Ollama instance) and receives ready-to-run test code in return.

The LLM integration is:
- **Opt-in**: the core pipeline runs without it; no API key required for basic usage.
- **Provider-swappable**: select provider via `.testsmith.yaml`; all three use plain `net/http` calls — no heavyweight SDK.
- **Language-aware**: each driver ships its own prompt template, so pytest prompts differ from Go `testing` prompts.
- **Non-blocking**: if an LLM call fails (network, quota, timeout) the member falls back to a TODO stub and a warning is printed; generation continues.

---

## CLI

```
testsmith generate src/payment.py --llm
testsmith generate --all --llm --lang python
```

No additional flags required beyond `--llm`. LLM configuration lives in `.testsmith.yaml`.

---

## Configuration

```yaml
# .testsmith.yaml
llm:
  enabled: false                  # set true, or pass --llm at runtime
  provider: "anthropic"           # anthropic | openai | ollama
  model: "claude-sonnet-4-6"
  max_tokens_per_function: 1500
  temperature: 0.0
  api_key_env_var: "ANTHROPIC_API_KEY"
  base_url: ""                    # for OpenAI-compatible or Ollama endpoints
```

Environment variable takes precedence over the config value for the API key.

---

## Generated Output (Python example)

Without `--llm`:
```python
def test_charge(self, mock_stripe):
    # TODO: implement
    pass
```

With `--llm`:
```python
def test_charge_success(self, mock_stripe):
    # arrange
    mock_stripe.Charge.create.return_value = {"id": "ch_test", "status": "succeeded"}
    processor = PaymentProcessor(mock_stripe)
    # act
    result = processor.charge(amount=1000, currency="usd", token="tok_test")
    # assert
    assert result["status"] == "succeeded"
    mock_stripe.Charge.create.assert_called_once()

def test_charge_declined(self, mock_stripe):
    mock_stripe.Charge.create.side_effect = stripe.error.CardError("declined", None, "card_declined")
    processor = PaymentProcessor(mock_stripe)
    with pytest.raises(stripe.error.CardError):
        processor.charge(amount=1000, currency="usd", token="tok_bad")
```

---

## Go Implementation

### Interface (defined in domain)

```go
// internal/domain/driver.go
type BodyGenerator interface {
    GenerateBodies(ctx context.Context, req BodyGenRequest) ([]BodyGenResult, error)
}
```

### Provider Adapter

```go
// internal/llm/generator.go
type LLMBodyGenerator struct {
    provider  Provider         // net/http client (Anthropic / OpenAI / Ollama)
    prompts   PromptRegistry   // map[language]templateString
}

func (g *LLMBodyGenerator) GenerateBodies(ctx context.Context, req domain.BodyGenRequest) ([]domain.BodyGenResult, error)
```

### Prompt Registry

Each driver embeds its own prompt template:

```go
// internal/drivers/python/prompts.go
//go:embed prompts/generate_body.tmpl
var generateBodyPrompt string
```

The `PromptRegistry` is populated at startup by calling `driver.BodyGenerationPrompt()` for every registered driver, keyed by `driver.Language()`.

### Files Involved

| File | Role |
|------|------|
| `internal/domain/driver.go` | `BodyGenerator` interface |
| `internal/llm/generator.go` | Implements `BodyGenerator`; renders prompts; fans out calls |
| `internal/llm/anthropic/provider.go` | Anthropic Messages API (`net/http`) |
| `internal/llm/openai/provider.go` | OpenAI-compatible Chat Completions API |
| `internal/llm/ollama/provider.go` | Ollama local REST API |
| `internal/drivers/<lang>/prompts/generate_body.tmpl` | Language-specific prompt template |
| `internal/config/schema.go` | `LLMConfig` struct |

### Security Note

The API key is **always** read from the environment variable named in `llm.api_key_env_var`. It is never written to `.testsmith.yaml` or committed to the repository.
