import pytest
from testsmith.core.project_detector import (
    find_project_root,
    scan_packages,
    detect_conftest,
    build_project_context,
)
from testsmith.support.exceptions import ProjectRootNotFoundError
from testsmith.support.config import TestSmithConfig as Config


def test_find_root_pyproject(tmp_path):
    (tmp_path / "pyproject.toml").touch()
    assert find_project_root(tmp_path) == tmp_path


def test_find_root_nested(tmp_path):
    (tmp_path / ".git").mkdir()
    sub = tmp_path / "src" / "my_app"
    sub.mkdir(parents=True)
    assert find_project_root(sub) == tmp_path


def test_find_root_fails(tmp_path):
    # Ensure no markers exist up (assuming test env is clean)
    with pytest.raises(ProjectRootNotFoundError):
        find_project_root(tmp_path / "somewhere")


def test_scan_packages_flat(tmp_path):
    (tmp_path / "pkg_a").mkdir()
    (tmp_path / "pkg_a" / "__init__.py").touch()
    (tmp_path / "utils.py").touch()

    pkg_map = scan_packages(tmp_path, exclude_dirs=[])
    assert "pkg_a" in pkg_map
    assert pkg_map["pkg_a"] == tmp_path / "pkg_a"
    assert "utils" in pkg_map
    assert pkg_map["utils"] == tmp_path / "utils.py"


def test_scan_packages_src_layout(tmp_path):
    src = tmp_path / "src"
    src.mkdir()
    (src / "app").mkdir()
    (src / "app" / "__init__.py").touch()

    pkg_map = scan_packages(tmp_path, exclude_dirs=[])
    assert "app" in pkg_map
    assert pkg_map["app"] == src / "app"


def test_detect_conftest_empty(tmp_path):
    assert detect_conftest(tmp_path) == (None, [])


def test_detect_conftest_with_paths(tmp_path):
    cf = tmp_path / "conftest.py"
    cf.write_text("paths_to_add = ['src/', 'tests/']", encoding="utf-8")

    path, paths = detect_conftest(tmp_path)
    assert path == cf
    assert paths == ["src/", "tests/"]


def test_build_context(tmp_path):
    (tmp_path / "pyproject.toml").touch()
    (tmp_path / "src").mkdir()
    (tmp_path / "src" / "main.py").touch()

    cfg = Config()
    ctx = build_project_context(tmp_path / "src" / "main.py", cfg)

    assert ctx.root == tmp_path

    assert ctx.conftest_path is None


def test_build_context_configured_root(tmp_path):
    """Test that explicit root config bypasses marker detection."""
    # No markers at tmp_path, generally find_project_root would fail
    src = tmp_path / "src"
    src.mkdir()
    (src / "main.py").touch()

    # Provide explicit root
    cfg = Config(root=str(tmp_path))
    ctx = build_project_context(src / "main.py", cfg)

    assert ctx.root == tmp_path


def test_build_context_custom_conftest(tmp_path):
    """Test using a custom conftest path."""
    (tmp_path / "pyproject.toml").touch()

    # Create custom conftest location
    fixtures_dir = tmp_path / "tests" / "fixtures"
    fixtures_dir.mkdir(parents=True)
    conftest = fixtures_dir / "conftest.py"
    conftest.write_text("paths_to_add = []", encoding="utf-8")

    # Configure usage
    cfg = Config()
    cfg.conftest_path = "tests/fixtures/conftest.py"

    # Also need source file
    src = tmp_path / "main.py"
    src.touch()

    ctx = build_project_context(src, cfg)

    assert ctx.conftest_path == conftest
