package kotlin

import (
	"path/filepath"
	"strings"

	"github.com/orieken/testsmith/internal/domain"
)

// kotlinStdlib is the set of Kotlin/JVM standard library package roots.
var kotlinStdlib = map[string]bool{
	"kotlin":  true,
	"kotlinx": true,
	"java":    true,
	"javax":   true,
	"sun":     true,
	"com.sun": true,
	"android": true, // Android SDK — treated as stdlib for classification purposes.
}

func classifyDependency(imp domain.ImportInfo, ctx *domain.ProjectContext) domain.DependencyCategory {
	module := imp.Module

	root := rootPackage(module)
	if kotlinStdlib[root] {
		return domain.DepStdlib
	}

	// Check against the project's own group ID if available.
	if ctx != nil {
		if moduleName, ok := ctx.Metadata["module"].(string); ok {
			if strings.HasPrefix(module, moduleName) {
				return domain.DepInternal
			}
		}
		// Relative-style intra-project imports don't exist in Kotlin,
		// but a module can have internal sub-packages — detect by matching
		// the project root's last segment against the package prefix.
		projectPkg := strings.ToLower(strings.ReplaceAll(filepath.Base(ctx.Root), "-", "."))
		if strings.HasPrefix(strings.ToLower(module), projectPkg) {
			return domain.DepInternal
		}
	}

	return domain.DepExternal
}

// rootPackage returns the first segment of a dot-separated package name.
func rootPackage(module string) string {
	if idx := strings.IndexByte(module, '.'); idx != -1 {
		return module[:idx]
	}
	return module
}
