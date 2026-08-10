package csharp

import (
	"strings"

	"github.com/orieken/assay/internal/domain"
)

// dotnetBclPrefixes are namespaces that ship with the .NET BCL.
var dotnetBclPrefixes = []string{
	"System", "Microsoft.CSharp", "Microsoft.Win32",
	"Microsoft.Extensions.Logging", "Microsoft.Extensions.DependencyInjection",
	"Microsoft.Extensions.Configuration", "Microsoft.Extensions.Hosting",
	"mscorlib",
}

func classifyDependency(imp domain.ImportInfo, ctx *domain.ProjectContext) domain.DependencyCategory {
	ns := imp.Module

	for _, prefix := range dotnetBclPrefixes {
		if ns == prefix || strings.HasPrefix(ns, prefix+".") {
			return domain.DepStdlib
		}
	}

	rootNS, _ := ctx.Metadata["root_namespace"].(string)
	if rootNS != "" && (ns == rootNS || strings.HasPrefix(ns, rootNS+".")) {
		return domain.DepInternal
	}

	return domain.DepExternal
}
