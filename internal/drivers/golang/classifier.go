package golang

import (
	"strings"

	"github.com/orieken/testsmith/internal/domain"
)

func classifyDependency(imp domain.ImportInfo, ctx *domain.ProjectContext) domain.DependencyCategory {
	path := imp.Module
	mod, _ := ctx.Metadata["module"].(string)

	// Internal: starts with the module path.
	if mod != "" && (path == mod || strings.HasPrefix(path, mod+"/")) {
		return domain.DepInternal
	}

	// Stdlib: first path segment has no dot (e.g. "fmt", "net/http", "os/exec").
	first := path
	if idx := strings.IndexByte(path, '/'); idx != -1 {
		first = path[:idx]
	}
	if !strings.Contains(first, ".") {
		return domain.DepStdlib
	}

	return domain.DepExternal
}
