"""
Dependency graph builder for TestSmith.
"""

from pathlib import Path
from collections import defaultdict

from testsmith.support.config import TestSmithConfig
from testsmith.support.models import (
    DependencyGraph,
    GraphNode,
    GraphEdge,
    ModuleMetrics,
    ProjectContext,
)
from testsmith.core.source_analyzer import analyze_file


def build_dependency_graph(
    project_context: ProjectContext, config: TestSmithConfig
) -> DependencyGraph:
    """
    Build a dependency graph for the entire project.

    Scans all Python source files, analyzes imports, and constructs a directed graph.
    """
    project_root = project_context.root
    nodes = []
    edges = []

    # Discover all Python files
    python_files = []
    for path in project_root.rglob("*.py"):
        # Skip excluded dirs
        if any(excluded in path.parts for excluded in config.exclude_dirs):
            continue
        # Skip test files and __init__
        if path.name.startswith("test_") or path.name == "__init__.py":
            continue
        # Skip files in test directory
        test_root_name = config.test_root.rstrip("/").split("/")[-1]
        if test_root_name in path.parts:
            continue

        python_files.append(path)

    # Analyze each file
    module_data = {}
    for source_path in python_files:
        try:
            analysis = analyze_file(source_path, project_context)

            # Derive module name
            try:
                rel = source_path.relative_to(project_root)
            except ValueError:
                rel = Path(source_path.name)

            parts = list(rel.parts)
            parts[-1] = rel.stem
            if parts and parts[0] == "src":
                parts = parts[1:]
            module_name = ".".join(parts)

            # Derive package name (top-level)
            package = parts[0] if parts else "root"

            # Count external dependencies
            external_deps = set()
            for imp in analysis.imports.external:
                root_pkg = imp.module.split(".")[0]
                external_deps.add(root_pkg)

            external_count = len(external_deps)

            # Create node
            node = GraphNode(
                name=module_name,
                path=source_path,
                package=package,
                external_dep_count=external_count,
            )
            nodes.append(node)

            # Store for edge creation
            module_data[module_name] = {
                "internal": analysis.imports.internal,
                "external": analysis.imports.external,
            }

        except Exception as e:
            # Skip files that fail to analyze
            print(f"Warning: Failed to analyze {source_path}: {e}")
            continue

    # Build edges
    for source_module, imports in module_data.items():
        # Internal edges
        for imp in imports["internal"]:
            # Derive target module name from import
            target = imp.module
            edges.append(
                GraphEdge(source=source_module, target=target, edge_type="internal")
            )

        # External edges
        for imp in imports["external"]:
            root_pkg = imp.module.split(".")[0]
            edges.append(
                GraphEdge(source=source_module, target=root_pkg, edge_type="external")
            )

    return DependencyGraph(nodes=nodes, edges=edges)


def compute_metrics(graph: DependencyGraph) -> dict[str, ModuleMetrics]:
    """
    Compute metrics for each module in the graph.

    Returns:
        Mapping of module name to ModuleMetrics.
    """
    metrics = {}

    # Build lookup structures
    node_map = {node.name: node for node in graph.nodes}

    # Count dependencies and dependents
    internal_deps = defaultdict(int)
    external_deps = defaultdict(int)
    dependents = defaultdict(int)

    for edge in graph.edges:
        if edge.edge_type == "internal":
            internal_deps[edge.source] += 1
            if edge.target in node_map:
                dependents[edge.target] += 1
        else:  # external
            external_deps[edge.source] += 1

    # Calculate coupling score
    # Simple formula: external_deps * 2 + internal_deps
    for node in graph.nodes:
        int_deps = internal_deps[node.name]
        ext_deps = external_deps[node.name]
        deps_count = dependents[node.name]

        coupling = ext_deps * 2.0 + int_deps * 0.5

        metrics[node.name] = ModuleMetrics(
            name=node.name,
            internal_dependencies=int_deps,
            external_dependencies=ext_deps,
            dependents=deps_count,
            coupling_score=coupling,
        )

    return metrics
