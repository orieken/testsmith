"""
Mermaid diagram renderer for dependency graphs.
"""

from testsmith.support.models import DependencyGraph, ModuleMetrics


def render_mermaid(graph: DependencyGraph, metrics: dict[str, ModuleMetrics]) -> str:
    """
    Render a Mermaid diagram from the dependency graph.

    Returns:
        Mermaid markdown string.
    """
    lines = []
    lines.append("```mermaid")
    lines.append("graph TD")
    lines.append("")

    # Group nodes by package
    packages = {}
    external_nodes = set()

    for node in graph.nodes:
        if node.package not in packages:
            packages[node.package] = []
        packages[node.package].append(node)

    # Collect external dependencies
    for edge in graph.edges:
        if edge.edge_type == "external":
            external_nodes.add(edge.target)

    # Render internal packages as subgraphs
    for package, nodes in sorted(packages.items()):
        lines.append(f"    subgraph {package}")
        for node in nodes:
            # Color code by coupling
            metric = metrics.get(node.name)
            if metric:
                if metric.coupling_score < 2:
                    style = ":::lowCoupling"
                elif metric.coupling_score < 5:
                    style = ":::mediumCoupling"
                else:
                    style = ":::highCoupling"
            else:
                style = ""

            # Sanitize node name for Mermaid
            node_id = node.name.replace(".", "_")
            lines.append(f"        {node_id}[{node.name}]{style}")
        lines.append("    end")
        lines.append("")

    # Render external dependencies
    if external_nodes:
        lines.append("    subgraph External")
        for ext in sorted(external_nodes):
            ext_id = ext.replace(".", "_").replace("-", "_")
            lines.append(f"        {ext_id}[{ext}]:::external")
        lines.append("    end")
        lines.append("")

    # Render edges
    for edge in graph.edges:
        source_id = edge.source.replace(".", "_")
        target_id = edge.target.replace(".", "_").replace("-", "_")

        if edge.edge_type == "internal":
            lines.append(f"    {source_id} --> {target_id}")
        else:
            lines.append(f"    {source_id} -.-> {target_id}")

    # Add styles
    lines.append("")
    lines.append("    classDef lowCoupling fill:#90EE90,stroke:#333,stroke-width:2px")
    lines.append(
        "    classDef mediumCoupling fill:#FFD700,stroke:#333,stroke-width:2px"
    )
    lines.append("    classDef highCoupling fill:#FF6347,stroke:#333,stroke-width:2px")
    lines.append("    classDef external fill:#87CEEB,stroke:#333,stroke-width:2px")

    lines.append("```")
    return "\n".join(lines)


def render_metrics_table(metrics: dict[str, ModuleMetrics]) -> str:
    """
    Render a markdown table of module metrics.

    Returns:
        Markdown table string.
    """
    lines = []
    lines.append("## Module Coupling Metrics")
    lines.append("")
    lines.append(
        "| Module | Internal Deps | External Deps | Dependents | Coupling Score |"
    )
    lines.append(
        "|--------|---------------|---------------|------------|----------------|"
    )

    # Sort by coupling score descending
    sorted_metrics = sorted(
        metrics.values(), key=lambda m: m.coupling_score, reverse=True
    )

    for metric in sorted_metrics:
        lines.append(
            f"| {metric.name} | {metric.internal_dependencies} | "
            f"{metric.external_dependencies} | {metric.dependents} | "
            f"{metric.coupling_score:.1f} |"
        )

    lines.append("")
    lines.append("**Legend:**")
    lines.append("- **Coupling Score** = (External Deps × 2) + (Internal Deps × 0.5)")
    lines.append(
        "- Higher scores indicate modules that are harder to test in isolation"
    )
    lines.append("")

    return "\n".join(lines)
