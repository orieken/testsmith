package typescript

import (
	"strings"

	"github.com/orieken/testsmith/internal/domain"
)

func classifyDependency(imp domain.ImportInfo, _ *domain.ProjectContext) domain.DependencyCategory {
	module := imp.Module

	// Relative/path-alias imports are internal.
	if strings.HasPrefix(module, "./") || strings.HasPrefix(module, "../") {
		return domain.DepInternal
	}

	// The node: scheme is the definitive Node.js built-in prefix.
	if strings.HasPrefix(module, "node:") {
		return domain.DepStdlib
	}

	root := rootPkg(module)
	if nodeStdlib[root] {
		return domain.DepStdlib
	}

	return domain.DepExternal
}

// rootPkg returns the importable root of a module path.
// Scoped packages (@scope/name) are returned as-is up to the second segment.
func rootPkg(module string) string {
	if strings.HasPrefix(module, "@") {
		parts := strings.SplitN(module, "/", 3)
		if len(parts) >= 2 {
			return parts[0] + "/" + parts[1]
		}
		return module
	}
	if idx := strings.IndexByte(module, '/'); idx != -1 {
		return module[:idx]
	}
	return module
}
