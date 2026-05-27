package generation_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/orieken/testsmith/internal/domain"
	"github.com/orieken/testsmith/internal/generation"
)

// ---- ScanUsedDependencies ----------------------------------------------------

func TestScanUsedDependencies(t *testing.T) {
	analyses := []*domain.SourceAnalysis{
		{Imports: domain.ClassifiedImports{External: []domain.ImportInfo{
			{Module: "stripe"},
			{Module: "stripe.error"},
			{Module: "requests"},
		}}},
		{Imports: domain.ClassifiedImports{External: []domain.ImportInfo{
			{Module: "boto3"},
		}}},
	}

	used := generation.ScanUsedDependencies(analyses)
	for _, want := range []string{"stripe", "requests", "boto3"} {
		if !used[want] {
			t.Errorf("expected %q in used set", want)
		}
	}
	if used["stripe.error"] {
		t.Error("sub-module key should be root only (stripe), not stripe.error")
	}
}

// ---- ScanExistingFixtures ----------------------------------------------------

func TestScanExistingFixtures_Empty(t *testing.T) {
	fixtures, err := generation.ScanExistingFixtures("", domain.TestFrameworkConfig{})
	if err != nil || len(fixtures) != 0 {
		t.Errorf("empty dir should return nil, nil; got %v, %v", fixtures, err)
	}
}

func TestScanExistingFixtures_Discovers(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"stripe_fixture.py", "requests_fixture.py", "unrelated.py"} {
		_ = os.WriteFile(filepath.Join(dir, name), []byte(""), 0o644)
	}

	cfg := domain.TestFrameworkConfig{FixtureSuffix: "_fixture.py"}
	fixtures, err := generation.ScanExistingFixtures(dir, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(fixtures) != 2 {
		t.Errorf("expected 2 fixtures, got %d", len(fixtures))
	}
	names := make(map[string]bool)
	for _, f := range fixtures {
		names[f.DepName] = true
	}
	if !names["stripe"] || !names["requests"] {
		t.Errorf("expected stripe and requests dep names, got %v", names)
	}
}

// ---- IdentifyUnused ----------------------------------------------------------

func TestIdentifyUnused(t *testing.T) {
	used := map[string]bool{"stripe": true, "requests": true}
	fixtures := []generation.FixtureFile{
		{DepName: "stripe"},
		{DepName: "boto3"},
		{DepName: "requests"},
		{DepName: "redis"},
	}
	unused := generation.IdentifyUnused(used, fixtures)
	if len(unused) != 2 {
		t.Errorf("expected 2 unused, got %d", len(unused))
	}
	for _, f := range unused {
		if f.DepName == "stripe" || f.DepName == "requests" {
			t.Errorf("used dep %q should not be in unused list", f.DepName)
		}
	}
}

// ---- PruneFixtures -----------------------------------------------------------

func TestPruneFixtures_DryRun(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "boto3_fixture.py")
	_ = os.WriteFile(path, []byte(""), 0o644)

	unused := []generation.FixtureFile{{AbsPath: path, DepName: "boto3"}}
	results := generation.PruneFixtures(unused, true)

	if len(results) != 1 || results[0].Action != "skipped" {
		t.Errorf("dry-run should skip; got %+v", results)
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("dry-run must not delete the file")
	}
}

func TestPruneFixtures_Delete(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "redis_fixture.py")
	_ = os.WriteFile(path, []byte(""), 0o644)

	unused := []generation.FixtureFile{{AbsPath: path, DepName: "redis"}}
	results := generation.PruneFixtures(unused, false)

	if len(results) != 1 || results[0].Action != "deleted" {
		t.Errorf("expected deleted action; got %+v", results)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("file should have been deleted")
	}
}

// ---- UpdateTestImports -------------------------------------------------------

func TestUpdateTestImports_CommentsOutStaleImport(t *testing.T) {
	dir := t.TempDir()
	testFile := filepath.Join(dir, "test_payments.py")
	content := "import pytest\nfrom tests.fixtures.redis_fixture import mock_redis\n\ndef test_foo(): pass\n"
	_ = os.WriteFile(testFile, []byte(content), 0o644)

	modified, err := generation.UpdateTestImports(dir, []string{"redis"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(modified) != 1 {
		t.Errorf("expected 1 modified file, got %d", len(modified))
	}

	data, _ := os.ReadFile(testFile)
	if !strings.Contains(string(data), "# [testsmith-pruned]") {
		t.Error("stale import line should be commented out with [testsmith-pruned] prefix")
	}
}

func TestUpdateTestImports_NoMatch(t *testing.T) {
	dir := t.TempDir()
	testFile := filepath.Join(dir, "test_payments.py")
	content := "import pytest\nfrom tests.fixtures.stripe_fixture import mock_stripe\n"
	_ = os.WriteFile(testFile, []byte(content), 0o644)

	modified, err := generation.UpdateTestImports(dir, []string{"redis"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(modified) != 0 {
		t.Errorf("no match expected, got %d modified", len(modified))
	}
}
