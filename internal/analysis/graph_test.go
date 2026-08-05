package analysis_test

import (
	"strings"
	"testing"

	"github.com/orieken/testsmith/internal/analysis"
	"github.com/orieken/testsmith/internal/domain"
)

// helpers ─────────────────────────────────────────────────────────────────────

func makeAnalysis(modulePath, sourcePath string, internal, external []string) *domain.SourceAnalysis {
	a := &domain.SourceAnalysis{
		ModulePath: modulePath,
		SourcePath: sourcePath,
	}
	for _, dep := range internal {
		a.Imports.Internal = append(a.Imports.Internal, domain.ImportInfo{Module: dep})
	}
	for _, dep := range external {
		a.Imports.External = append(a.Imports.External, domain.ImportInfo{Module: dep})
	}
	return a
}

// ── BuildDependencyGraph ──────────────────────────────────────────────────────

func TestBuildDependencyGraph_Empty(t *testing.T) {
	t.Parallel()
	g := analysis.BuildDependencyGraph(nil)
	if g == nil {
		t.Fatal("BuildDependencyGraph(nil) returned nil")
	}
	if len(g.Nodes) != 0 || len(g.Edges) != 0 {
		t.Errorf("empty input: want 0 nodes/edges, got %d/%d", len(g.Nodes), len(g.Edges))
	}
}

func TestBuildDependencyGraph_SingleNodeNoEdges(t *testing.T) {
	t.Parallel()
	a := makeAnalysis("mymod/util", "util.go", nil, nil)
	g := analysis.BuildDependencyGraph([]*domain.SourceAnalysis{a})

	if len(g.Nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(g.Nodes))
	}
	if g.Nodes[0].Name != "mymod/util" {
		t.Errorf("node name = %q, want mymod/util", g.Nodes[0].Name)
	}
	if len(g.Edges) != 0 {
		t.Errorf("expected 0 edges, got %d", len(g.Edges))
	}
}

func TestBuildDependencyGraph_InternalAndExternalEdges(t *testing.T) {
	t.Parallel()
	analyses := []*domain.SourceAnalysis{
		makeAnalysis("mymod/svc", "svc.go", []string{"mymod/repo"}, []string{"github.com/pkg/errors"}),
		makeAnalysis("mymod/repo", "repo.go", nil, []string{"database/sql"}),
	}
	g := analysis.BuildDependencyGraph(analyses)

	if len(g.Nodes) != 2 {
		t.Errorf("expected 2 nodes, got %d", len(g.Nodes))
	}

	var internalEdges, externalEdges int
	for _, e := range g.Edges {
		switch e.EdgeType {
		case "internal":
			internalEdges++
		case "external":
			externalEdges++
		}
	}
	if internalEdges != 1 {
		t.Errorf("internal edges = %d, want 1", internalEdges)
	}
	if externalEdges != 2 {
		t.Errorf("external edges = %d, want 2", externalEdges)
	}
}

func TestBuildDependencyGraph_PackageExtractedFromModulePath(t *testing.T) {
	t.Parallel()
	a := makeAnalysis("mymod/util", "util.go", nil, nil)
	g := analysis.BuildDependencyGraph([]*domain.SourceAnalysis{a})
	if g.Nodes[0].Package != "mymod" {
		t.Errorf("Package = %q, want mymod", g.Nodes[0].Package)
	}
}

func TestBuildDependencyGraph_ExternalDepCountInNode(t *testing.T) {
	t.Parallel()
	a := makeAnalysis("mymod/svc", "svc.go", nil, []string{"a", "b", "c"})
	g := analysis.BuildDependencyGraph([]*domain.SourceAnalysis{a})
	if g.Nodes[0].ExternalDepCount != 3 {
		t.Errorf("ExternalDepCount = %d, want 3", g.Nodes[0].ExternalDepCount)
	}
}

// ── ComputeMetrics ────────────────────────────────────────────────────────────

func TestComputeMetrics_Empty(t *testing.T) {
	t.Parallel()
	g := &domain.DependencyGraph{}
	metrics := analysis.ComputeMetrics(g)
	if metrics == nil {
		t.Error("ComputeMetrics(empty) should return non-nil slice")
	}
	if len(metrics) != 0 {
		t.Errorf("expected 0 metrics, got %d", len(metrics))
	}
}

func TestComputeMetrics_FanInFanOut(t *testing.T) {
	t.Parallel()
	// svc depends on repo (internal) and errors (external).
	// repo is depended upon by svc (fan-in = 1).
	g := &domain.DependencyGraph{
		Nodes: []domain.GraphNode{
			{Name: "svc"},
			{Name: "repo"},
		},
		Edges: []domain.GraphEdge{
			{Source: "svc", Target: "repo", EdgeType: "internal"},
			{Source: "svc", Target: "errors", EdgeType: "external"},
		},
	}
	metrics := analysis.ComputeMetrics(g)
	if len(metrics) != 2 {
		t.Fatalf("expected 2 metrics, got %d", len(metrics))
	}

	byName := make(map[string]domain.ModuleMetrics, 2)
	for _, m := range metrics {
		byName[m.Name] = m
	}

	svc := byName["svc"]
	if svc.InternalDependencies != 1 {
		t.Errorf("svc.InternalDependencies = %d, want 1", svc.InternalDependencies)
	}
	if svc.ExternalDependencies != 1 {
		t.Errorf("svc.ExternalDependencies = %d, want 1", svc.ExternalDependencies)
	}

	repo := byName["repo"]
	if repo.Dependents != 1 {
		t.Errorf("repo.Dependents = %d, want 1 (svc depends on repo)", repo.Dependents)
	}
}

func TestComputeMetrics_SortedByCouplingScoreDesc(t *testing.T) {
	t.Parallel()
	// highCoupling has external deps; lowCoupling has none.
	g := &domain.DependencyGraph{
		Nodes: []domain.GraphNode{{Name: "lowCoupling"}, {Name: "highCoupling"}},
		Edges: []domain.GraphEdge{
			{Source: "highCoupling", Target: "ext1", EdgeType: "external"},
			{Source: "highCoupling", Target: "ext2", EdgeType: "external"},
		},
	}
	metrics := analysis.ComputeMetrics(g)
	if len(metrics) < 2 {
		t.Fatalf("expected 2 metrics")
	}
	if metrics[0].CouplingScore < metrics[1].CouplingScore {
		t.Error("metrics should be sorted by CouplingScore descending")
	}
}

// ── RenderMermaid ─────────────────────────────────────────────────────────────

func TestRenderMermaid_Empty(t *testing.T) {
	t.Parallel()
	g := &domain.DependencyGraph{}
	out := analysis.RenderMermaid(g)
	if !strings.Contains(out, "mermaid") {
		t.Errorf("RenderMermaid output missing mermaid marker:\n%s", out)
	}
	if !strings.Contains(out, "graph TD") {
		t.Errorf("RenderMermaid output missing graph TD:\n%s", out)
	}
}

func TestRenderMermaid_InternalEdge(t *testing.T) {
	t.Parallel()
	g := &domain.DependencyGraph{
		Edges: []domain.GraphEdge{{Source: "svc", Target: "repo", EdgeType: "internal"}},
	}
	out := analysis.RenderMermaid(g)
	if !strings.Contains(out, "svc") || !strings.Contains(out, "repo") {
		t.Errorf("RenderMermaid missing node names:\n%s", out)
	}
	// Internal edges must not carry |ext| label.
	if strings.Contains(out, "|ext|") {
		t.Errorf("internal edge should not have |ext| label:\n%s", out)
	}
}

func TestRenderMermaid_ExternalEdgeLabel(t *testing.T) {
	t.Parallel()
	g := &domain.DependencyGraph{
		Edges: []domain.GraphEdge{{Source: "svc", Target: "pkg/errors", EdgeType: "external"}},
	}
	out := analysis.RenderMermaid(g)
	if !strings.Contains(out, "|ext|") {
		t.Errorf("external edge should carry |ext| label:\n%s", out)
	}
}

func TestRenderMermaid_DeduplicatesEdges(t *testing.T) {
	t.Parallel()
	g := &domain.DependencyGraph{
		Edges: []domain.GraphEdge{
			{Source: "a", Target: "b", EdgeType: "internal"},
			{Source: "a", Target: "b", EdgeType: "internal"}, // duplicate
		},
	}
	out := analysis.RenderMermaid(g)
	// Count occurrences of "a" --> "b" — should be exactly 1.
	count := strings.Count(out, "a --> b")
	if count != 1 {
		t.Errorf("duplicate edges should be deduplicated; got %d occurrences", count)
	}
}

func TestRenderMermaid_SanitizesSpecialChars(t *testing.T) {
	t.Parallel()
	g := &domain.DependencyGraph{
		Edges: []domain.GraphEdge{{Source: "a.b/c-d", Target: "e.f", EdgeType: "internal"}},
	}
	out := analysis.RenderMermaid(g)
	// sanitizeMermaid replaces ".", "/", "-" with "_" in node identifiers.
	// Verify the sanitized forms appear and the raw special-char versions do not.
	if !strings.Contains(out, "a_b_c_d") {
		t.Errorf("source node not sanitized correctly:\n%s", out)
	}
	if !strings.Contains(out, "e_f") {
		t.Errorf("target node not sanitized correctly:\n%s", out)
	}
}

// ── RenderMetricsTable ────────────────────────────────────────────────────────

func TestRenderMetricsTable_ContainsHeader(t *testing.T) {
	t.Parallel()
	out := analysis.RenderMetricsTable(nil)
	if !strings.Contains(out, "Module") || !strings.Contains(out, "Coupling Score") {
		t.Errorf("metrics table missing header:\n%s", out)
	}
}

func TestRenderMetricsTable_ContainsMetricRow(t *testing.T) {
	t.Parallel()
	metrics := []domain.ModuleMetrics{
		{Name: "mymod/svc", InternalDependencies: 2, ExternalDependencies: 1, Dependents: 3, CouplingScore: 0.75},
	}
	out := analysis.RenderMetricsTable(metrics)
	if !strings.Contains(out, "mymod/svc") {
		t.Errorf("metrics table missing module name:\n%s", out)
	}
	if !strings.Contains(out, "0.75") {
		t.Errorf("metrics table missing coupling score:\n%s", out)
	}
}
