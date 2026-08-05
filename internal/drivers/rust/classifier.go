package rust

import (
	"strings"

	"github.com/orieken/testsmith/internal/domain"
)

// stdlibRoots is the set of Rust standard library crate roots.
var stdlibRoots = map[string]bool{
	"std":        true,
	"core":       true,
	"alloc":      true,
	"proc_macro": true,
	"test":       true,
}

func classifyDependency(imp domain.ImportInfo, ctx *domain.ProjectContext) domain.DependencyCategory {
	module := imp.Module

	// Relative paths (self::, super::, crate::) are internal.
	if strings.HasPrefix(module, "self") ||
		strings.HasPrefix(module, "super") ||
		strings.HasPrefix(module, "crate") {
		return domain.DepInternal
	}

	// Check standard library roots.
	root := rootCrate(module)
	if stdlibRoots[root] {
		return domain.DepStdlib
	}

	// Crate name matches the project crate name → internal.
	if ctx != nil {
		if crate, ok := ctx.Metadata["crate"].(string); ok && root == crate {
			return domain.DepInternal
		}
	}

	return domain.DepExternal
}

// rootCrate returns the first segment of a :: -separated crate path.
func rootCrate(module string) string {
	if idx := strings.Index(module, "::"); idx != -1 {
		return module[:idx]
	}
	return module
}
