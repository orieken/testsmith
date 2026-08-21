package python

import (
	"testing"
)

// ── FrameworkConfig ────────────────────────────────────────────────────────────

// backfill / AC: pytestPytestMockAdapter.FrameworkConfig returns pytest naming conventions
func TestPytestMockAdapter_FrameworkConfig(t *testing.T) {
	t.Parallel()
	a := &pytestPytestMockAdapter{}
	cfg := a.FrameworkConfig()
	if cfg.Name != "pytest" {
		t.Errorf("FrameworkConfig().Name = %q, want %q", cfg.Name, "pytest")
	}
	if cfg.TestFilePrefix != "test_" {
		t.Errorf("FrameworkConfig().TestFilePrefix = %q, want %q", cfg.TestFilePrefix, "test_")
	}
	if cfg.TestFileSuffix != ".py" {
		t.Errorf("FrameworkConfig().TestFileSuffix = %q, want %q", cfg.TestFileSuffix, ".py")
	}
	if cfg.FixtureDir != "tests/fixtures/" {
		t.Errorf("FrameworkConfig().FixtureDir = %q, want %q", cfg.FixtureDir, "tests/fixtures/")
	}
	if cfg.FixtureSuffix != "_fixture.py" {
		t.Errorf("FrameworkConfig().FixtureSuffix = %q, want %q", cfg.FixtureSuffix, "_fixture.py")
	}
	if cfg.BootstrapFile != "conftest.py" {
		t.Errorf("FrameworkConfig().BootstrapFile = %q, want %q", cfg.BootstrapFile, "conftest.py")
	}
	if cfg.TestFuncPrefix != "test_" {
		t.Errorf("FrameworkConfig().TestFuncPrefix = %q, want %q", cfg.TestFuncPrefix, "test_")
	}
}

// backfill / AC: pytestUnittestMockAdapter.FrameworkConfig delegates to pytestPytestMockAdapter via toBase
func TestPytestUnittestMockAdapter_FrameworkConfig(t *testing.T) {
	t.Parallel()
	a := &pytestUnittestMockAdapter{}
	cfg := a.FrameworkConfig()
	if cfg.Name != "pytest" {
		t.Errorf("FrameworkConfig().Name = %q, want %q", cfg.Name, "pytest")
	}
	if cfg.BootstrapFile != "conftest.py" {
		t.Errorf("FrameworkConfig().BootstrapFile = %q, want %q", cfg.BootstrapFile, "conftest.py")
	}
	if cfg.TestFilePrefix != "test_" {
		t.Errorf("FrameworkConfig().TestFilePrefix = %q, want %q", cfg.TestFilePrefix, "test_")
	}
}

// backfill / AC: pytestUnittestMockAdapter.toBase returns a pytestPytestMockAdapter
func TestPytestUnittestMockAdapter_ToBase(t *testing.T) {
	t.Parallel()
	a := &pytestUnittestMockAdapter{}
	base := a.toBase()
	if base == nil {
		t.Fatal("toBase() must not return nil")
	}
	cfg := base.FrameworkConfig()
	if cfg.Name != "pytest" {
		t.Errorf("toBase().FrameworkConfig().Name = %q, want %q", cfg.Name, "pytest")
	}
}

// backfill / AC: unittestAdapter.FrameworkConfig returns unittest-specific naming conventions
func TestUnittestAdapter_FrameworkConfig(t *testing.T) {
	t.Parallel()
	a := &unittestAdapter{}
	cfg := a.FrameworkConfig()
	if cfg.Name != "unittest" {
		t.Errorf("FrameworkConfig().Name = %q, want %q", cfg.Name, "unittest")
	}
	if cfg.BootstrapFile != "" {
		t.Errorf("FrameworkConfig().BootstrapFile = %q, want empty (unittest uses no conftest)", cfg.BootstrapFile)
	}
	if cfg.FixtureDir != "tests/" {
		t.Errorf("FrameworkConfig().FixtureDir = %q, want %q", cfg.FixtureDir, "tests/")
	}
	if cfg.FixtureSuffix != "_test.py" {
		t.Errorf("FrameworkConfig().FixtureSuffix = %q, want %q", cfg.FixtureSuffix, "_test.py")
	}
	if cfg.TestFuncPrefix != "test_" {
		t.Errorf("FrameworkConfig().TestFuncPrefix = %q, want %q", cfg.TestFuncPrefix, "test_")
	}
}

// ── LLMVocabulary ─────────────────────────────────────────────────────────────

// backfill / AC: pytestPytestMockAdapter.LLMVocabulary returns pytest-mock vocabulary keys
func TestPytestMockAdapter_LLMVocabulary(t *testing.T) {
	t.Parallel()
	a := &pytestPytestMockAdapter{}
	v := a.LLMVocabulary()
	if v["framework"] != "pytest" {
		t.Errorf("LLMVocabulary()[\"framework\"] = %q, want %q", v["framework"], "pytest")
	}
	if v["mock_library"] != "pytest-mock" {
		t.Errorf("LLMVocabulary()[\"mock_library\"] = %q, want %q", v["mock_library"], "pytest-mock")
	}
	if v["assert_style"] == "" {
		t.Error("LLMVocabulary()[\"assert_style\"] must not be empty")
	}
	if v["mock_style"] == "" {
		t.Error("LLMVocabulary()[\"mock_style\"] must not be empty")
	}
	if v["fixture_decorator"] != "@pytest.fixture" {
		t.Errorf("LLMVocabulary()[\"fixture_decorator\"] = %q, want %q", v["fixture_decorator"], "@pytest.fixture")
	}
}

// backfill / AC: pytestUnittestMockAdapter.LLMVocabulary returns unittest.mock vocabulary keys
func TestPytestUnittestMockAdapter_LLMVocabulary(t *testing.T) {
	t.Parallel()
	a := &pytestUnittestMockAdapter{}
	v := a.LLMVocabulary()
	if v["framework"] != "pytest" {
		t.Errorf("LLMVocabulary()[\"framework\"] = %q, want %q", v["framework"], "pytest")
	}
	if v["mock_library"] != "unittest.mock" {
		t.Errorf("LLMVocabulary()[\"mock_library\"] = %q, want %q", v["mock_library"], "unittest.mock")
	}
	if v["fixture_decorator"] != "@patch" {
		t.Errorf("LLMVocabulary()[\"fixture_decorator\"] = %q, want %q", v["fixture_decorator"], "@patch")
	}
	if v["assert_style"] == "" {
		t.Error("LLMVocabulary()[\"assert_style\"] must not be empty")
	}
}

// backfill / AC: unittestAdapter.LLMVocabulary returns unittest vocabulary with assertEqual style
func TestUnittestAdapter_LLMVocabulary(t *testing.T) {
	t.Parallel()
	a := &unittestAdapter{}
	v := a.LLMVocabulary()
	if v["framework"] != "unittest" {
		t.Errorf("LLMVocabulary()[\"framework\"] = %q, want %q", v["framework"], "unittest")
	}
	if v["mock_library"] != "unittest.mock" {
		t.Errorf("LLMVocabulary()[\"mock_library\"] = %q, want %q", v["mock_library"], "unittest.mock")
	}
	if v["assert_style"] == "" {
		t.Error("LLMVocabulary()[\"assert_style\"] must not be empty")
	}
	if v["mock_style"] == "" {
		t.Error("LLMVocabulary()[\"mock_style\"] must not be empty")
	}
	if v["fixture_decorator"] != "@patch" {
		t.Errorf("LLMVocabulary()[\"fixture_decorator\"] = %q, want %q", v["fixture_decorator"], "@patch")
	}
}
