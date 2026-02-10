from pathlib import Path
from testsmith.core.discovery import (
    discover_untested_files,
    discover_files_in_path,
    is_source_file,
)
from testsmith.support.config import TestSmithConfig


def test_is_source_file():
    config = TestSmithConfig(exclude_dirs=["node_modules", ".venv"])
    root = Path("/project")

    # Valid
    assert is_source_file(root / "src/foo.py", root, config)
    assert is_source_file(root / "app.py", root, config)

    # Invalid extensions
    assert not is_source_file(root / "src/foo.txt", root, config)

    # Invalid names
    assert not is_source_file(root / "conftest.py", root, config)
    assert not is_source_file(root / "__init__.py", root, config)
    assert not is_source_file(root / "test_foo.py", root, config)
    assert not is_source_file(root / "foo_test.py", root, config)

    # Excluded dirs
    assert not is_source_file(root / "node_modules/foo.py", root, config)
    assert not is_source_file(root / ".venv/lib/site-packages/foo.py", root, config)


def test_discover_untested_files(tmp_path):
    # Setup
    # project/
    #   src/
    #     tested.py
    #     untested.py
    #   tests/
    #     src/
    #       test_tested.py

    project = tmp_path
    src = project / "src"
    src.mkdir()
    tests = project / "tests"
    tests_src = tests / "src"
    tests_src.mkdir(parents=True)

    (src / "tested.py").touch()
    (src / "untested.py").touch()
    (tests_src / "test_tested.py").touch()

    config = TestSmithConfig()

    files = discover_untested_files(project, tests, config)

    assert len(files) == 1
    assert files[0].name == "untested.py"


def test_discover_files_in_path(tmp_path):
    project = tmp_path
    src = project / "src"
    src.mkdir()

    (src / "one.py").touch()
    (src / "two.py").touch()

    config = TestSmithConfig()

    # Test directory discovery
    files = discover_files_in_path(src, project, project / "tests", config)
    assert len(files) == 2

    # Test single file discovery
    files = discover_files_in_path(src / "one.py", project, project / "tests", config)
    assert len(files) == 1
    assert files[0].name == "one.py"
