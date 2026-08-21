package main

import (
	"os"
	"path/filepath"
	"testing"
)

// collectRenameOps tests — pure function, safe to parallelise.

func TestCollectRenameOps_ConfigFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	touch(t, filepath.Join(dir, ".assay.yaml"))

	ops := collectRenameOps(dir, "assay", "scaffoldsmith")

	found := false
	for _, op := range ops {
		if op.from == filepath.Join(dir, ".assay.yaml") &&
			op.to == filepath.Join(dir, ".scaffoldsmith.yaml") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected .assay.yaml → .scaffoldsmith.yaml op, got %+v", ops)
	}
}

func TestCollectRenameOps_PatternsDir(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".assay"), 0o755); err != nil {
		t.Fatal(err)
	}

	ops := collectRenameOps(dir, "assay", "scaffoldsmith")

	found := false
	for _, op := range ops {
		if op.from == filepath.Join(dir, ".assay") &&
			op.to == filepath.Join(dir, ".scaffoldsmith") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected .assay/ → .scaffoldsmith/ op, got %+v", ops)
	}
}

func TestCollectRenameOps_KnowledgeFiles(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	subDir := filepath.Join(dir, "src")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}
	touch(t, filepath.Join(dir, "ASSAY.md"))
	touch(t, filepath.Join(subDir, "ASSAY.md"))

	ops := collectRenameOps(dir, "assay", "scaffoldsmith")

	var found int
	for _, op := range ops {
		if filepath.Base(op.from) == "ASSAY.md" && filepath.Base(op.to) == "SCAFFOLDSMITH.md" {
			found++
		}
	}
	if found != 2 {
		t.Errorf("expected 2 ASSAY.md ops, got %d — ops: %+v", found, ops)
	}
}

func TestCollectRenameOps_AgentFiles(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	agentsDir := filepath.Join(dir, ".claude", "agents")
	if err := os.MkdirAll(agentsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	touch(t, filepath.Join(agentsDir, "assay-test-author.md"))
	touch(t, filepath.Join(agentsDir, "assay-pattern-curator.md"))
	touch(t, filepath.Join(agentsDir, "other-agent.md")) // must not be renamed

	ops := collectRenameOps(dir, "assay", "scaffoldsmith")

	var renamed int
	for _, op := range ops {
		if filepath.Dir(op.from) == agentsDir {
			renamed++
			if filepath.Base(op.to) == "other-agent.md" {
				t.Errorf("other-agent.md must not be renamed")
			}
		}
	}
	if renamed != 2 {
		t.Errorf("expected 2 agent rename ops, got %d", renamed)
	}
}

func TestCollectRenameOps_NothingPresent(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	ops := collectRenameOps(dir, "assay", "scaffoldsmith")

	if len(ops) != 0 {
		t.Errorf("expected no ops for empty dir, got %d: %+v", len(ops), ops)
	}
}

// runMigrateConfig integration tests — use testChdir, no t.Parallel().

func TestRunMigrateConfig_RenamesAllArtifacts(t *testing.T) {
	dir := t.TempDir()
	agentsDir := filepath.Join(dir, ".claude", "agents")
	if err := os.MkdirAll(agentsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, ".assay"), 0o755); err != nil {
		t.Fatal(err)
	}
	touch(t, filepath.Join(dir, ".assay.yaml"))
	touch(t, filepath.Join(dir, "ASSAY.md"))
	touch(t, filepath.Join(agentsDir, "assay-test-author.md"))

	testChdir(t, dir)

	if err := runMigrateConfig("assay", "scaffoldsmith"); err != nil {
		t.Fatalf("runMigrateConfig: %v", err)
	}

	assertExists(t, filepath.Join(dir, ".scaffoldsmith.yaml"))
	assertExists(t, filepath.Join(dir, ".scaffoldsmith"))
	assertExists(t, filepath.Join(dir, "SCAFFOLDSMITH.md"))
	assertExists(t, filepath.Join(agentsDir, "scaffoldsmith-test-author.md"))
	assertNotExist(t, filepath.Join(dir, ".assay.yaml"))
	assertNotExist(t, filepath.Join(dir, ".assay"))
	assertNotExist(t, filepath.Join(dir, "ASSAY.md"))
	assertNotExist(t, filepath.Join(agentsDir, "assay-test-author.md"))
}

func TestRunMigrateConfig_DryRunLeavesFilesUnchanged(t *testing.T) {
	dir := t.TempDir()
	touch(t, filepath.Join(dir, ".assay.yaml"))

	testChdir(t, dir)
	dryRun = true
	defer func() { dryRun = false }()

	if err := runMigrateConfig("assay", "scaffoldsmith"); err != nil {
		t.Fatalf("runMigrateConfig dry-run: %v", err)
	}

	assertExists(t, filepath.Join(dir, ".assay.yaml"))
	assertNotExist(t, filepath.Join(dir, ".scaffoldsmith.yaml"))
}

func TestRunMigrateConfig_NoArtifacts_ReturnsNil(t *testing.T) {
	dir := t.TempDir()
	testChdir(t, dir)

	if err := runMigrateConfig("assay", "scaffoldsmith"); err != nil {
		t.Errorf("expected nil for empty dir, got %v", err)
	}
}

// helpers

func touch(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, nil, 0o644); err != nil {
		t.Fatalf("touch %s: %v", path, err)
	}
}

func assertExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("expected %s to exist", path)
	}
}

func assertNotExist(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err == nil {
		t.Errorf("expected %s to not exist", path)
	}
}
