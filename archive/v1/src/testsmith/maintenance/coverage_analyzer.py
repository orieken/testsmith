"""
Coverage gap analysis for TestSmith.
"""

from pathlib import Path
import re

from testsmith.support.config import TestSmithConfig
from testsmith.support.models import CoverageGap, ModuleMetrics
from testsmith.generation.test_generator import derive_test_path


def detect_test_coverage(
    project_root: Path, test_root: Path, config: TestSmithConfig
) -> dict[str, str]:
    """
    Detect test coverage status for each source file.

    Returns:
        Mapping of source file path (as string) to coverage status.
        Status values: "no_test", "skeleton_only", "partial", "covered"
    """
    coverage = {}

    # Find all source files
    for source_path in project_root.rglob("*.py"):
        # Skip test files and excluded dirs
        if any(excluded in source_path.parts for excluded in config.exclude_dirs):
            continue
        if source_path.name.startswith("test_") or source_path.name == "__init__.py":
            continue
        test_root_name = (
            test_root.name
            if isinstance(test_root, Path)
            else test_root.rstrip("/").split("/")[-1]
        )
        if test_root_name in source_path.parts:
            continue

        # Determine test file path
        try:
            # Try the standard mirrored structure first
            test_path = derive_test_path(source_path, project_root, config)

            # If that doesn't exist, try a flat structure (tests/test_filename.py)
            if not test_path.exists():
                flat_test_path = test_root / f"test_{source_path.stem}.py"
                if flat_test_path.exists():
                    test_path = flat_test_path
        except Exception:
            coverage[str(source_path)] = "no_test"
            continue

        # Check if test file exists
        if not test_path.exists():
            coverage[str(source_path)] = "no_test"
            continue

        # Analyze test file content
        try:
            test_content = test_path.read_text(encoding="utf-8")

            # Count TODO stubs and real assertions
            todo_count = len(re.findall(r"#\s*TODO", test_content, re.IGNORECASE))
            assertion_count = len(re.findall(r"\bassert\b", test_content))

            # Determine status
            if todo_count > 0 and assertion_count == 0:
                coverage[str(source_path)] = "skeleton_only"
            elif todo_count > 0 and assertion_count > 0:
                coverage[str(source_path)] = "partial"
            elif todo_count == 0 and assertion_count > 0:
                coverage[str(source_path)] = "covered"
            else:
                # No TODOs and no assertions - treat as skeleton
                coverage[str(source_path)] = "skeleton_only"
        except Exception:
            coverage[str(source_path)] = "no_test"

    return coverage


def prioritize_gaps(
    coverage: dict[str, str], metrics: dict[str, ModuleMetrics]
) -> list[CoverageGap]:
    """
    Prioritize coverage gaps based on coupling metrics.

    Args:
        coverage: Mapping of source path to coverage status
        metrics: Mapping of module name to metrics

    Returns:
        Sorted list of CoverageGap objects (highest priority first)
    """
    gaps = []

    for source_path_str, status in coverage.items():
        # Skip fully covered files
        if status == "covered":
            continue

        source_path = Path(source_path_str)

        # Try to find matching metrics
        # Module name might be derived differently, so try a few approaches
        module_name = source_path.stem
        metric = None

        # Try exact match first
        if module_name in metrics:
            metric = metrics[module_name]
        else:
            # Try to find by partial match
            for mod_name, mod_metric in metrics.items():
                if module_name in mod_name or mod_name.endswith(module_name):
                    metric = mod_metric
                    break

        # Calculate priority score
        if metric:
            external_deps = metric.external_dependencies
            dependents = metric.dependents
        else:
            external_deps = 0
            dependents = 0

        # Priority formula
        if status == "no_test":
            status_weight = 1.0
        elif status == "skeleton_only":
            status_weight = 0.5
        else:  # partial
            status_weight = 0.2

        priority_score = external_deps * 2 + dependents * 3 + status_weight

        # Suggested command
        if status == "no_test":
            suggested_command = f"testsmith {source_path}"
        else:
            suggested_command = f"testsmith --generate-bodies {source_path}"

        gap = CoverageGap(
            source_path=source_path,
            status=status,
            priority_score=priority_score,
            external_deps=external_deps,
            dependents=dependents,
            suggested_command=suggested_command,
        )
        gaps.append(gap)

    # Sort by priority descending
    gaps.sort(key=lambda g: g.priority_score, reverse=True)

    return gaps


def generate_report(gaps: list[CoverageGap], coverage: dict[str, str]) -> str:
    """
    Generate a markdown coverage gap report.

    Args:
        gaps: List of coverage gaps
        coverage: Full coverage mapping for summary stats

    Returns:
        Markdown report string
    """
    lines = []
    lines.append("# TestSmith Coverage Gap Analysis")
    lines.append("")

    # Summary stats
    total = len(coverage)
    no_test = sum(1 for s in coverage.values() if s == "no_test")
    skeleton = sum(1 for s in coverage.values() if s == "skeleton_only")
    partial = sum(1 for s in coverage.values() if s == "partial")
    covered = sum(1 for s in coverage.values() if s == "covered")

    lines.append("## Summary")
    lines.append("")
    lines.append(f"- **Total source files**: {total}")
    lines.append(
        f"- **No test**: {no_test} ({no_test/total*100:.1f}%)"
        if total > 0
        else "- **No test**: 0"
    )
    lines.append(
        f"- **Skeleton only**: {skeleton} ({skeleton/total*100:.1f}%)"
        if total > 0
        else "- **Skeleton only**: 0"
    )
    lines.append(
        f"- **Partial coverage**: {partial} ({partial/total*100:.1f}%)"
        if total > 0
        else "- **Partial coverage**: 0"
    )
    lines.append(
        f"- **Fully covered**: {covered} ({covered/total*100:.1f}%)"
        if total > 0
        else "- **Fully covered**: 0"
    )
    lines.append("")

    if not gaps:
        lines.append("✓ **All source files have complete test coverage!**")
        lines.append("")
        return "\n".join(lines)

    # Priority table
    lines.append("## Priority Coverage Gaps")
    lines.append("")
    lines.append(
        "Files are prioritized by coupling (external dependencies + dependents) and coverage status."
    )
    lines.append("")
    lines.append(
        "| Priority | File | Status | Ext Deps | Dependents | Suggested Command |"
    )
    lines.append(
        "|----------|------|--------|----------|------------|-------------------|"
    )

    for i, gap in enumerate(gaps[:20], 1):  # Top 20
        rel_path = gap.source_path.name  # Just filename for brevity
        status_emoji = (
            "❌"
            if gap.status == "no_test"
            else "⚠️" if gap.status == "skeleton_only" else "🔸"
        )
        lines.append(
            f"| {i} | {rel_path} | {status_emoji} {gap.status} | {gap.external_deps} | "
            f"{gap.dependents} | `{gap.suggested_command}` |"
        )

    if len(gaps) > 20:
        lines.append("")
        lines.append(f"*... and {len(gaps) - 20} more gaps*")

    lines.append("")
    lines.append("**Legend:**")
    lines.append("- ❌ `no_test`: No test file exists")
    lines.append("- ⚠️ `skeleton_only`: Test file has only TODO stubs")
    lines.append("- 🔸 `partial`: Test file has some real tests and some TODOs")
    lines.append("")

    return "\n".join(lines)
