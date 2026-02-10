"""
Integration tests for binary build configuration.
"""
import pytest
import re
from pathlib import Path


def test_pyinstaller_spec_exists():
    """Test that PyInstaller spec file exists."""
    spec_file = Path("testsmith.spec")
    assert spec_file.exists(), "testsmith.spec file not found"


def test_pyinstaller_spec_is_valid():
    """Test that PyInstaller spec file is valid Python."""
    spec_file = Path("testsmith.spec")
    spec_content = spec_file.read_text(encoding="utf-8")
    
    # Should be valid Python
    try:
        compile(spec_content, "testsmith.spec", "exec")
    except SyntaxError as e:
        pytest.fail(f"testsmith.spec has syntax error: {e}")


def test_version_exists():
    """Test that __version__ is set."""
    from testsmith import __version__
    assert __version__, "__version__ is not set"


def test_version_format():
    """Test that __version__ matches semver format."""
    from testsmith import __version__
    
    # Semver pattern: X.Y.Z or X.Y.Z-prerelease
    semver_pattern = r'^\d+\.\d+\.\d+(-[a-zA-Z0-9.]+)?$'
    assert re.match(semver_pattern, __version__), \
        f"__version__ '{__version__}' does not match semver format"


def test_version_flag_works(tmp_path):
    """Test that --version flag outputs version string."""
    import subprocess
    import sys
    
    # Run testsmith --version
    result = subprocess.run(
        [sys.executable, "-m", "testsmith", "--version"],
        capture_output=True,
        text=True,
        cwd=Path.cwd()
    )
    
    assert result.returncode == 0, f"--version flag failed: {result.stderr}"
    
    # Should contain "testsmith" and version number
    output = result.stdout.strip()
    assert "testsmith" in output.lower(), f"Output doesn't contain 'testsmith': {output}"
    assert re.search(r'\d+\.\d+\.\d+', output), f"Output doesn't contain version number: {output}"


def test_build_scripts_exist():
    """Test that build scripts exist."""
    bash_script = Path("scripts/build-local.sh")
    ps_script = Path("scripts/build-local.ps1")
    
    assert bash_script.exists(), "scripts/build-local.sh not found"
    assert ps_script.exists(), "scripts/build-local.ps1 not found"


def test_bash_script_is_executable():
    """Test that bash script is executable."""
    import os
    bash_script = Path("scripts/build-local.sh")
    
    # Check if file has execute permission
    is_executable = os.access(bash_script, os.X_OK)
    assert is_executable, "scripts/build-local.sh is not executable"


def test_gitignore_includes_build_artifacts():
    """Test that .gitignore includes PyInstaller artifacts."""
    gitignore = Path(".gitignore")
    assert gitignore.exists(), ".gitignore not found"
    
    content = gitignore.read_text(encoding="utf-8")
    
    # Should ignore build artifacts
    assert "dist/" in content, ".gitignore doesn't include dist/"
    assert "build/" in content, ".gitignore doesn't include build/"


def test_ci_workflow_exists():
    """Test that CI workflow exists."""
    ci_workflow = Path(".github/workflows/ci.yml")
    assert ci_workflow.exists(), "CI workflow not found"


def test_release_workflow_exists():
    """Test that release workflow exists."""
    release_workflow = Path(".github/workflows/release.yml")
    assert release_workflow.exists(), "Release workflow not found"


def test_releasing_docs_exist():
    """Test that RELEASING.md exists."""
    releasing_docs = Path("RELEASING.md")
    assert releasing_docs.exists(), "RELEASING.md not found"
