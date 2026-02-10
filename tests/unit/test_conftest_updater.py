import pytest
from pathlib import Path
from testsmith.generation.conftest_updater import (
    parse_paths_to_add,
    compute_required_paths,
    diff_paths,
    update_conftest
)
from testsmith.support.config import TestSmithConfig

@pytest.fixture
def config():
    return TestSmithConfig()

def test_parse_paths_to_add():
    content = """
def pytest_configure(config):
    paths_to_add = [
        "src",
        "tests/fixtures",
    ]
    """
    paths, start, end = parse_paths_to_add(content, "paths_to_add")
    assert "src" in paths
    assert "tests/fixtures" in paths
    assert len(paths) == 2
    assert start > 0
    assert end > start

    # Not found
    assert parse_paths_to_add("x = 1", "paths_to_add") == ([], -1, -1)

def test_compute_required_paths(tmp_path):
    root = tmp_path
    src = root / "src/pkg/mod.py"
    test = root / "tests/src/pkg/test_mod.py"
    fix = root / "tests/fixtures"
    
    paths = compute_required_paths(src, test, fix, root)
    assert "src" in paths
    assert "tests" in paths
    assert "tests/fixtures" in paths
    
def test_diff_paths():
    existing = ["src", "tests/fixtures"]
    required = ["src", "tests", "tests/fixtures", "new/path"]
    
    missing = diff_paths(existing, required)
    assert missing == ["tests", "new/path"]
    
    # Check normalization
    assert diff_paths(["src/"], ["src"]) == []

def test_update_conftest_create(tmp_path, config):
    conftest = tmp_path / "conftest.py"
    new_paths = ["src", "tests"]
    
    path, action = update_conftest(conftest, new_paths, config)
    assert action == "created"
    assert path.exists()
    content = path.read_text()
    assert 'paths_to_add = [' in content
    assert '"src",' in content
    assert '"tests",' in content
    assert "def pytest_configure" in content

def test_update_conftest_append_new(tmp_path, config):
    conftest = tmp_path / "conftest.py"
    conftest.write_text("# Existing file\n", encoding="utf-8")
    
    new_paths = ["src"]
    path, action = update_conftest(conftest, new_paths, config)
    assert action == "updated"
    content = path.read_text()
    assert "# Existing file" in content
    assert "def pytest_configure" in content
    assert '"src"' in content

def test_update_conftest_insert_existing(tmp_path, config):
    conftest = tmp_path / "conftest.py"
    conftest.write_text("""
def pytest_configure(config):
    paths_to_add = [
        "old",
    ]
""", encoding="utf-8")
    
    new_paths = ["new"]
    path, action = update_conftest(conftest, new_paths, config)
    assert action == "updated"
    content = path.read_text()
    assert '"old",' in content
    assert '"new", # Added by TestSmith' in content
    
    # Check skipped
    path, action = update_conftest(conftest, ["old", "new"], config)
    assert action == "skipped"

def test_update_conftest_single_line(tmp_path, config):
    conftest = tmp_path / "conftest.py"
    conftest.write_text('paths_to_add = ["a"]', encoding="utf-8")
    
    path, action = update_conftest(conftest, ["b"], config)
    assert action == "updated"
    content = path.read_text()
    # Expect: paths_to_add = ["a", "b", # Added by TestSmith] or similar
    # The simple injection might result in: methods=["a", "b", # Added...]
    assert '"b", # Added by TestSmith' in content
    assert '"a"' in content
