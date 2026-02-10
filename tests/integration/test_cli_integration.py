"""
Integration tests for TestSmith CLI.
"""
import pytest
import sys
from pathlib import Path
from unittest.mock import patch
from testsmith.cli import main

@pytest.fixture
def sample_project(tmp_path):
    """
    Create a sample project structure in tmp_path.
    """
    project_root = tmp_path / "sample_project"
    project_root.mkdir()
    
    # pyproject.toml
    (project_root / "pyproject.toml").write_text("""
[tool.poetry]
name = "sample_project"
version = "0.1.0"
""", encoding="utf-8")
    
    # Source code
    src_dir = project_root / "src" / "sample"
    src_dir.mkdir(parents=True)
    
    source_file = src_dir / "user_service.py"
    source_file.write_text("""
import requests
from sample.db import Database

class UserService:
    def __init__(self, db: Database):
        self.db = db
        
    def get_user(self, user_id: int):
        resp = requests.get(f"https://api.example.com/users/{user_id}")
        return resp.json()
""", encoding="utf-8")

    # Dependency mock for analysis (simulated existence/import)
    # The analyzer checks if `sample.db` is internal. 
    # It just needs file existence or import classification.
    # `sample.db` will be classified internal if `src/sample/db.py` exists?
    # Or just based on project root.
    # We should create `src/sample/db.py` to be safe.
    (src_dir / "db.py").write_text("class Database: pass", encoding="utf-8")
    
    return project_root, source_file

def test_cli_end_to_end(sample_project, capsys):
    """
    Test the full flow:
    1. Detect project
    2. Analyze user_service.py
    3. Generate fixtures (requests)
    4. Generate test file (test_user_service.py)
    5. Update conftest.py
    """
    project_root, source_file = sample_project
    
    # Run CLI
    # We mock sys.argv
    # testsmith user_service.py --config pyproject.toml (implicit via root detection?)
    # We simulate running from project root.
    
    with patch("sys.argv", ["testsmith", str(source_file)]), \
         patch("pathlib.Path.cwd", return_value=project_root):
        
        # We expect exit code 0
        try:
            main()
        except SystemExit as e:
            assert e.code == 0, f"CLI exited with {e.code}"
            
    # Check Output
    captured = capsys.readouterr()
    print(captured.out) # Print to stdout for -s inspection if needed
    
    # New summary format checks - verify paths are present in output
    assert "TestSmith Summary" in captured.out, f"Summary header missing. Output:\n{captured.out}"
    assert "tests/fixtures/requests_fixture.py" in captured.out, f"Fixture output missing. Output:\n{captured.out}"
    assert "tests/src/sample/test_user_service.py" in captured.out, f"Test file output missing. Output:\n{captured.out}"
    assert "conftest.py" in captured.out, f"Conftest output missing. Output:\n{captured.out}"
    
    # Verify Files
    
    # 1. Fixture
    fixture_path = project_root / "tests/fixtures/requests_fixture.py" # fixture_suffix? default is ".fixture.py"?
    # Config default: fixture_suffix = ".fixture.py" -> `requests.fixture.py`
    # Wait, `fixture_generator` uses `derive_fixture_filename`.
    # default suffix in config is `.fixture.py`.
    # name is `requests`. -> `requests.fixture.py`?
    # Check `fixture_generator.py` logic.
    # `return fixture_dir / f"{fixture_name}{config.fixture_suffix}"`
    # fixture_name derived from dependency `requests` -> `requests`.
    # So `requests.fixture.py`.
    
    
    fixture_path = project_root / "tests/fixtures/requests_fixture.py"
    assert fixture_path.exists()
    assert "mock_requests" in fixture_path.read_text()
    
    # 2. Test File
    # src/sample/user_service.py -> tests/src/sample/test_user_service.py
    test_path = project_root / "tests/src/sample/test_user_service.py"
    assert test_path.exists()
    content = test_path.read_text()
    assert "class TestUserService" in content
    assert "mock_requests" in content # derived from fixture params
    
    # 3. Conftest
    conftest_path = project_root / "conftest.py"
    assert conftest_path.exists()
    c_content = conftest_path.read_text()
    assert "paths_to_add" in c_content
    assert "tests/fixtures" in c_content
    assert "src" in c_content

def test_cli_dry_run(sample_project, capsys):
    project_root, source_file = sample_project
    
    with patch("sys.argv", ["testsmith", str(source_file), "--dry-run"]), \
         patch("pathlib.Path.cwd", return_value=project_root):
        
        try:
            main()
        except SystemExit as e:
            assert e.code == 0
            
    captured = capsys.readouterr()
    # assert "[DRY RUN]" in captured.out # Only shows in verbose now
    assert "Dry-run" in captured.out # Summary check
    
    # Verify NO files created
    assert not (project_root / "tests").exists()
    assert not (project_root / "conftest.py").exists()

def test_cli_file_not_found(sample_project, capsys):
    project_root, _ = sample_project
    
    with patch("sys.argv", ["testsmith", "nonexistent.py"]), \
         patch("pathlib.Path.cwd", return_value=project_root):
        
        try:
            main()
        except SystemExit as e:
            assert e.code == 1
            
    captured = capsys.readouterr()
    assert "Source file not found" in captured.out

def test_cli_no_args(capsys):
    """Test no arguments (should print help or usage)."""
    # Argparse handles this and might exit(2)
    # We patch sys.stderr to avoid polluting output?
    
    # Actually just running with empty args usually shows errors.
    # "testsmith" without file?
    # `parser.add_argument("source_file", nargs="?")`
    # It's optional?
    # BUT `if not args.source_file:` check in `run` returns 1.
    
    with patch("sys.argv", ["testsmith"]):
         try:
            main()
         except SystemExit as e:
            assert e.code == 1
    
    captured = capsys.readouterr()
    assert "Error: You must provide a source file, --all, or --path" in captured.out
