"""
Discovery mechanisms for finding source files to test.
"""

from pathlib import Path
from testsmith.support.config import TestSmithConfig
from testsmith.generation.test_generator import derive_test_path


def is_source_file(path: Path, project_root: Path, config: TestSmithConfig) -> bool:
    """
    Check if a file is a valid source file for testing.
    Rules:
    - Must be a .py file
    - Must not be in excluded directories
    - Must not be a test file (starts with test_ or ends with _test.py)
    - Must not be conftest.py or __init__.py
    """
    if path.suffix != ".py":
        return False

    if path.name in ("conftest.py", "__init__.py"):
        return False

    # Check simplified test file naming convention
    # This might need to be more robust if config defines test pattern
    if path.name.startswith("test_") or path.name.endswith("_test.py"):
        return False

    # Check exclusion dirs
    # We check if any part of the relative path is in exclude_dirs
    try:
        rel_path = path.relative_to(project_root)
        for part in rel_path.parts:
            if part in config.exclude_dirs:
                return False
    except ValueError:
        pass  # path not relative to root?

    return True


def discover_untested_files(
    project_root: Path, test_root: Path, config: TestSmithConfig
) -> list[Path]:
    """
    Find all source files in the project that do not have a corresponding test file.
    """
    untested_files = []

    # Iterate over all files in project root
    for file_path in project_root.rglob("*.py"):
        if not is_source_file(file_path, project_root, config):
            continue

        # Check against test root to ensure we don't pick up files inside tests/ if they happen to look like source
        # Although default structure separates them, some projects might mix.
        # If file is INSIDE test_root (e.g. tests/helpers.py), we skip it as it's test support code.
        try:
            file_path.relative_to(test_root)
            continue  # It is inside test_root
        except ValueError:
            pass  # It is not inside test_root

        # Logic: Calculate expected test path. If it doesn't exist, it's untested.
        # Note: derive_test_path requires a ProjectContext-like object or we simulate it?
        # Actually `derive_test_path` signature is `(source_path: Path, project_root: Path, config: TestSmithConfig)`.
        # Let's check `test_generator.py` signature.

        # Wait, I need to verify `derive_test_path` signature.
        # It's likely `derive_test_path(source_path, project_root, config)`.

        test_path = derive_test_path(file_path, project_root, config)
        if not test_path.exists():
            untested_files.append(file_path)

    return sorted(untested_files)


def discover_files_in_path(
    target_path: Path, project_root: Path, test_root: Path, config: TestSmithConfig
) -> list[Path]:
    """
    Find all source files in a specific directory that DO NOT have a test file.
    """
    untested_files = []

    if target_path.is_file():
        if is_source_file(target_path, project_root, config):
            test_path = derive_test_path(target_path, project_root, config)
            if not test_path.exists():
                return [target_path]
        return []

    for file_path in target_path.rglob("*.py"):
        if not is_source_file(file_path, project_root, config):
            continue

        # Same check for being inside test root
        try:
            file_path.relative_to(test_root)
            continue
        except ValueError:
            pass

        test_path = derive_test_path(file_path, project_root, config)
        if not test_path.exists():
            untested_files.append(file_path)

    return sorted(untested_files)
