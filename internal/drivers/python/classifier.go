package python

import (
	"strings"

	"github.com/orieken/assay/internal/domain"
)

func classifyDependency(imp domain.ImportInfo, ctx *domain.ProjectContext) domain.DependencyCategory {
	// Relative imports are always internal.
	if strings.HasPrefix(imp.Module, ".") {
		return domain.DepInternal
	}

	root := rootPkg(imp.Module)

	if pythonStdlib[root] {
		return domain.DepStdlib
	}
	if _, ok := ctx.PackageMap[root]; ok {
		return domain.DepInternal
	}
	return domain.DepExternal
}

func rootPkg(module string) string {
	if idx := strings.IndexByte(module, '.'); idx != -1 {
		return module[:idx]
	}
	return module
}
