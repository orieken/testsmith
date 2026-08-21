package kotlin

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/orieken/assay/internal/domain"
)

var (
	rePackage      = regexp.MustCompile(`(?m)^package\s+([\w.]+)`)
	reImport       = regexp.MustCompile(`(?m)^import\s+([\w.]+(?:\.\*)?)`)
	rePubFun       = regexp.MustCompile(`(?m)^(?:public\s+)?(?:suspend\s+)?fun\s+(\w+)`)
	rePubClass     = regexp.MustCompile(`(?m)^(?:public\s+)?(?:open\s+|abstract\s+|data\s+|sealed\s+)?class\s+(\w+)`)
	rePubObject    = regexp.MustCompile(`(?m)^(?:public\s+)?object\s+(\w+)`)
	rePubInterface = regexp.MustCompile(`(?m)^(?:public\s+)?interface\s+(\w+)`)
	reMethod       = regexp.MustCompile(`(?m)^\s+(?:public\s+)?(?:override\s+)?(?:suspend\s+)?fun\s+(\w+)`)
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
	for _, match := range reImport.FindAllStringSubmatch(src, -1) {
		mod := match[1]
		if !seen[mod] {
			seen[mod] = true
			imports = append(imports, domain.ImportInfo{Module: mod})
		}
	}
	return imports
}

func extractPublicAPI(src string) []domain.PublicMember {
	var members []domain.PublicMember

	// Top-level functions.
	for _, match := range rePubFun.FindAllStringSubmatch(src, -1) {
		members = append(members, domain.PublicMember{
			Name: match[1],
			Kind: domain.KindFunction,
		})
	}

	// Classes.
	for _, match := range rePubClass.FindAllStringSubmatch(src, -1) {
		name := match[1]
		members = append(members, domain.PublicMember{
			Name:    name,
			Kind:    domain.KindClass,
			Methods: extractMembersOf(src, name),
		})
	}

	// Objects (singletons).
	for _, match := range rePubObject.FindAllStringSubmatch(src, -1) {
		name := match[1]
		members = append(members, domain.PublicMember{
			Name:    name,
			Kind:    domain.KindClass,
			Methods: extractMembersOf(src, name),
		})
	}

	// Interfaces.
	for _, match := range rePubInterface.FindAllStringSubmatch(src, -1) {
		members = append(members, domain.PublicMember{
			Name: match[1],
			Kind: domain.KindInterface,
		})
	}

	return members
}

// extractMembersOf returns the public methods for a named class/object by scanning
// the block that follows its declaration. This is a best-effort regex approach.
func extractMembersOf(src, typeName string) []domain.MethodInfo {
	// Find the start of the type body.
	pattern := regexp.MustCompile(`(?m)(?:class|object)\s+` + regexp.QuoteMeta(typeName) + `[^\{]*\{`)
	loc := pattern.FindStringIndex(src)
	if loc == nil {
		return nil
	}

	// Extract the block up to the next top-level declaration.
	block := src[loc[1]:]
	nextDecl := regexp.MustCompile(`(?m)^(?:public\s+)?(?:open\s+|abstract\s+|data\s+|sealed\s+)?(?:class|object|interface|fun)\s+\w+`)
	if nextIdx := nextDecl.FindStringIndex(block); nextIdx != nil {
		block = block[:nextIdx[0]]
	}

	var methods []domain.MethodInfo
	for _, m := range reMethod.FindAllStringSubmatch(block, -1) {
		methods = append(methods, domain.MethodInfo{
			Name:     m[1],
			IsPublic: true,
		})
	}
	return methods
}

func deriveTestPath(sourcePath string, ctx *domain.ProjectContext) (string, error) {
	rel, err := filepath.Rel(ctx.Root, sourcePath)
	if err != nil {
		rel = filepath.Base(sourcePath)
	}

	// src/main/kotlin/com/example/Foo.kt → src/test/kotlin/com/example/FooTest.kt
	rel = filepath.ToSlash(rel)
	rel = strings.Replace(rel, "src/main/kotlin/", "src/test/kotlin/", 1)
	rel = strings.Replace(rel, "main/", "test/", 1)

	ext := filepath.Ext(rel)
	base := rel[:len(rel)-len(ext)]
	return filepath.Join(ctx.Root, filepath.FromSlash(base+"Test"+ext)), nil
}

func deriveModulePath(sourcePath string, ctx *domain.ProjectContext) (string, error) {
	src, err := os.ReadFile(sourcePath)
	if err != nil {
		return filepath.Base(sourcePath), nil //nolint:nilerr // graceful fallback to basename
	}

	if match := rePackage.FindSubmatch(src); len(match) > 1 {
		pkg := string(match[1])
		base := filepath.Base(sourcePath)
		ext := filepath.Ext(base)
		return pkg + "." + base[:len(base)-len(ext)], nil
	}

	rel, err2 := filepath.Rel(ctx.Root, sourcePath)
	if err2 != nil {
		return filepath.Base(sourcePath), nil //nolint:nilerr // graceful fallback to basename
	}
	ext := filepath.Ext(rel)
	return filepath.ToSlash(rel[:len(rel)-len(ext)]), nil
}
