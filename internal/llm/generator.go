// Package llm provides the LLMBodyGenerator adapter that implements domain.BodyGenerator.
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"text/template"

	"github.com/orieken/testsmith/internal/domain"
)

// Provider is the low-level interface each LLM backend implements.
type Provider interface {
	Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error)
}

// CompletionRequest is the provider-agnostic request payload.
type CompletionRequest struct {
	SystemPrompt   string
	UserPrompt     string
	Model          string
	MaxTokens      int
	Temperature    float64
	// ResponseFormat instructs the provider to return structured output.
	// Currently only "json_object" is defined. Providers that do not support
	// structured output silently ignore this field.
	ResponseFormat string
}

// CompletionResponse is the provider-agnostic response.
type CompletionResponse struct {
	Content    string
	TokensUsed int
}

// LLMBodyGenerator implements domain.BodyGenerator.
type LLMBodyGenerator struct {
	provider    Provider
	prompts     map[string]string // language -> prompt template string
	model       string
	maxTokens   int
	temperature float64
	cache       *ResultCache
}

// CacheStats returns the (hits, misses, size) triple from the in-process
// result cache. Used by the CLI to print a summary under --verbose.
func (g *LLMBodyGenerator) CacheStats() (hits, misses, size int) {
	return g.cache.Stats()
}

// New returns an LLMBodyGenerator.
func New(provider Provider, prompts map[string]string, model string, maxTokens int, temperature float64) *LLMBodyGenerator {
	return &LLMBodyGenerator{
		provider:    provider,
		prompts:     prompts,
		model:       model,
		maxTokens:   maxTokens,
		temperature: temperature,
		cache:       NewResultCache(),
	}
}

// GenerateBodies calls the LLM for one member and returns the parsed code lines.
// Results are cached by request content — repeated calls for unchanged source
// return immediately without an API round-trip (critical for watch mode).
func (g *LLMBodyGenerator) GenerateBodies(ctx context.Context, req domain.BodyGenRequest) ([]domain.BodyGenResult, error) {
	key := CacheKey(req)
	if cached, ok := g.cache.Get(key); ok {
		return cached, nil
	}

	promptTmpl, ok := g.prompts[req.Language]
	if !ok {
		promptTmpl = defaultPrompt
	}

	var fixtureNames []string
	for _, f := range req.Fixtures {
		fixtureNames = append(fixtureNames, f.FuncName)
	}

	data := domain.BodyPromptData{
		MemberName:          req.MemberName,
		MemberKind:          string(req.MemberKind),
		SourceCode:          req.SourceCode,
		FixtureNames:        fixtureNames,
		Extra:               req.Extra,
		ModulePath:          req.ModulePath,
		DepsSignatures:      req.DepsSignatures,
		ExistingTestSnippet: req.ExistingTestSnippet,
		ProjectKnowledge:    req.ProjectKnowledge,
	}

	userPrompt, err := renderPrompt(promptTmpl, data)
	if err != nil {
		return nil, err
	}

	systemPrompt := "You are a strict code generation assistant. Output only valid code in a markdown code block."
	if req.ProjectKnowledge != "" {
		systemPrompt = req.ProjectKnowledge + "\n\n" + systemPrompt
	}

	resp, err := g.provider.Complete(ctx, CompletionRequest{
		SystemPrompt: systemPrompt,
		UserPrompt:   userPrompt,
		Model:        g.model,
		MaxTokens:    g.maxTokens,
		Temperature:  g.temperature,
	})
	if err != nil {
		return nil, err
	}

	lines := parseCodeBlock(resp.Content)
	results := []domain.BodyGenResult{{
		MemberName: req.MemberName,
		CodeLines:  lines,
		TokensUsed: resp.TokensUsed,
	}}
	g.cache.Set(key, results)
	return results, nil
}

// GenerateBatchBodies generates test bodies for all members in a single LLM
// call, reducing API round-trips from N to 1 and giving the model the full
// inter-member context. Results are individually cached after parsing.
//
// If any individual member's block cannot be parsed, that member is silently
// omitted from the results — the caller can fall back to per-member generation.
func (g *LLMBodyGenerator) GenerateBatchBodies(ctx context.Context, reqs []domain.BodyGenRequest) ([]domain.BodyGenResult, error) {
	if len(reqs) == 0 {
		return nil, nil
	}
	if len(reqs) == 1 {
		return g.GenerateBodies(ctx, reqs[0])
	}

	// Use the first request for shared fields (source, framework, project knowledge).
	base := reqs[0]

	// Build a member list for the prompt.
	var memberList strings.Builder
	for i, r := range reqs {
		fmt.Fprintf(&memberList, "%d. %s `%s`\n", i+1, r.MemberKind, r.MemberName)
	}

	lang := base.Extra["language"]
	if lang == "" {
		lang = base.Language
	}
	framework := base.Extra["framework"]

	pd := batchPromptData{
		Language:            lang,
		Framework:           framework,
		ModulePath:          base.ModulePath,
		SourceCode:          base.SourceCode,
		DepsSignatures:      base.DepsSignatures,
		ExistingTestSnippet: base.ExistingTestSnippet,
		MockLibrary:         base.Extra["mock_library"],
		AssertStyle:         base.Extra["assert_style"],
		MemberList:          memberList.String(),
		UseJSONOutput:       true,
	}
	userPrompt := buildBatchPrompt(pd)

	systemPrompt := "You are a strict code generation assistant. Follow the output format exactly."
	if base.ProjectKnowledge != "" {
		systemPrompt = base.ProjectKnowledge + "\n\n" + systemPrompt
	}

	// Scale max tokens: batch output is proportional to member count.
	maxTokens := g.maxTokens * len(reqs)

	resp, err := g.provider.Complete(ctx, CompletionRequest{
		SystemPrompt:   systemPrompt,
		UserPrompt:     userPrompt,
		Model:          g.model,
		MaxTokens:      maxTokens,
		Temperature:    g.temperature,
		ResponseFormat: "json_object",
	})
	if err != nil {
		return nil, err
	}

	// Try JSON first (structured output path); fall back to delimiter regex when
	// the provider doesn't support response_format or the JSON is malformed.
	parsed := parseBatchJSON(resp.Content, reqs)
	if parsed == nil {
		parsed = parseBatchResponse(resp.Content, reqs)
	}

	// Cache each member result individually so watch-mode hits are granular.
	tokensPerMember := resp.TokensUsed / len(reqs)
	for i := range parsed {
		parsed[i].TokensUsed = tokensPerMember
		singleReq := reqs[i]
		g.cache.Set(CacheKey(singleReq), []domain.BodyGenResult{parsed[i]})
	}

	return parsed, nil
}

// batchPromptData carries the fields needed by the batch prompt template.
type batchPromptData struct {
	Language, Framework, ModulePath string
	SourceCode, DepsSignatures      string
	ExistingTestSnippet             string
	MockLibrary, AssertStyle        string
	MemberList                      string
	// UseJSONOutput asks the model to return a JSON object instead of
	// delimiter-separated fenced code blocks. Set to true when the provider
	// supports response_format=json_object (OpenAI-compatible endpoints).
	UseJSONOutput bool
}

func buildBatchPrompt(d batchPromptData) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Generate %s test bodies for the following members of `%s`.\n\n", d.Framework, d.ModulePath)
	sb.WriteString("Members to test:\n")
	sb.WriteString(d.MemberList)
	fmt.Fprintf(&sb, "\nSource:\n```%s\n%s\n```\n", d.Language, d.SourceCode)
	if d.DepsSignatures != "" {
		fmt.Fprintf(&sb, "\nInternal dependencies:\n%s\n", d.DepsSignatures)
	}
	if d.ExistingTestSnippet != "" {
		fmt.Fprintf(&sb, "\nFollow the style of existing tests:\n```\n%s\n```\n", d.ExistingTestSnippet)
	}
	if d.MockLibrary != "" {
		fmt.Fprintf(&sb, "\nMock library: %s\n", d.MockLibrary)
	}
	if d.AssertStyle != "" {
		fmt.Fprintf(&sb, "Assert style: %s\n", d.AssertStyle)
	}

	if d.UseJSONOutput {
		sb.WriteString(`
Return a JSON object with this exact schema:
{"tests":[{"name":"<MemberName>","code":"<test body as a single string>"}]}

Use ONLY the member names listed above. Output all members in order. Escape newlines as \n inside code strings.`)
	} else {
		sb.WriteString(`
For EACH member output EXACTLY:
<!-- TEST: <MemberName> -->
` + "```" + d.Language + `
<test body>
` + "```" + `

Use ONLY the member names listed above. Output all members in order.`)
	}
	return sb.String()
}

// batchBlockRe matches <!-- TEST: Name --> followed by a fenced code block.
var batchBlockRe = regexp.MustCompile("(?s)<!-- TEST: ([^\\s>]+) -->\\s*```[a-zA-Z]*\\n(.*?)\\n```")

// parseBatchResponse extracts per-member code blocks from a batch LLM response.
// Members whose blocks are not found get nil CodeLines (caller treats as stub).
func parseBatchResponse(response string, reqs []domain.BodyGenRequest) []domain.BodyGenResult {
	// Index parsed blocks by member name.
	blocks := make(map[string][]string)
	for _, match := range batchBlockRe.FindAllStringSubmatch(response, -1) {
		name := match[1]
		lines := strings.Split(match[2], "\n")
		blocks[name] = lines
	}

	results := make([]domain.BodyGenResult, len(reqs))
	for i, r := range reqs {
		results[i] = domain.BodyGenResult{
			MemberName: r.MemberName,
			CodeLines:  blocks[r.MemberName],
		}
	}
	return results
}

// parseBatchJSON parses a structured JSON batch response produced when
// response_format=json_object is used. Returns nil if the JSON is absent or
// does not contain a "tests" array — the caller falls back to delimiter parsing.
func parseBatchJSON(response string, reqs []domain.BodyGenRequest) []domain.BodyGenResult {
	// The model may wrap the JSON in a markdown code block; strip it first.
	clean := strings.TrimSpace(response)
	if idx := strings.Index(clean, "{"); idx > 0 {
		clean = clean[idx:]
	}
	if idx := strings.LastIndex(clean, "}"); idx >= 0 && idx < len(clean)-1 {
		clean = clean[:idx+1]
	}

	var envelope struct {
		Tests []struct {
			Name string `json:"name"`
			Code string `json:"code"`
		} `json:"tests"`
	}
	if err := json.Unmarshal([]byte(clean), &envelope); err != nil || len(envelope.Tests) == 0 {
		return nil
	}

	// Build name → code map from the JSON.
	codeByName := make(map[string][]string, len(envelope.Tests))
	for _, t := range envelope.Tests {
		codeByName[t.Name] = strings.Split(t.Code, "\n")
	}

	results := make([]domain.BodyGenResult, len(reqs))
	for i, r := range reqs {
		results[i] = domain.BodyGenResult{
			MemberName: r.MemberName,
			CodeLines:  codeByName[r.MemberName],
		}
	}
	return results
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

const defaultPrompt = `Generate {{index .Extra "framework"}} test code for the {{.MemberKind}} ` + "`{{.MemberName}}`" + ` in module ` + "`{{.ModulePath}}`" + `.

Source:
` + "```{{index .Extra \"language\"}}\n{{.SourceCode}}\n```" + `
{{if .DepsSignatures}}
Internal dependencies (public API):
{{.DepsSignatures}}
{{end}}{{if .ExistingTestSnippet}}
Follow the style of existing tests in this module:
` + "```\n{{.ExistingTestSnippet}}\n```" + `
{{end}}
Available fixtures: {{range .FixtureNames}}{{.}} {{end}}
Mock library: {{index .Extra "mock_library"}}
Assert style: {{index .Extra "assert_style"}}

Output ONLY valid {{index .Extra "language"}} code in a single markdown code block.`
