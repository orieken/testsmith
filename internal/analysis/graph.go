package analysis

import (
	"fmt"
	"sort"
	"strings"

	"github.com/orieken/assay/internal/domain"
)

// BuildDependencyGraph constructs a DependencyGraph from a set of SourceAnalyses.
func BuildDependencyGraph(analyses []*domain.SourceAnalysis) *domain.DependencyGraph {
	g := &domain.DependencyGraph{}
	nodeSet := make(map[string]bool)

	for _, a := range analyses {
		nodeName := a.ModulePath
		if !nodeSet[nodeName] {
			nodeSet[nodeName] = true
			extCount := len(a.Imports.External)
			pkg := ""
			if parts := strings.SplitN(nodeName, "/", 2); len(parts) > 1 {
				pkg = parts[0]
			}
			g.Nodes = append(g.Nodes, domain.GraphNode{
				Name:             nodeName,
				Path:             a.SourcePath,
				Package:          pkg,
				ExternalDepCount: extCount,
			})
		}

		for _, imp := range a.Imports.Internal {
			g.Edges = append(g.Edges, domain.GraphEdge{
				Source:   nodeName,
				Target:   imp.Module,
				EdgeType: "internal",
			})
		}
		for _, imp := range a.Imports.External {
			g.Edges = append(g.Edges, domain.GraphEdge{
				Source:   nodeName,
				Target:   imp.Module,
				EdgeType: "external",
			})
		}
	}
	return g
}

// ComputeMetrics calculates fan-in, fan-out, and coupling scores for each node.
func ComputeMetrics(g *domain.DependencyGraph) []domain.ModuleMetrics {
	fanOut := make(map[string]int)
	fanOutExt := make(map[string]int)
	fanIn := make(map[string]int)

	for _, e := range g.Edges {
		if e.EdgeType == "internal" {
			fanOut[e.Source]++
			fanIn[e.Target]++
		} else {
			fanOutExt[e.Source]++
		}
	}

	maxFanOut := 1
	maxFanIn := 1
	for _, n := range g.Nodes {
		if v := fanOutExt[n.Name] + fanOut[n.Name]; v > maxFanOut {
			maxFanOut = v
		}
		if v := fanIn[n.Name]; v > maxFanIn {
			maxFanIn = v
		}
	}

	metrics := make([]domain.ModuleMetrics, 0, len(g.Nodes))
	for _, n := range g.Nodes {
		fo := fanOut[n.Name]
		foExt := fanOutExt[n.Name]
		fi := fanIn[n.Name]
		score := (float64(foExt)*0.6 + float64(fi)*0.4) / float64(maxFanOut)
		metrics = append(metrics, domain.ModuleMetrics{
			Name:                 n.Name,
			InternalDependencies: fo,
			ExternalDependencies: foExt,
			Dependents:           fi,
			CouplingScore:        score,
		})
	}

	sort.Slice(metrics, func(i, j int) bool {
		return metrics[i].CouplingScore > metrics[j].CouplingScore
	})
	return metrics
}

// RenderMermaid produces a Mermaid flowchart string for the dependency graph.
func RenderMermaid(g *domain.DependencyGraph) string {
	var sb strings.Builder
	sb.WriteString("```mermaid\ngraph TD\n")
	seen := make(map[string]bool)
	for _, e := range g.Edges {
		key := e.Source + "->" + e.Target
		if seen[key] {
			continue
		}
		seen[key] = true
		src := sanitizeMermaid(e.Source)
		tgt := sanitizeMermaid(e.Target)
		if e.EdgeType == "external" {
			fmt.Fprintf(&sb, "    %s -->|ext| %s\n", src, tgt)
		} else {
			fmt.Fprintf(&sb, "    %s --> %s\n", src, tgt)
		}
	}
	sb.WriteString("```\n")
	return sb.String()
}

// RenderMetricsTable produces a Markdown table of module metrics.
func RenderMetricsTable(metrics []domain.ModuleMetrics) string {
	var sb strings.Builder
	sb.WriteString("| Module | Internal Deps | External Deps | Dependents | Coupling Score |\n")
	sb.WriteString("|--------|--------------|--------------|------------|----------------|\n")
	for _, m := range metrics {
		fmt.Fprintf(&sb, "| %s | %d | %d | %d | %.2f |\n",
			m.Name, m.InternalDependencies, m.ExternalDependencies, m.Dependents, m.CouplingScore)
	}
	return sb.String()
}

func sanitizeMermaid(s string) string {
	r := strings.NewReplacer(".", "_", "/", "_", "-", "_")
	return r.Replace(s)
}
