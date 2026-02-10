"""
Integration tests for LLM features in CLI.
"""
import pytest
from unittest.mock import patch
from testsmith.cli import main

@pytest.fixture
def sample_project(tmp_path):
    project_root = tmp_path / "sample_project"
    project_root.mkdir()
    (project_root / "pyproject.toml").write_text('[tool.testsmith]\n', encoding="utf-8")
    src_dir = project_root / "src" / "pkg"
    src_dir.mkdir(parents=True)
    (src_dir / "mod.py").write_text("def foo(): pass", encoding="utf-8")
    return project_root, src_dir / "mod.py"

def test_cli_generate_bodies_flag(sample_project, capsys):
    """Test that --generate-bodies triggers LLM generation."""
    project_root, source_file = sample_project
    
    # Mock LLM generator to avoid real API calls and dependency issues
    with patch("testsmith.cli.generate_test_bodies") as mock_gen, \
         patch("sys.argv", ["testsmith", str(source_file), "--generate-bodies"]), \
         patch("pathlib.Path.cwd", return_value=project_root):
        
        # Setup mock return
        mock_gen.return_value = {"foo": ["def test_foo():", "    assert True"]}
        
        try:
            main()
        except SystemExit as e:
            assert e.code == 0
            
        # Verify it was called
        mock_gen.assert_called_once()
        
        # Verify file content
        test_file = project_root / "tests/src/pkg/test_mod.py"
        assert test_file.exists()
        content = test_file.read_text()
        assert "def test_foo():" in content
        assert "assert True" in content
        assert "# TODO" not in content

def test_cli_generate_bodies_missing_key(sample_project, capsys):
    """Test behavior when API key is missing (or library missing)."""
    project_root, source_file = sample_project
    
    # Simulate error in generator
    with patch("testsmith.cli.generate_test_bodies", side_effect=Exception("API Key missing")), \
         patch("sys.argv", ["testsmith", str(source_file), "--generate-bodies"]), \
         patch("pathlib.Path.cwd", return_value=project_root):
             
        try:
            main()
        except SystemExit as e:
            assert e.code == 0 # Should fallback to stubs, not crash
            
        captured = capsys.readouterr()
        assert "Warning: LLM generation failed" in captured.out
        assert "Falling back to stubs" in captured.out
        
        # Verify file content (stubs)
        test_file = project_root / "tests/src/pkg/test_mod.py"
        content = test_file.read_text()
        assert "# TODO" in content
