"""
Unit tests for graph_builder module.
"""

import pytest
from testsmith.visualization.graph_builder import (
    build_dependency_graph,
    compute_metrics,
)
from testsmith.support.config import TestSmithConfig as Config
from testsmith.support.models import DependencyGraph, ProjectContext


@pytest.fixture
def sample_project_context(tmp_path):
    """Create a minimal project context for testing."""
    project_root = tmp_path / "project"
    project_root.mkdir()

    # Create pyproject.toml
    (project_root / "pyproject.toml").write_text("[tool.testsmith]\n", encoding="utf-8")

    # Create src structure
    src = project_root / "src" / "myapp"
    src.mkdir(parents=True)
    (src / "__init__.py").write_text("", encoding="utf-8")

    # Module with external deps
    (src / "api.py").write_text(
        """
import requests
from stripe import Charge

def fetch_data():
    pass
""",
        encoding="utf-8",
    )

    # Module with internal deps
    (src / "service.py").write_text(
        """
from myapp.api import fetch_data

def process():
    pass
""",
        encoding="utf-8",
    )

    # Isolated module
    (src / "utils.py").write_text(
        """
def helper():
    pass
""",
        encoding="utf-8",
    )

    # Construct a real or mock context
    # Usually we scan valid packages.
    # Package map for src/myapp
    package_map = {"myapp": src}

    return ProjectContext(
        root=project_root,
        package_map=package_map,
        conftest_path=None,
        existing_paths=[],
    )


def test_build_dependency_graph(sample_project_context):
    """Test graph construction."""
    config = Config()
    graph = build_dependency_graph(sample_project_context, config)

    assert isinstance(graph, DependencyGraph)

    # Now that we fixed the signature, we expect successful analysis
    assert len(graph.nodes) > 0, "Graph should have nodes"

    # Check for expected nodes
    node_names = [n.name for n in graph.nodes]
    assert "myapp.api" in node_names or "api" in node_names

    assert isinstance(graph.edges, list)


def test_compute_metrics(sample_project_context):
    """Test metrics calculation."""
    config = Config()
    graph = build_dependency_graph(sample_project_context, config)
    metrics = compute_metrics(graph)

    # Metrics should be a dict
    assert isinstance(metrics, dict)

    # If we have nodes, check metric structure
    for metric in metrics.values():
        assert hasattr(metric, "name")
        assert hasattr(metric, "coupling_score")
        assert metric.coupling_score >= 0


def test_build_dependency_graph_empty_project(tmp_path):
    """Test with empty project."""
    project_root = tmp_path / "empty"
    project_root.mkdir()
    (project_root / "pyproject.toml").write_text("", encoding="utf-8")

    config = Config()
    context = ProjectContext(
        root=project_root, package_map={}, conftest_path=None, existing_paths=[]
    )

    graph = build_dependency_graph(context, config)

    assert len(graph.nodes) == 0
    assert len(graph.edges) == 0
