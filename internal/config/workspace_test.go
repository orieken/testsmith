package config_test

import (
	"os"
	"testing"

	"github.com/orieken/testsmith/internal/config"
)

func TestWorkspaceID_UsesNameWhenSet(t *testing.T) {
	ws := &config.WorkspaceConfig{Name: "api", Path: "services/api"}
	if got := config.WorkspaceID(ws); got != "api" {
		t.Errorf("WorkspaceID = %q, want %q", got, "api")
	}
}

func TestWorkspaceID_FallsBackToPath(t *testing.T) {
	ws := &config.WorkspaceConfig{Path: "services/api"}
	if got := config.WorkspaceID(ws); got != "services/api" {
		t.Errorf("WorkspaceID = %q, want %q", got, "services/api")
	}
}

func TestWorkspaceLLM_UsesWorkspaceOverride(t *testing.T) {
	root := config.LLMConfig{Provider: "anthropic", Model: "claude-opus-4-7"}
	wsCfg := &config.LLMConfig{Provider: "openai", Model: "gpt-4o"}
	ws := &config.WorkspaceConfig{LLM: wsCfg}

	got := config.WorkspaceLLM(root, ws)
	if got.Provider != "openai" {
		t.Errorf("provider: got %q, want openai", got.Provider)
	}
}

func TestWorkspaceLLM_InheritsRootWhenNil(t *testing.T) {
	root := config.LLMConfig{Provider: "anthropic", Model: "claude-opus-4-7"}
	ws := &config.WorkspaceConfig{}

	got := config.WorkspaceLLM(root, ws)
	if got.Provider != "anthropic" {
		t.Errorf("provider: got %q, want anthropic", got.Provider)
	}
}

func TestWorkspaceConfig_YAMLRoundTrip(t *testing.T) {
	content := `
workspaces:
  - name: api
    path: services/api
    language: go
  - path: frontend
    language: typescript
`
	tmpFile := t.TempDir() + "/ws.yaml"
	if err := os.WriteFile(tmpFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.LoadFromFile(tmpFile)
	if err != nil {
		t.Fatalf("LoadFromFile: %v", err)
	}
	if len(cfg.Workspaces) != 2 {
		t.Fatalf("expected 2 workspaces, got %d", len(cfg.Workspaces))
	}
	if cfg.Workspaces[0].Name != "api" {
		t.Errorf("workspace[0].Name = %q, want api", cfg.Workspaces[0].Name)
	}
	if cfg.Workspaces[0].Language != "go" {
		t.Errorf("workspace[0].Language = %q, want go", cfg.Workspaces[0].Language)
	}
	if cfg.Workspaces[1].Path != "frontend" {
		t.Errorf("workspace[1].Path = %q, want frontend", cfg.Workspaces[1].Path)
	}
}
