"""
Data models for TestSmith.
"""
from dataclasses import dataclass, field
from pathlib import Path


@dataclass
class ProjectContext:
    """Detected project structure information."""
    root: Path
    package_map: dict[str, Path]
    conftest_path: Path | None
    existing_paths: list[str]


@dataclass
class ImportInfo:
    """A single import statement parsed from source."""
    module: str
    names: list[str]
    is_from: bool
    alias: str | None
    line_number: int


@dataclass
class ClassifiedImports:
    """Imports sorted by category."""
    stdlib: list[ImportInfo] = field(default_factory=list)
    internal: list[ImportInfo] = field(default_factory=list)
    external: list[ImportInfo] = field(default_factory=list)


@dataclass
class PublicMember:
    """A public class or function from the source module."""
    name: str
    kind: str  # "class" or "function"
    parameters: list[str]
    methods: list[str]
    docstring: str | None


@dataclass
class AnalysisResult:
    """Complete analysis output for a source file."""
    source_path: Path
    module_name: str
    imports: ClassifiedImports
    public_api: list[PublicMember]
    project: ProjectContext


@dataclass
class LLMConfig:
    """Configuration for LLM-based generation."""
    enabled: bool = False
    model: str = "claude-3-sonnet-20240229"
    max_tokens_per_function: int = 1500
    api_key_env_var: str = "ANTHROPIC_API_KEY"


@dataclass
class GraphNode:
    """Node in dependency graph."""
    name: str
    path: Path
    package: str
    external_dep_count: int


@dataclass
class GraphEdge:
    """Edge in dependency graph."""
    source: str
    target: str
    edge_type: str  # "internal" or "external"


@dataclass
class DependencyGraph:
    """Complete dependency graph."""
    nodes: list[GraphNode]
    edges: list[GraphEdge]


@dataclass
class ModuleMetrics:
    """Metrics for a single module."""
    name: str
    internal_dependencies: int
    external_dependencies: int
    dependents: int
    coupling_score: float


@dataclass
class CoverageGap:
    """Coverage gap information for a source file."""
    source_path: Path
    status: str  # "no_test", "skeleton_only", "partial"
    priority_score: float
    external_deps: int
    dependents: int
    suggested_command: str
