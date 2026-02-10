"""
Fixture pruning utilities for TestSmith.
"""

from pathlib import Path
import re

from testsmith.support.config import TestSmithConfig
from testsmith.core.source_analyzer import analyze_file
from testsmith.core.project_detector import build_project_context


def scan_used_dependencies(project_root: Path, config: TestSmithConfig) -> set[str]:
    """
    Scan all source files to identify actively used external dependencies.

    Returns:
        Set of root package names that are imported somewhere in the project.
    """
    used_deps = set()

    # Build project context
    try:
        project_context = build_project_context(project_root, config)
    except Exception:
        return used_deps

    # Discover all Python source files
    for path in project_root.rglob("*.py"):
        # Skip test files and excluded dirs
        if any(excluded in path.parts for excluded in config.exclude_dirs):
            continue
        if path.name.startswith("test_") or path.name == "__init__.py":
            continue
        test_root_name = config.test_root.rstrip("/").split("/")[-1]
        if test_root_name in path.parts:
            continue

        try:
            # Analyze file to get imports
            analysis = analyze_file(path, project_context)

            # Extract external dependency root packages
            for imp in analysis.imports.external:
                root_pkg = imp.module.split(".")[0]
                used_deps.add(root_pkg)
        except Exception:
            # Skip files that fail to analyze
            continue

    return used_deps


def scan_existing_fixtures(
    fixture_dir: Path, config: TestSmithConfig
) -> dict[str, Path]:
    """
    Scan fixture directory for all *.fixture.py files.

    Returns:
        Mapping of dependency name to fixture file path.
    """
    fixtures = {}

    if not fixture_dir.exists():
        return fixtures

    for fixture_file in fixture_dir.glob("*.fixture.py"):
        # Extract dependency name from filename
        # e.g., "stripe.fixture.py" -> "stripe"
        dep_name = fixture_file.stem.replace(".fixture", "")
        fixtures[dep_name] = fixture_file

    return fixtures


def identify_unused_fixtures(
    used: set[str], existing: dict[str, Path]
) -> list[tuple[str, Path]]:
    """
    Identify fixtures that exist but are not used.

    Returns:
        List of (dependency_name, fixture_path) tuples for unused fixtures.
    """
    unused = []

    for dep_name, fixture_path in existing.items():
        if dep_name not in used:
            unused.append((dep_name, fixture_path))

    return unused


def prune_fixtures(
    unused: list[tuple[str, Path]], dry_run: bool = True
) -> list[tuple[str, str]]:
    """
    Delete unused fixture files.

    Args:
        unused: List of (dependency_name, fixture_path) tuples
        dry_run: If True, don't actually delete files

    Returns:
        List of (fixture_name, action) tuples where action is "would_delete" or "deleted"
    """
    results = []

    for dep_name, fixture_path in unused:
        if dry_run:
            results.append((dep_name, "would_delete"))
        else:
            try:
                fixture_path.unlink()
                results.append((dep_name, "deleted"))
            except Exception as e:
                results.append((dep_name, f"error: {e}"))

    # Update conftest.py if not dry run
    if not dry_run and unused:
        _update_fixture_conftest(unused)

    return results


def _update_fixture_conftest(deleted: list[tuple[str, Path]]) -> None:
    """
    Update tests/fixtures/conftest.py to remove re-exports of deleted fixtures.
    """
    if not deleted:
        return

    # Get conftest path from first deleted fixture
    fixture_dir = deleted[0][1].parent
    conftest_path = fixture_dir / "conftest.py"

    if not conftest_path.exists():
        return

    # Read conftest
    content = conftest_path.read_text(encoding="utf-8")
    lines = content.split("\n")

    # Build set of deleted fixture names
    deleted_names = {dep_name for dep_name, _ in deleted}

    # Filter out lines that import deleted fixtures
    new_lines = []
    for line in lines:
        # Check if line imports a deleted fixture
        skip = False
        for dep_name in deleted_names:
            if (
                f"from .{dep_name}.fixture import" in line
                or f"from .{dep_name}_fixture import" in line
            ):
                skip = True
                break

        if not skip:
            new_lines.append(line)

    # Write back
    conftest_path.write_text("\n".join(new_lines), encoding="utf-8")


def update_test_imports(project_root: Path, deleted_fixtures: list[str]) -> list[Path]:
    """
    Comment out imports of deleted fixtures in test files.

    Args:
        project_root: Project root directory
        deleted_fixtures: List of deleted fixture names

    Returns:
        List of modified test file paths
    """
    modified = []

    if not deleted_fixtures:
        return modified

    # Find all test files
    test_files = list(project_root.rglob("test_*.py"))

    for test_file in test_files:
        try:
            content = test_file.read_text(encoding="utf-8")
            lines = content.split("\n")
            modified_lines = False

            new_lines = []
            for line in lines:
                # Check if line imports a deleted fixture
                commented = False
                for fixture_name in deleted_fixtures:
                    # Match various import patterns
                    # Look for fixture_name or fixture_name_fixture in imports
                    patterns = [
                        rf"\b{fixture_name}\b",  # exact match
                        rf"\b{fixture_name}_fixture\b",  # with _fixture suffix
                        rf"\.{fixture_name}\.fixture\b",  # .fixture module
                    ]

                    for pattern in patterns:
                        if re.search(pattern, line) and (
                            "import" in line or "from" in line
                        ):
                            new_lines.append(
                                f"# PRUNED by TestSmith: fixture no longer needed - {line}"
                            )
                            modified_lines = True
                            commented = True
                            break

                    if commented:
                        break

                if not commented:
                    new_lines.append(line)

            if modified_lines:
                test_file.write_text("\n".join(new_lines), encoding="utf-8")
                modified.append(test_file)

        except Exception:
            # Skip files that fail to process
            continue

    return modified
