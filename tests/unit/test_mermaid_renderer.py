"""
Unit tests for mermaid_renderer module.
"""
import pytest
from pathlib import Path
from testsmith.visualization.mermaid_renderer import render_mermaid, render_metrics_table
from testsmith.support.models import (
    DependencyGraph, GraphNode, GraphEdge, ModuleMetrics
)


def test_render_mermaid():
    """Test Mermaid diagram generation."""
    nodes = [
        GraphNode("myapp.api", Path("/src/api.py"), "myapp", 2),
        GraphNode("myapp.service", Path("/src/service.py"), "myapp", 0),
        GraphNode("utils.helper", Path("/src/utils/helper.py"), "utils", 1),
    ]
    
    edges = [
        GraphEdge("myapp.api", "requests", "external"),
        GraphEdge("myapp.api", "stripe", "external"),
        GraphEdge("myapp.service", "myapp.api", "internal"),
        GraphEdge("utils.helper", "json", "external"),
    ]
    
    graph = DependencyGraph(nodes=nodes, edges=edges)
    
    metrics = {
        "myapp.api": ModuleMetrics("myapp.api", 0, 2, 1, 4.0),
        "myapp.service": ModuleMetrics("myapp.service", 1, 0, 0, 0.5),
        "utils.helper": ModuleMetrics("utils.helper", 0, 1, 0, 2.0),
    }
    
    result = render_mermaid(graph, metrics)
    
    # Check basic structure
    assert "```mermaid" in result
    assert "graph TD" in result
    assert "```" in result.split("```mermaid")[1]
    
    # Check subgraphs
    assert "subgraph myapp" in result
    assert "subgraph utils" in result
    assert "subgraph External" in result
    
    # Check nodes
    assert "myapp_api" in result
    assert "myapp_service" in result
    assert "utils_helper" in result
    
    # Check external nodes
    assert "requests" in result
    assert "stripe" in result
    
    # Check edges (solid for internal, dashed for external)
    assert "-->" in result  # internal edge
    assert "-.->" in result  # external edge
    
    # Check styles
    assert "classDef lowCoupling" in result
    assert "classDef mediumCoupling" in result
    assert "classDef highCoupling" in result
    assert "classDef external" in result


def test_render_metrics_table():
    """Test metrics table generation."""
    metrics = {
        "myapp.api": ModuleMetrics("myapp.api", 0, 2, 1, 4.0),
        "myapp.service": ModuleMetrics("myapp.service", 1, 0, 0, 0.5),
        "utils.helper": ModuleMetrics("utils.helper", 0, 1, 0, 2.0),
    }
    
    result = render_metrics_table(metrics)
    
    # Check structure
    assert "## Module Coupling Metrics" in result
    assert "| Module |" in result
    assert "|--------|" in result
    
    # Check data rows (should be sorted by coupling score descending)
    lines = result.split("\n")
    data_lines = [l for l in lines if l.startswith("| myapp") or l.startswith("| utils")]
    
    # First should be highest coupling (myapp.api with 4.0)
    assert "myapp.api" in data_lines[0]
    assert "4.0" in data_lines[0]
    
    # Check legend
    assert "Legend" in result
    assert "Coupling Score" in result


def test_render_mermaid_empty_graph():
    """Test with empty graph."""
    graph = DependencyGraph(nodes=[], edges=[])
    metrics = {}
    
    result = render_mermaid(graph, metrics)
    
    assert "```mermaid" in result
    assert "graph TD" in result
    # Should still have valid structure even if empty


def test_render_metrics_table_empty():
    """Test with no metrics."""
    result = render_metrics_table({})
    
    assert "## Module Coupling Metrics" in result
    assert "| Module |" in result
    # Table should still have headers
