package java

import (
	"strings"

	"github.com/orieken/assay/internal/domain"
)

// javaStdlibPrefixes are root packages that ship with the JDK.
var javaStdlibPrefixes = []string{
	"java.", "javax.", "jakarta.", "sun.", "com.sun.",
	"org.xml.", "org.w3c.", "org.ietf.", "org.omg.",
}

func classifyDependency(imp domain.ImportInfo, ctx *domain.ProjectContext) domain.DependencyCategory {
	path := imp.Module

	for _, prefix := range javaStdlibPrefixes {
		if strings.HasPrefix(path, prefix) {
			return domain.DepStdlib
		}
	}

	basePkg, _ := ctx.Metadata["base_package"].(string)
	if basePkg != "" && strings.HasPrefix(path, basePkg) {
		return domain.DepInternal
	}

	return domain.DepExternal
}
