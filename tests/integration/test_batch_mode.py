
import sys
import shutil
import pytest
from pathlib import Path
from unittest.mock import patch, MagicMock
from testsmith.cli import main

@pytest.fixture
def temp_project(tmp_path):
    """
    Creates a temporary project structure:
    project/
      pyproject.toml
      src/
        service.py       (untested)
        utils.py         (untested)
        core/
          models.py      (untested)
      tests/
        conftest.py
    """
    project_root = tmp_path / "batch_project"
    project_root.mkdir()
    
    (project_root / "pyproject.toml").write_text("""
[tool.poetry]
name = "batch_project"
version = "0.1.0"
[tool.testsmith]
test_root = "tests/"
""", encoding="utf-8")

    src = project_root / "src"
    src.mkdir()
    (src / "service.py").write_text("def service(): pass", encoding="utf-8")
    (src / "utils.py").write_text("def util(): pass", encoding="utf-8")
    
    models_dir = src / "core"
    models_dir.mkdir()
    (models_dir / "models.py").write_text("class Model: pass", encoding="utf-8")
    
    tests = project_root / "tests"
    tests.mkdir()
    (tests / "conftest.py").write_text("", encoding="utf-8")
    
    return project_root

def test_batch_all(temp_project, capsys):
    """Test --all processes all untested files."""
    project_root = temp_project
    
    with patch("sys.argv", ["testsmith", "--all"]), \
         patch("pathlib.Path.cwd", return_value=project_root):
             
        try:
            main()
        except SystemExit as e:
            assert e.code == 0
            
    captured = capsys.readouterr()
    
    assert "TestSmith Batch Summary" in captured.out
    assert "Processed 3 source files" in captured.out
    assert "Created:  3 test files" in captured.out
    
    # Verify files created
    assert (project_root / "tests/src/test_service.py").exists()
    assert (project_root / "tests/src/test_utils.py").exists()
    assert (project_root / "tests/src/core/test_models.py").exists()

def test_batch_path(temp_project, capsys):
    """Test --path scope."""
    project_root = temp_project
    
    # Run only on src/core
    target_path = project_root / "src" / "core"
    
    with patch("sys.argv", ["testsmith", "--path", str(target_path)]), \
         patch("pathlib.Path.cwd", return_value=project_root):
             
        try:
            main()
        except SystemExit as e:
            assert e.code == 0
            
    captured = capsys.readouterr()
    print(captured.out)
    
    assert "TestSmith Batch Summary" in captured.out
    assert "Processed 1 source files" in captured.out
    assert "Created:  1 test files" in captured.out
    
    assert (project_root / "tests/src/core/test_models.py").exists()
    assert not (project_root / "tests/src/test_service.py").exists()

def test_batch_skip_existing(temp_project, capsys):
    """Test that existing tests are skipped in batch mode."""
    project_root = temp_project
    
    # Create one test file beforehand
    test_service = project_root / "tests/src/test_service.py"
    test_service.parent.mkdir(parents=True, exist_ok=True)
    test_service.touch()
    
    with patch("sys.argv", ["testsmith", "--all"]), \
         patch("pathlib.Path.cwd", return_value=project_root):
             
        try:
            main()
        except SystemExit as e:
            assert e.code == 0
            
    captured = capsys.readouterr()
    
    # service.py is NOT listed as candidate because discovery checks existence?
    # Wait, discovery Logic: "if not test_path.exists(): untested_files.append()"
    # So if test exists, verify logic should skip it efficiently or discovery won't even return it.
    # discovery implementation: `if not test_path.exists()` -> returns it.
    # So `service.py` should NOT be in the processed list.
    
    assert "Processed 2 source files" in captured.out # utils.py and models.py
    assert "Created:  2 test files" in captured.out
    
    # But skipped count? 
    # If discovery filters them out, CLI won't even see them to say "Skipped".
    # That's fine, "Processed" implies work attempted.
    # If users want to know coverage, --coverage-gaps (Prompt 14) will show status.
    # Or if we want "Skipped" in summary, we'd need discovery to return "Tested" files too?
    # Prompt 10 Requirements: "Scan project for all .py source files... Return list of source files that have NO corresponding test file"
    # So CLI summary "Skipped: 4 test files (already exist)" implies logic change or clarification.
    # If discovery ONLY returns untested, CLI can't report skipped.
    # UNLESS CLI also runs a check? 
    # Let's check my implementation. `discovery.py` filters `if not test_path.exists()`.
    # So `Processed` will only be untested files.
    # The requirement "Skipped: 4 test files" in summary might imply we should check everything?
    # "Process: Collect list of source files to process" -> "Process each sequentially"
    # If I want to report "skipped", I should probably let `discover_untested_files` return everything and let CLI skip?
    # But performance...
    # Re-reading Prompt 10 TODO: "Return list of source files that have NO corresponding test file".
    # So discovery filters.
    # Where did the summary "Skipped: 4 test files" come from in the Prompt?
    # Maybe "TestSmith Batch Summary" example was generic?
    # Or maybe "untested" means "needs generation".
    # If CLI `run` logic calls `generate_test`, and that returns "skipped", then we count it.
    # But if discovery filters it out, `generate_test` is never called.
    # I will stick to discovery filtering for efficiency. The summary will reflect files actually processed (created/error). 
    # If 0 files processed because all tested, it says "Found 0 files".

    # So in this test, we expect 2 processed.
    pass

def test_batch_error_resilience(temp_project, capsys):
    """Test that one file failure doesn't stop batch."""
    project_root = temp_project
    
    # Valid file
    (project_root / "src/valid.py").write_text("def foo(): pass", encoding="utf-8")
    
    # Malformed file (syntax error)
    (project_root / "src/malformed.py").write_text("def broken(:", encoding="utf-8")
    
    with patch("sys.argv", ["testsmith", "--all"]), \
         patch("pathlib.Path.cwd", return_value=project_root):
        
        # We expect exit code 1 because at least one failed
        try:
            main()
        except SystemExit as e:
            assert e.code == 1
            
    captured = capsys.readouterr()
    
    assert "Processed 5 source files" in captured.out # service, utils, models, valid, malformed
    assert "Errors:   1 files failed" in captured.out
    assert "malformed.py" in captured.out
    assert "Created:  4 test files" in captured.out

