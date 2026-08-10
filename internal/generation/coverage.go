package generation

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/orieken/assay/internal/analysis"
	"github.com/orieken/assay/internal/domain"
)

// DetectCoverage inspects each source analysis to determine whether a
// corresponding test file exists and how complete it is.
func DetectCoverage(
	analyses []*domain.SourceAnalysis,
	driver domain.LanguageDriver,
) []domain.CoverageGap {
	var gaps []domain.CoverageGap

	for _, a := range analyses {
		testPath, err := driver.DeriveTestPath(a.SourcePath, a.Project)
		if err != nil {
			continue
		}

		status := assessTestFile(testPath, driver)
		if status == domain.CoverageFull {
			continue
		}

		cmd := fmt.Sprintf("assay generate %s", a.SourcePath)
		if status == domain.CoveragePartial || status == domain.CoverageSkeletonOnly {
			cmd += " --overwrite"
		}

		gaps = append(gaps, domain.CoverageGap{
			SourcePath:       a.SourcePath,
			Status:           status,
			ExternalDeps:     len(a.Imports.External),
			SuggestedCommand: cmd,
		})
	}
	return gaps
}

// PrioritizeGaps sorts gaps by computed priority score descending.
func PrioritizeGaps(gaps []domain.CoverageGap, metrics []domain.ModuleMetrics) []domain.CoverageGap {
	metricsByName := make(map[string]domain.ModuleMetrics, len(metrics))
	for _, m := range metrics {
		metricsByName[m.Name] = m
	}

	maxFanOut := 1
	maxDeps := 1
	for _, g := range gaps {
		if m, ok := metricsByName[moduleStem(g.SourcePath)]; ok {
			if m.Dependents > maxFanOut {
				maxFanOut = m.Dependents
			}
		}
		if g.ExternalDeps > maxDeps {
			maxDeps = g.ExternalDeps
		}
	}

	for i := range gaps {
		g := &gaps[i]
		statusWeight := statusScore(g.Status)
		fanIn := 0.0
		if m, ok := metricsByName[moduleStem(g.SourcePath)]; ok {
			fanIn = float64(m.Dependents) / float64(maxFanOut)
		}
		fanOut := float64(g.ExternalDeps) / float64(maxDeps)
		g.PriorityScore = (statusWeight + 0.4*fanOut + 0.3*fanIn) / 1.7
	}

	sort.Slice(gaps, func(i, j int) bool {
		return gaps[i].PriorityScore > gaps[j].PriorityScore
	})
	return gaps
}

// GenerateReport renders a prioritised Markdown coverage report.
func GenerateReport(gaps []domain.CoverageGap, totalSources int) string {
	covered := totalSources - len(gaps)
	pct := 0
	if totalSources > 0 {
		pct = covered * 100 / totalSources
	}

	var sb strings.Builder
	sb.WriteString("# Assay Coverage Gap Report\n\n")
	fmt.Fprintf(&sb, "**%d / %d source files have test coverage (%d%%)**\n\n", covered, totalSources, pct)

	if len(gaps) == 0 {
		sb.WriteString("✓ No coverage gaps found.\n")
		return sb.String()
	}

	sb.WriteString("| Priority | File | Status | Ext Deps | Command |\n")
	sb.WriteString("|----------|------|--------|----------|---------|\n")

	for _, g := range gaps {
		rel := g.SourcePath
		fmt.Fprintf(&sb, "| %.2f | `%s` | %s | %d | `%s` |\n",
			g.PriorityScore, rel, g.Status, g.ExternalDeps, g.SuggestedCommand)
	}
	return sb.String()
}

// assessTestFile categorises how complete a test file is.
func assessTestFile(testPath string, driver domain.LanguageDriver) domain.CoverageStatus {
	data, err := os.ReadFile(testPath)
	if errors.Is(err, os.ErrNotExist) {
		return domain.CoverageNoTest
	}
	if err != nil {
		return domain.CoverageNoTest
	}

	content := string(data)
	cfg := driver.GetTestFrameworkConfig()
	return classifyTestContent(content, cfg)
}

// classifyTestContent checks whether a test file is a skeleton, partial, or full.
func classifyTestContent(content string, cfg domain.TestFrameworkConfig) domain.CoverageStatus {
	_ = cfg
	lines := strings.Split(content, "\n")

	hasRealAssertion := false
	hasStub := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Skeleton indicators (language-agnostic heuristics).
		if trimmed == "pass" || trimmed == "// TODO: implement" ||
			strings.HasPrefix(trimmed, "# TODO") || trimmed == "throw new NotImplementedException();" {
			hasStub = true
		}
		// Real assertion indicators.
		if strings.Contains(trimmed, "assert") || strings.Contains(trimmed, "Assert") ||
			strings.Contains(trimmed, "expect(") || strings.Contains(trimmed, "should.") {
			hasRealAssertion = true
		}
	}

	switch {
	case hasRealAssertion && hasStub:
		return domain.CoveragePartial
	case hasRealAssertion:
		return domain.CoverageFull
	case hasStub:
		return domain.CoverageSkeletonOnly
	default:
		return domain.CoverageSkeletonOnly
	}
}

func statusScore(s domain.CoverageStatus) float64 {
	switch s {
	case domain.CoverageNoTest:
		return 1.0
	case domain.CoverageSkeletonOnly:
		return 0.6
	case domain.CoveragePartial:
		return 0.3
	default:
		return 0.0
	}
}

func moduleStem(path string) string {
	base := path
	// Strip extension.
	if idx := strings.LastIndex(base, "."); idx != -1 {
		base = base[:idx]
	}
	// Use the last two path components as the module key.
	parts := strings.Split(strings.ReplaceAll(base, "\\", "/"), "/")
	if len(parts) >= 2 {
		return strings.Join(parts[len(parts)-2:], "/")
	}
	return base
}

// ClassifyContentForTest exposes classifyTestContent for package-external tests.
func ClassifyContentForTest(content string, cfg domain.TestFrameworkConfig) domain.CoverageStatus {
	return classifyTestContent(content, cfg)
}

// GapsForAnalyses is a convenience wrapper used by the gaps command.
func GapsForAnalyses(analyses []*domain.SourceAnalysis, driver domain.LanguageDriver) ([]domain.CoverageGap, error) {
	gaps := DetectCoverage(analyses, driver)
	graph := analysis.BuildDependencyGraph(analyses)
	metrics := analysis.ComputeMetrics(graph)
	return PrioritizeGaps(gaps, metrics), nil
}
