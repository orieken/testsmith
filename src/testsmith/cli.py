"""
TestSmith CLI Entry Point.
"""

import argparse
import sys
from pathlib import Path

from testsmith.support.config import load_config
from testsmith.support.exceptions import ProjectRootNotFoundError
from testsmith.core.project_detector import build_project_context
from testsmith.core.source_analyzer import analyze_file
from testsmith.core.discovery import discover_untested_files, discover_files_in_path
from testsmith.generation.conftest_updater import (
    update_conftest,
    compute_required_paths,
)
from testsmith.generation.llm_generator import generate_test_bodies
from testsmith.generation.test_generator import generate_test
from testsmith.generation.fixture_generator import generate_or_update_fixture
from testsmith.visualization.graph_builder import (
    build_dependency_graph,
    compute_metrics,
)
from testsmith.visualization.mermaid_renderer import (
    render_mermaid,
    render_metrics_table,
)
from testsmith.maintenance.fixture_pruner import (
    scan_used_dependencies,
    scan_existing_fixtures,
    identify_unused_fixtures,
    prune_fixtures,
    update_test_imports,
)
from testsmith.maintenance.coverage_analyzer import (
    detect_test_coverage,
    prioritize_gaps,
    generate_report,
)
from testsmith.watch import watch_project
from testsmith import __version__


def parse_args(argv: list[str] | None = None) -> argparse.Namespace:
    """Parse command line arguments."""
    parser = argparse.ArgumentParser(
        description="TestSmith: Automatic test generator for Python."
    )

    parser.add_argument(
        "source_file",
        nargs="?",
        help="Path to the source file to generate tests for. Required unless --all or --path is used.",
    )

    parser.add_argument("--config", help="Path to pyproject.toml configuration file.")

    parser.add_argument(
        "--init",
        action="store_true",
        help="Initialize TestSmith configuration and directories only.",
    )

    parser.add_argument(
        "--version", action="version", version=f"testsmith {__version__}"
    )

    parser.add_argument(
        "--dry-run", action="store_true", help="Simulate actions without writing files."
    )

    parser.add_argument(
        "--verbose", "-v", action="store_true", help="Enable verbose output."
    )

    parser.add_argument(
        "--all",
        action="store_true",
        help="Process all untested source files in the project.",
    )

    parser.add_argument(
        "--path", help="Process all untested source files in a specific directory."
    )

    parser.add_argument(
        "--generate-bodies",
        action="store_true",
        help="Use LLM to generate test bodies (requires ANTHROPIC_API_KEY).",
    )

    parser.add_argument(
        "--graph", action="store_true", help="Generate dependency graph visualization."
    )

    parser.add_argument(
        "--graph-output",
        default="testsmith_graph.md",
        help="Output file for dependency graph (default: testsmith_graph.md).",
    )

    parser.add_argument(
        "--prune", action="store_true", help="Identify and remove unused fixture files."
    )

    parser.add_argument(
        "--confirm",
        action="store_true",
        help="Confirm deletion when using --prune (default is dry-run).",
    )

    parser.add_argument(
        "--coverage-gaps",
        action="store_true",
        help="Analyze and report test coverage gaps.",
    )

    parser.add_argument(
        "--coverage-output",
        default="testsmith_coverage_report.md",
        help="Output file for coverage report (default: testsmith_coverage_report.md).",
    )

    parser.add_argument(
        "--watch",
        action="store_true",
        help="Watch for file changes and automatically regenerate tests.",
    )

    return parser.parse_args(argv)


def process_file(source_path: Path, project_context, config, args) -> dict:
    """
    Process a single source file.
    Returns a dict with action results.
    """
    result = {
        "source": source_path,
        "fixtures": [],
        "test": "skipped",
        "test_path": None,
        "conftest": "skipped",
        "conftest_path": None,
        "error": None,
    }

    try:
        # Analyze Source
        if args.verbose:
            print(f"Analyzing {source_path}...")

        analysis = analyze_file(source_path, project_context)

        if args.verbose:
            print(f"Detected {len(analysis.public_api)} public members.")
            print(f"Detected {len(analysis.imports.external)} external dependencies.")

        # DRY RUN CHECK
        if args.dry_run:
            if args.verbose:
                print("[DRY RUN] Analysis complete. Skipping file generation.")
            result["test"] = "dry-run"
            # Calculate path for summary
            from testsmith.generation.test_generator import derive_test_path

            result["test_path"] = derive_test_path(
                source_path, project_context.root, config
            )

            # Also calculate fixtures?
            # Fixtures in loop below.
            # If we return here, we skip fixture "generation" (mocking loop).
            # So fixtures list is empty.
            # We should probably run the fixture loop in dry run too?
            # Prompt 10: "--dry-run works in batch mode too".
            # The current implementation skips loop.
            # So summary shows "Fixtures" section empty?
            # Prompt doesn't explicitly require dry run to show fixtures.
            # But it would be nice.
            # For now, let's just fix the test failure which checks for "Dry-run".
            return result

        # Generate Fixtures
        # Group external imports by module to avoid duplicate fixtures
        external_modules = {
            imp.module.split(".")[0] for imp in analysis.imports.external
        }

        for dep in external_modules:
            # Gather usage info for this dependency
            relevant_imports = [
                imp
                for imp in analysis.imports.external
                if imp.module == dep or imp.module.startswith(dep + ".")
            ]

            imported_names = {}
            found_submodules = {dep}

            for imp in relevant_imports:
                found_submodules.add(imp.module)
                if imp.module not in imported_names:
                    imported_names[imp.module] = []
                imported_names[imp.module].extend(imp.names)

            fix_path, action = generate_or_update_fixture(
                dep,
                sub_modules=sorted(list(found_submodules)),
                imported_names=imported_names,
                project_root=project_context.root,
                config=config,
            )
            if action != "skipped":
                result["fixtures"].append((fix_path, action, dep))
                if args.verbose:
                    print(f"{action.capitalize()} fixture for {dep}: {fix_path.name}")

        # Generate Test Bodies (LLM)
        test_bodies = None
        if args.generate_bodies and not args.dry_run:
            # Enable LLM in config if not already (CLI flag override)
            config.llm.enabled = True
            try:
                test_bodies = generate_test_bodies(analysis, config.llm)
            except Exception as e:
                # If LLM fails, we log but continue with stubs
                print(f"Warning: LLM generation failed: {e}. Falling back to stubs.")
                test_bodies = None

        # Generate Test
        test_path, test_action = generate_test(analysis, config, test_bodies)
        result["test"] = test_action
        result["test_path"] = test_path
        if args.verbose:
            print(f"{test_action.capitalize()} test file: {test_path}")

        # Update Conftest
        paths = compute_required_paths(
            source_path,
            test_path,
            project_context.root / config.fixture_dir,
            project_context.root,
        )

        conftest_path = project_context.root / config.conftest_path
        c_path, c_action = update_conftest(conftest_path, paths, config)
        result["conftest"] = c_action
        result["conftest_path"] = c_path

        if c_action != "skipped" and args.verbose:
            print(f"{c_action.capitalize()} conftest.py: {c_path}")

        return result

    except Exception as e:
        result["error"] = str(e)
        if args.verbose:
            print(f"Error processing {source_path}: {e}")
        return result


def print_result_summary(result: dict, project_root: Path):
    """Print summary for a single file run."""
    source_path = result["source"]
    print("\nTestSmith Summary")
    print("─────────────────")
    print(f"Source:   {source_path.relative_to(project_root)}")
    print(f"Project:  {project_root}")
    print("")

    if result["error"]:
        print(f"Status:   FAILED ({result['error']})")
        return

    print("Actions:")

    # Fixtures
    for fpath, fact, fdep in result["fixtures"]:
        symbol = "✓" if fact in ["created", "updated"] else "·"
        rel_fpath = fpath.relative_to(project_root)
        print(f"  {symbol} {fact.capitalize():<8} {rel_fpath}")

    # Test File
    # Test File
    test_action = result["test"]
    if test_action != "skipped" and result.get("test_path"):
        symbol = "✓"
        test_path = result["test_path"]
        rel_tpath = test_path.relative_to(project_root)
        print(f"  {symbol} {test_action.capitalize():<8} {rel_tpath}")

    # Conftest
    c_action = result["conftest"]
    if c_action != "skipped" and result.get("conftest_path"):
        symbol = "✓"
        rel_cpath = result["conftest_path"].relative_to(project_root)
        print(f"  {symbol} {c_action.capitalize():<8} {rel_cpath}")


def run(args: argparse.Namespace) -> int:
    """Main command logic.
    Returns exit code (0 for success, non-zero for error).
    """

    # Handle watch mode
    if args.watch:
        try:
            config = load_config(Path(args.config) if args.config else None)
            project_context = build_project_context(Path.cwd(), config)

            # Define file processor for watch mode
            def process_changed_file(file_path: Path):
                """Process a single changed file."""
                result = process_file(file_path, project_context, config, args)
                if result.get("status") == "success":
                    return True
                return False

            # Start watching
            watch_project(project_context.root, config, process_changed_file)
            return 0
        except KeyboardInterrupt:
            return 0
        except Exception as e:
            print(f"Error in watch mode: {e}")
            return 1

    # Handle fixture pruning
    if args.prune:
        try:
            config = load_config(Path(args.config) if args.config else None)
            project_context = build_project_context(Path.cwd(), config)

            print("Scanning project for used dependencies...")
            used_deps = scan_used_dependencies(project_context.root, config)

            fixture_dir = project_context.root / config.test_root / "fixtures"
            existing_fixtures = scan_existing_fixtures(fixture_dir, config)

            unused = identify_unused_fixtures(used_deps, existing_fixtures)

            if not unused:
                print("\n✓ No unused fixtures found. All fixtures are actively used.")
                return 0

            # Print summary
            print("\nTestSmith Prune Summary")
            print("───────────────────────")
            print(f"Unused fixtures found: {len(unused)}\n")

            for dep_name, fixture_path in unused:
                rel_path = fixture_path.relative_to(project_context.root)
                print(f"  ✗ {rel_path} — no source files import {dep_name}")

            # Perform pruning
            dry_run = not args.confirm
            results = prune_fixtures(unused, dry_run=dry_run)

            if dry_run:
                print("\nRun with --prune --confirm to delete these fixtures.")
            else:
                print("\nDeleted fixtures:")
                deleted_names = [
                    name for name, action in results if action == "deleted"
                ]
                for name in deleted_names:
                    print(f"  ✓ {name}.fixture.py")

                # Update test imports
                if deleted_names:
                    modified_tests = update_test_imports(
                        project_context.root, deleted_names
                    )
                    if modified_tests:
                        print(
                            f"\nUpdated {len(modified_tests)} test file(s) to comment out deleted fixture imports."
                        )

            return 0
        except Exception as e:
            print(f"Error during pruning: {e}")
            return 1

    # Handle coverage gap analysis
    if args.coverage_gaps:
        try:
            config = load_config(Path(args.config) if args.config else None)
            project_context = build_project_context(Path.cwd(), config)

            print("Analyzing test coverage...")
            test_root_path = project_context.root / config.test_root
            coverage = detect_test_coverage(
                project_context.root, test_root_path, config
            )

            # Build dependency graph for metrics
            print("Computing dependency metrics...")
            graph = build_dependency_graph(project_context.root, config)
            metrics = compute_metrics(graph)

            # Prioritize gaps
            gaps = prioritize_gaps(coverage, metrics)

            # Generate report
            report = generate_report(gaps, coverage)

            # Write to file
            output_path = Path(args.coverage_output)
            output_path.write_text(report, encoding="utf-8")
            print(f"\nCoverage report written to: {output_path}")

            # Print summary to stdout
            lines = report.split("\n")
            summary_end = lines.index("") if "" in lines[5:15] else 15
            print("\n" + "\n".join(lines[:summary_end]))

            if gaps:
                print(
                    f"\nFound {len(gaps)} coverage gap(s). See {output_path} for details."
                )

            return 0
        except Exception as e:
            print(f"Error during coverage analysis: {e}")
            import traceback

            traceback.print_exc()
            return 1

    # Handle graph visualization
    if args.graph:
        try:
            # Load config and project context for graph generation
            # Note: For graph generation, we might not have a specific source file,
            # so we default to current working directory for context building.
            config = load_config(Path(args.config) if args.config else None)
            project_context = build_project_context(Path.cwd(), config)

            print("Building dependency graph...")
            graph = build_dependency_graph(project_context.root, config)
            metrics = compute_metrics(graph)

            # Render outputs
            mermaid_diagram = render_mermaid(graph, metrics)
            metrics_table = render_metrics_table(metrics)

            # Write to file
            output_path = Path(args.graph_output)
            output_content = f"# TestSmith Dependency Graph\n\n{metrics_table}\n\n{mermaid_diagram}\n"
            output_path.write_text(output_content, encoding="utf-8")

            print(f"\nDependency graph written to: {output_path}")
            print("\n" + metrics_table)

            return 0
        except Exception as e:
            print(f"Error generating graph: {e}")
            return 1

    # Original logic for test generation

    try:
        # Load config
        config_path = Path(args.config) if args.config else None
        config = load_config(config_path)

        # Detect Project Context
        start_path = Path(args.source_file) if args.source_file else Path.cwd()
        if args.path:
            start_path = Path(args.path)

        try:
            project_context = build_project_context(start_path, config)
            if args.verbose:
                print(f"Project Root: {project_context.root}")
                print(f"Package Map: {project_context.package_map}")
        except ProjectRootNotFoundError:
            print(
                "Error: Could not detect project root. Ensure a pyproject.toml exists."
            )
            return 1

        if args.init:
            print(
                "Initialization not fully implemented in V1. Use 'testsmith <file>' to generate."
            )
            return 0

        # Determine files to process
        files_to_process = []

        if args.all:
            if args.verbose:
                print("Discovering all untested files...")
            files_to_process = discover_untested_files(
                project_context.root, project_context.root / config.test_root, config
            )
        elif args.path:
            target_path = Path(args.path).resolve()
            if not target_path.exists():
                print(f"Error: Path not found: {target_path}")
                return 1
            if args.verbose:
                print(f"Discovering files in {target_path}...")
            files_to_process = discover_files_in_path(
                target_path,
                project_context.root,
                project_context.root / config.test_root,
                config,
            )
        elif args.source_file:
            source_path = Path(args.source_file).resolve()
            if not source_path.exists():
                print(f"Error: Source file not found: {source_path}")
                return 1
            files_to_process = [source_path]
        else:
            print("Error: You must provide a source file, --all, or --path.")
            return 1

        if not files_to_process:
            print("No files found to process.")
            return 0

        if args.verbose:
            print(f"Found {len(files_to_process)} files to process.")

        # Process files
        results = []
        for src in files_to_process:
            if len(files_to_process) > 1:
                print(f"Processing {src.relative_to(project_context.root)}...")

            res = process_file(src, project_context, config, args)
            results.append(res)

            if res["error"] and len(files_to_process) == 1:
                # Single file mode error behavior
                print(f"Error: {res['error']}")
                return 1

        # Summary
        if len(files_to_process) == 1 and not (args.all or args.path):
            # Legacy single file summary
            res = results[0]
            print_result_summary(res, project_context.root)

        else:
            # Batch Summary
            print("\nTestSmith Batch Summary")
            print("───────────────────────")
            print(f"Processed {len(results)} source files")

            created_tests = len([r for r in results if r["test"] == "created"])
            skipped_tests = len([r for r in results if r["test"] == "skipped"])
            errors = len([r for r in results if r["error"]])

            print(f"Created:  {created_tests} test files")
            print(f"Skipped:  {skipped_tests} test files (already exist)")
            if errors > 0:
                print(f"Errors:   {errors} files failed")

            # Detailed error list
            if errors > 0:
                print("\nFailures:")
                for r in results:
                    if r["error"]:
                        print(f"  ✗ {r['source'].name}: {r['error']}")

        if any(r["error"] for r in results):
            return 1

        return 0

    except Exception as e:
        print(f"Unexpected error: {e}")
        import traceback

        traceback.print_exc()
        return 1


def main():
    """Entry point for console script."""
    sys.exit(run(parse_args()))


if __name__ == "__main__":
    main()
