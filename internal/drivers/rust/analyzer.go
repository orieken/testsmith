package rust

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/orieken/assay/internal/domain"
)

// Regex patterns for Rust source analysis.
var (
	rePubFn       = regexp.MustCompile(`(?m)^pub(?:\(crate\))?\s+(?:async\s+)?fn\s+(\w+)`)
	rePubStruct   = regexp.MustCompile(`(?m)^pub(?:\(crate\))?\s+struct\s+(\w+)`)
	rePubEnum     = regexp.MustCompile(`(?m)^pub(?:\(crate\))?\s+enum\s+(\w+)`)
	rePubTrait    = regexp.MustCompile(`(?m)^pub(?:\(crate\))?\s+trait\s+(\w+)`)
	reImpl        = regexp.MustCompile(`(?m)^impl(?:<[^>]+>)?\s+(\w+)`)
	rePubMethod   = regexp.MustCompile(`(?m)^\s+pub(?:\(crate\))?\s+(?:async\s+)?fn\s+(\w+)`)
	reUseDecl     = regexp.MustCompile(`(?m)^use\s+([\w::{}, ]+);`)
	reExternCrate = regexp.MustCompile(`(?m)^extern\s+crate\s+(\w+);`)
)

func analyzeFile(path string, ctx *domain.ProjectContext) (*domain.SourceAnalysis, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	content := string(src)

	imports := extractImports(content, ctx)
	publicAPI := extractPublicAPI(content)
	modulePath, _ := deriveModulePath(path, ctx)

	classified := domain.ClassifiedImports{}
	for _, imp := range imports {
		switch classifyDependency(imp, ctx) {
		case domain.DepStdlib:
			classified.Stdlib = append(classified.Stdlib, imp)
		case domain.DepInternal:
			classified.Internal = append(classified.Internal, imp)
		default:
			classified.External = append(classified.External, imp)
		}
	}

	return &domain.SourceAnalysis{
		SourcePath: path,
		ModulePath: modulePath,
		Imports:    classified,
		PublicAPI:  publicAPI,
		Project:    ctx,
		RawSource:  content,
	}, nil
}

func extractImports(src string, _ *domain.ProjectContext) []domain.ImportInfo {
	var imports []domain.ImportInfo
	seen := make(map[string]bool)

	for _, match := range reUseDecl.FindAllStringSubmatch(src, -1) {
		module := normalizeUseDecl(match[1])
		if !seen[module] {
			seen[module] = true
			imports = append(imports, domain.ImportInfo{Module: module})
		}
	}

	for _, match := range reExternCrate.FindAllStringSubmatch(src, -1) {
		if !seen[match[1]] {
			seen[match[1]] = true
			imports = append(imports, domain.ImportInfo{Module: match[1]})
		}
	}

	return imports
}

// normalizeUseDecl extracts the root crate from a use declaration.
func normalizeUseDecl(decl string) string {
	decl = strings.TrimSpace(decl)
	// Drop everything after the first :: or {
	if idx := strings.IndexAny(decl, ":{"); idx != -1 {
		decl = decl[:idx]
	}
	return strings.TrimSpace(decl)
}

func extractPublicAPI(src string) []domain.PublicMember {
	var members []domain.PublicMember

	// Collect impl blocks to associate methods with types.
	implMethods := collectImplMethods(src)

	// Public functions (not inside impl blocks).
	for _, match := range rePubFn.FindAllStringSubmatch(src, -1) {
		name := match[1]
		members = append(members, domain.PublicMember{
			Name: name,
			Kind: domain.KindFunction,
		})
	}

	// Public structs, enums, traits.
	for _, match := range rePubStruct.FindAllStringSubmatch(src, -1) {
		name := match[1]
		members = append(members, domain.PublicMember{
			Name:    name,
			Kind:    domain.KindClass,
			Methods: implMethods[name],
		})
	}

	for _, match := range rePubEnum.FindAllStringSubmatch(src, -1) {
		name := match[1]
		members = append(members, domain.PublicMember{
			Name:    name,
			Kind:    domain.KindClass,
			Methods: implMethods[name],
		})
	}

	for _, match := range rePubTrait.FindAllStringSubmatch(src, -1) {
		name := match[1]
		members = append(members, domain.PublicMember{
			Name: name,
			Kind: domain.KindInterface,
		})
	}

	return members
}

// collectImplMethods maps type name → public methods from its impl block.
func collectImplMethods(src string) map[string][]domain.MethodInfo {
	result := make(map[string][]domain.MethodInfo)

	// Split on impl blocks.
	implMatches := reImpl.FindAllStringSubmatchIndex(src, -1)
	for i, idx := range implMatches {
		typeName := src[idx[2]:idx[3]]

		// Extract block content until the next impl or end of file.
		blockStart := idx[1]
		blockEnd := len(src)
		if i+1 < len(implMatches) {
			blockEnd = implMatches[i+1][0]
		}
		block := src[blockStart:blockEnd]

		for _, m := range rePubMethod.FindAllStringSubmatch(block, -1) {
			result[typeName] = append(result[typeName], domain.MethodInfo{
				Name:     m[1],
				IsPublic: true,
			})
		}
	}

	return result
}

func deriveTestPath(sourcePath string, ctx *domain.ProjectContext) (string, error) {
	// Integration tests go in tests/<module>.rs; unit tests are inline.
	// For Assay, we generate an integration test file.
	base := filepath.Base(sourcePath)
	ext := filepath.Ext(base)
	name := base[:len(base)-len(ext)]

	testsDir := filepath.Join(ctx.Root, "tests")
	return filepath.Join(testsDir, name+"_test.rs"), nil
}

func deriveModulePath(sourcePath string, ctx *domain.ProjectContext) (string, error) {
	rel, err := filepath.Rel(ctx.Root, sourcePath)
	if err != nil {
		return filepath.Base(sourcePath), nil //nolint:nilerr // graceful fallback to basename
	}
	ext := filepath.Ext(rel)
	mod := rel[:len(rel)-len(ext)]
	return strings.ReplaceAll(filepath.ToSlash(mod), "/", "::"), nil
}
