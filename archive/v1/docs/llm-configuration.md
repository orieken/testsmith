# LLM Configuration Guide

TestSmith supports generating test bodies using Large Language Models (LLMs). By default, it uses Anthropic's Claude models, but it also supports OpenAI-compatible APIs (like OpenAI, Open-UI, LocalAI, etc.).

## Configuration

Configure the LLM settings in your `pyproject.toml` file under the `[tool.testsmith.llm]` section.

### 1. Anthropic (Default)

To use Anthropic's Claude models:

```toml
[tool.testsmith.llm]
enabled = true
provider = "anthropic"  # Default
model = "claude-3-sonnet-20240229"
api_key_env_var = "ANTHROPIC_API_KEY"
max_tokens_per_function = 1500
temperature = 0.0
```

Set your API key:
```bash
export ANTHROPIC_API_KEY="your-api-key"
```

### 2. OpenAI

To use OpenAI's GPT models:

```toml
[tool.testsmith.llm]
enabled = true
provider = "openai"
model = "gpt-4-turbo-preview"
api_key_env_var = "OPENAI_API_KEY"
max_tokens_per_function = 1500
temperature = 0.0
```

Set your API key:
```bash
export OPENAI_API_KEY="your-openai-api-key"
```

### 3. Custom OpenAI-Compatible API (e.g., Open-UI, LocalAI)

To use a custom endpoint that follows the OpenAI API format (like Open-UI):

```toml
[tool.testsmith.llm]
enabled = true
provider = "custom"
model = "llama-3-8b-instruct"  # Whatever model name your server expects
base_url = "http://localhost:3000/v1"  # Your custom endpoint
api_key_env_var = "OPENUI_API_KEY"     # Or whatever env var holds your key
max_tokens_per_function = 1500
temperature = 0.0
```

Set your API key (even if it's a dummy key if your local server requires one):
```bash
export OPENUI_API_KEY="sk-dummy-key"
```

## Usage

Once configured, run TestSmith with the `--generate-bodies` flag (if implemented) or simply run the tool normally if it uses LLM by default for generation.

```bash
testsmith src/myapp/module.py
```
