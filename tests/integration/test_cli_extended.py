"""
Extended integration tests for TestSmith CLI covering Watch, Prune, Coverage, and Graph modes.
"""

import pytest
from unittest.mock import patch, MagicMock
from pathlib import Path
from testsmith.cli import main


@pytest.fixture
def mock_project_context(tmp_path):
    """Create a mock project context."""
    project_root = tmp_path / "mock_project"
    project_root.mkdir()
    (project_root / "pyproject.toml").write_text(
        '[tool.testsmith]\ntest_root = "tests"', encoding="utf-8"
    )
    (project_root / "src").mkdir()
    (project_root / "tests").mkdir()
    (project_root / "tests" / "fixtures").mkdir()

    with patch("testsmith.cli.build_project_context") as mock_ctx_builder:
        mock_ctx = MagicMock()
        mock_ctx.root = project_root
        mock_ctx.package_map = {"src": "src"}
        mock_ctx_builder.return_value = mock_ctx
        yield project_root, mock_ctx


def test_cli_watch_mode(mock_project_context):
    """Test --watch argument triggers watch_project."""
    project_root, _ = mock_project_context

    with (
        patch("sys.argv", ["testsmith", "--watch"]),
        patch("pathlib.Path.cwd", return_value=project_root),
        patch("testsmith.cli.watch_project") as mock_watch,
    ):

        # Simulate KeyboardInterrupt to exit the watch loop cleanly if implemented that way
        # or just verify it's called if it blocks.
        # The CLI catches KeyboardInterrupt and returns 0.
        mock_watch.side_effect = KeyboardInterrupt()

        try:
            main()
        except SystemExit as e:
            assert e.code == 0

        mock_watch.assert_called_once()
        # Verify the callback passed to watch_project
        args, _ = mock_watch.call_args
        assert args[0] == project_root
        assert callable(args[2])  # process_changed_file callback


def test_cli_watch_mode_process_callback(mock_project_context):
    """Test the callback function defined inside run() for watch mode."""
    project_root, mock_ctx = mock_project_context

    with (
        patch("sys.argv", ["testsmith", "--watch"]),
        patch("pathlib.Path.cwd", return_value=project_root),
        patch("testsmith.cli.watch_project") as mock_watch,
        patch("testsmith.cli.process_file") as mock_process,
    ):

        # Capture the callback
        mock_watch.side_effect = KeyboardInterrupt()
        try:
            main()
        except SystemExit:
            pass

        callback = mock_watch.call_args[0][2]

        # Test callback success
        mock_process.return_value = {"status": "success"}
        assert callback(Path("some_file.py")) is True

        # Test callback failure
        mock_process.return_value = {"status": "error"}
        assert callback(Path("some_file.py")) is False


def test_cli_prune_mode_dry_run(mock_project_context, capsys):
    """Test --prune in dry-run mode (default)."""
    project_root, _ = mock_project_context

    with (
        patch("sys.argv", ["testsmith", "--prune"]),
        patch("pathlib.Path.cwd", return_value=project_root),
        patch("testsmith.cli.scan_used_dependencies"),
        patch("testsmith.cli.scan_existing_fixtures"),
        patch("testsmith.cli.identify_unused_fixtures") as mock_identify,
        patch("testsmith.cli.prune_fixtures") as mock_prune,
    ):

        mock_identify.return_value = [
            ("foo", project_root / "tests/fixtures/foo.fixture.py")
        ]
        # prune_fixtures returns list of (name, action)
        mock_prune.return_value = [("foo", "skipped")]

        try:
            main()
        except SystemExit as e:
            assert e.code == 0

        mock_prune.assert_called_with(
            [("foo", project_root / "tests/fixtures/foo.fixture.py")], dry_run=True
        )

        captured = capsys.readouterr()
        assert "Unused fixtures found: 1" in captured.out
        assert "Run with --prune --confirm" in captured.out


def test_cli_prune_mode_confirm(mock_project_context, capsys):
    """Test --prune --confirm execution."""
    project_root, _ = mock_project_context

    with (
        patch("sys.argv", ["testsmith", "--prune", "--confirm"]),
        patch("pathlib.Path.cwd", return_value=project_root),
        patch("testsmith.cli.scan_used_dependencies"),
        patch("testsmith.cli.scan_existing_fixtures"),
        patch("testsmith.cli.identify_unused_fixtures") as mock_identify,
        patch("testsmith.cli.prune_fixtures") as mock_prune,
        patch("testsmith.cli.update_test_imports") as mock_update,
    ):

        mock_identify.return_value = [
            ("foo", project_root / "tests/fixtures/foo.fixture.py")
        ]
        mock_prune.return_value = [("foo", "deleted")]
        mock_update.return_value = ["test_foo.py"]

        try:
            main()
        except SystemExit as e:
            assert e.code == 0

        mock_prune.assert_called_with(mock_identify.return_value, dry_run=False)
        mock_update.assert_called_once()

        captured = capsys.readouterr()
        assert "Deleted fixtures:" in captured.out
        assert "Updated 1 test file(s)" in captured.out


def test_cli_prune_no_unused(mock_project_context, capsys):
    """Test --prune when no unused fixtures found."""
    project_root, _ = mock_project_context

    with (
        patch("sys.argv", ["testsmith", "--prune"]),
        patch("pathlib.Path.cwd", return_value=project_root),
        patch("testsmith.cli.scan_used_dependencies"),
        patch("testsmith.cli.scan_existing_fixtures"),
        patch("testsmith.cli.identify_unused_fixtures") as mock_identify,
    ):

        mock_identify.return_value = []

        try:
            main()
        except SystemExit as e:
            assert e.code == 0

        captured = capsys.readouterr()
        assert "No unused fixtures found" in captured.out


def test_cli_coverage_gaps(mock_project_context, capsys):
    """Test --coverage-gaps."""
    project_root, _ = mock_project_context

    with (
        patch("sys.argv", ["testsmith", "--coverage-gaps"]),
        patch("pathlib.Path.cwd", return_value=project_root),
        patch("testsmith.cli.detect_test_coverage"),
        patch("testsmith.cli.build_dependency_graph"),
        patch("testsmith.cli.compute_metrics"),
        patch("testsmith.cli.prioritize_gaps") as mock_gaps,
        patch("testsmith.cli.generate_report") as mock_report,
    ):

        mock_gaps.return_value = ["some_gap"]
        mock_report.return_value = "Coverage Report Content"

        try:
            main()
        except SystemExit as e:
            assert e.code == 0

        assert (Path("testsmith_coverage_report.md")).exists()
        Path("testsmith_coverage_report.md").unlink()

        captured = capsys.readouterr()
        assert "Analyzing test coverage..." in captured.out
        assert "Coverage Report Content" in captured.out
        assert "Found 1 coverage gap(s)" in captured.out


def test_cli_graph_generation(mock_project_context, capsys):
    """Test --graph."""
    project_root, _ = mock_project_context

    with (
        patch("sys.argv", ["testsmith", "--graph"]),
        patch("pathlib.Path.cwd", return_value=project_root),
        patch("testsmith.cli.build_dependency_graph"),
        patch("testsmith.cli.compute_metrics") as mock_metrics,
        patch("testsmith.cli.render_mermaid") as mock_mermaid,
        patch("testsmith.cli.render_metrics_table") as mock_table,
    ):

        mock_metrics.return_value = {}
        mock_mermaid.return_value = "graph TD;"
        mock_table.return_value = "| Metrics |"

        try:
            main()
        except SystemExit as e:
            assert e.code == 0

        assert (Path("testsmith_graph.md")).exists()
        Path("testsmith_graph.md").unlink()

        captured = capsys.readouterr()
        assert "Building dependency graph..." in captured.out
        assert "Dependency graph written to" in captured.out


def test_cli_graph_error(mock_project_context, capsys):
    """Test --graph error handling."""
    project_root, _ = mock_project_context

    with (
        patch("sys.argv", ["testsmith", "--graph"]),
        patch("pathlib.Path.cwd", return_value=project_root),
        patch("testsmith.cli.build_dependency_graph") as mock_build,
    ):

        mock_build.side_effect = Exception("Graph Error")

        try:
            main()
        except SystemExit as e:
            assert e.code == 1

        captured = capsys.readouterr()
        assert "Error generating graph: Graph Error" in captured.out


def test_cli_watch_error(mock_project_context, capsys):
    """Test --watch error handling."""
    project_root, _ = mock_project_context

    with (
        patch("sys.argv", ["testsmith", "--watch"]),
        patch("pathlib.Path.cwd", return_value=project_root),
        patch("testsmith.cli.watch_project") as mock_watch,
    ):

        mock_watch.side_effect = Exception("Watch Error")

        try:
            main()
        except SystemExit as e:
            assert e.code == 1

        captured = capsys.readouterr()
        assert "Error in watch mode: Watch Error" in captured.out


def test_cli_prune_error(mock_project_context, capsys):
    """Test --prune error handling."""
    project_root, _ = mock_project_context

    with (
        patch("sys.argv", ["testsmith", "--prune"]),
        patch("pathlib.Path.cwd", return_value=project_root),
        patch("testsmith.cli.scan_used_dependencies") as mock_scan,
    ):

        mock_scan.side_effect = Exception("Prune Error")

        try:
            main()
        except SystemExit as e:
            assert e.code == 1

        captured = capsys.readouterr()
        assert "Error during pruning: Prune Error" in captured.out


def test_cli_coverage_error(mock_project_context, capsys):
    """Test --coverage-gaps error handling."""
    project_root, _ = mock_project_context

    with (
        patch("sys.argv", ["testsmith", "--coverage-gaps"]),
        patch("pathlib.Path.cwd", return_value=project_root),
        patch("testsmith.cli.detect_test_coverage") as mock_cov,
    ):

        mock_cov.side_effect = Exception("Coverage Error")

        try:
            main()
        except SystemExit as e:
            assert e.code == 1

        captured = capsys.readouterr()
        assert "Error during coverage analysis: Coverage Error" in captured.out
