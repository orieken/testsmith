"""
Unit tests for coverage_analyzer module.
"""

import pytest
from pathlib import Path
from testsmith.maintenance.coverage_analyzer import (
    detect_test_coverage,
    prioritize_gaps,
    generate_report,
)
from testsmith.support.config import TestSmithConfig as Config
from testsmith.support.models import ModuleMetrics


@pytest.fixture
def sample_project(tmp_path):
    """Create a project with various coverage states."""
    project_root = tmp_path / "project"
    project_root.mkdir()

    # Create pyproject.toml
    (project_root / "pyproject.toml").write_text("[tool.testsmith]\n", encoding="utf-8")

    # Create src structure
    src = project_root / "src" / "myapp"
    src.mkdir(parents=True)
    (src / "__init__.py").write_text("", encoding="utf-8")

    # Source file with no test
    (src / "no_test.py").write_text(
        """
def foo():
    pass
""",
        encoding="utf-8",
    )

    # Source file with skeleton test
    (src / "skeleton.py").write_text(
        """
def bar():
    pass
""",
        encoding="utf-8",
    )

    # Source file with partial test
    (src / "partial.py").write_text(
        """
def baz():
    pass
""",
        encoding="utf-8",
    )

    # Source file with full test
    (src / "covered.py").write_text(
        """
def qux():
    pass
""",
        encoding="utf-8",
    )

    # Create tests directory
    tests = project_root / "tests"
    tests.mkdir()

    # Skeleton test (only TODOs)
    (tests / "test_skeleton.py").write_text(
        """
def test_bar():
    # TODO: implement
    pass
""",
        encoding="utf-8",
    )

    # Partial test (mix of TODOs and assertions)
    (tests / "test_partial.py").write_text(
        """
def test_baz():
    assert True

def test_baz_other():
    # TODO: implement
    pass
""",
        encoding="utf-8",
    )

    # Full test (no TODOs)
    (tests / "test_covered.py").write_text(
        """
def test_qux():
    assert True
""",
        encoding="utf-8",
    )

    return project_root


def test_detect_test_coverage_no_test(sample_project):
    """Test detection of files with no test."""
    config = Config()
    test_root = sample_project / "tests"

    coverage = detect_test_coverage(sample_project, test_root, config)

    no_test_file = str(sample_project / "src" / "myapp" / "no_test.py")
    assert no_test_file in coverage
    assert coverage[no_test_file] == "no_test"


def test_detect_test_coverage_skeleton(sample_project):
    """Test detection of skeleton-only tests."""
    config = Config()
    test_root = sample_project / "tests"

    coverage = detect_test_coverage(sample_project, test_root, config)

    skeleton_file = str(sample_project / "src" / "myapp" / "skeleton.py")
    assert skeleton_file in coverage
    assert coverage[skeleton_file] == "skeleton_only"


def test_detect_test_coverage_partial(sample_project):
    """Test detection of partial coverage."""
    config = Config()
    test_root = sample_project / "tests"

    coverage = detect_test_coverage(sample_project, test_root, config)

    partial_file = str(sample_project / "src" / "myapp" / "partial.py")
    assert partial_file in coverage
    assert coverage[partial_file] == "partial"


def test_detect_test_coverage_covered(sample_project):
    """Test detection of fully covered files."""
    config = Config()
    test_root = sample_project / "tests"

    coverage = detect_test_coverage(sample_project, test_root, config)

    covered_file = str(sample_project / "src" / "myapp" / "covered.py")
    assert covered_file in coverage
    assert coverage[covered_file] == "covered"


def test_prioritize_gaps_scoring():
    """Test priority scoring formula."""
    coverage = {
        "/src/high_priority.py": "no_test",
        "/src/medium_priority.py": "skeleton_only",
        "/src/low_priority.py": "partial",
    }

    metrics = {
        "high_priority": ModuleMetrics("high_priority", 2, 5, 3, 10.0),
        "medium_priority": ModuleMetrics("medium_priority", 1, 2, 1, 5.0),
        "low_priority": ModuleMetrics("low_priority", 0, 1, 0, 2.0),
    }

    gaps = prioritize_gaps(coverage, metrics)

    assert len(gaps) == 3
    # High priority should be first (external_deps=5, dependents=3, status=no_test)
    # Score = 5*2 + 3*3 + 1.0 = 20.0
    assert gaps[0].source_path == Path("/src/high_priority.py")
    assert gaps[0].priority_score == 20.0


def test_prioritize_gaps_excludes_covered():
    """Test that fully covered files are excluded from gaps."""
    coverage = {
        "/src/uncovered.py": "no_test",
        "/src/covered.py": "covered",
    }

    metrics = {}

    gaps = prioritize_gaps(coverage, metrics)

    assert len(gaps) == 1
    assert gaps[0].source_path == Path("/src/uncovered.py")


def test_prioritize_gaps_suggested_commands():
    """Test suggested commands for different statuses."""
    coverage = {
        "/src/no_test.py": "no_test",
        "/src/skeleton.py": "skeleton_only",
    }

    metrics = {}

    gaps = prioritize_gaps(coverage, metrics)

    # No test should suggest basic testsmith command
    no_test_gap = [g for g in gaps if g.status == "no_test"][0]
    # Normalize path separators for cross-platform compatibility
    assert "/src/no_test.py" in no_test_gap.suggested_command.replace("\\", "/")
    assert "--generate-bodies" not in no_test_gap.suggested_command

    # Skeleton should suggest --generate-bodies
    skeleton_gap = [g for g in gaps if g.status == "skeleton_only"][0]
    assert "--generate-bodies" in skeleton_gap.suggested_command


def test_generate_report_summary():
    """Test report generation with summary stats."""
    coverage = {
        "/src/a.py": "no_test",
        "/src/b.py": "skeleton_only",
        "/src/c.py": "partial",
        "/src/d.py": "covered",
    }

    gaps = prioritize_gaps(coverage, {})
    report = generate_report(gaps, coverage)

    assert "# TestSmith Coverage Gap Analysis" in report
    assert "## Summary" in report
    assert "Total source files**: 4" in report
    assert "No test**: 1" in report
    assert "Skeleton only**: 1" in report
    assert "Partial coverage**: 1" in report
    assert "Fully covered**: 1" in report


def test_generate_report_no_gaps():
    """Test report when all files are covered."""
    coverage = {
        "/src/a.py": "covered",
        "/src/b.py": "covered",
    }

    gaps = prioritize_gaps(coverage, {})
    report = generate_report(gaps, coverage)

    assert "All source files have complete test coverage" in report


def test_generate_report_priority_table():
    """Test that report includes priority table."""
    coverage = {
        "/src/high.py": "no_test",
        "/src/low.py": "partial",
    }

    metrics = {
        "high": ModuleMetrics("high", 0, 5, 2, 10.0),
        "low": ModuleMetrics("low", 0, 0, 0, 0.0),
    }

    gaps = prioritize_gaps(coverage, metrics)
    report = generate_report(gaps, coverage)

    assert "## Priority Coverage Gaps" in report
    assert "| Priority | File | Status |" in report
    assert "high.py" in report
    assert "low.py" in report
