// Package llm provides the LLMBodyGenerator adapter that implements domain.BodyGenerator.
package llm

import (
	"bytes"
	"context"
	"regexp"
	"text/template"

	"github.com/orieken/testsmith/internal/domain"
)

// Provider is the low-level interface each LLM backend implements.
type Provider interface {
	Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error)
}

// CompletionRequest is the provider-agnostic request payload.
type CompletionRequest struct {
	SystemPrompt string
	UserPrompt   string
	Model        string
	MaxTokens    int
	Temperature  float64
}

// CompletionResponse is the provider-agnostic response.
type CompletionResponse struct {
	Content    string
	TokensUsed int
}

// LLMBodyGenerator implements domain.BodyGenerator.
type LLMBodyGenerator struct {
	provider Provider
	prompts  map[string]string // language -> prompt template string
	model    string
	maxTokens int
	temperature float64
}

// New returns an LLMBodyGenerator.
func New(provider Provider, prompts map[string]string, model string, maxTokens int, temperature float64) *LLMBodyGenerator {
	return &LLMBodyGenerator{
		provider:    provider,
		prompts:     prompts,
		model:       model,
		maxTokens:   maxTokens,
		temperature: temperature,
	}
}

// GenerateBodies calls the LLM for one member and returns the parsed code lines.
func (g *LLMBodyGenerator) GenerateBodies(ctx context.Context, req domain.BodyGenRequest) ([]domain.BodyGenResult, error) {
	promptTmpl, ok := g.prompts[req.Language]
	if !ok {
		promptTmpl = defaultPrompt
	}

	var fixtureNames []string
	for _, f := range req.Fixtures {
		fixtureNames = append(fixtureNames, f.FuncName)
	}

	data := domain.BodyPromptData{
		MemberName:   req.MemberName,
		MemberKind:   string(req.MemberKind),
		SourceCode:   req.SourceCode,
		FixtureNames: fixtureNames,
		Extra:        req.Extra,
	}

	userPrompt, err := renderPrompt(promptTmpl, data)
	if err != nil {
		return nil, err
	}

	resp, err := g.provider.Complete(ctx, CompletionRequest{
		SystemPrompt: "You are a strict code generation assistant. Output only valid code in a markdown code block.",
		UserPrompt:   userPrompt,
		Model:        g.model,
		MaxTokens:    g.maxTokens,
		Temperature:  g.temperature,
	})
	if err != nil {
		return nil, err
	}

	lines := parseCodeBlock(resp.Content)
	return []domain.BodyGenResult{{
		MemberName: req.MemberName,
		CodeLines:  lines,
		TokensUsed: resp.TokensUsed,
	}}, nil
}

var codeBlockRe = regexp.MustCompile("(?s)```(?:python|go|typescript|java|ruby|javascript)?\n(.*?)\n```")

func parseCodeBlock(response string) []string {
	matches := codeBlockRe.FindStringSubmatch(response)
	if len(matches) < 2 {
		return nil
	}
	var lines []string
	for _, line := range bytes.Split([]byte(matches[1]), []byte("\n")) {
		lines = append(lines, string(line))
	}
	return lines
}

func renderPrompt(tmplStr string, data domain.BodyPromptData) (string, error) {
	t, err := template.New("prompt").Parse(tmplStr)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

const defaultPrompt = `Generate test code for the {{.MemberKind}} named ` + "`{{.MemberName}}`" + `.

Source:
` + "```\n{{.SourceCode}}\n```" + `

Output only valid code in a markdown code block.`
