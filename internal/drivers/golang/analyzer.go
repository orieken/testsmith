package golang

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"

	"github.com/orieken/assay/internal/domain"
)

func analyzeFile(path string, ctx *domain.ProjectContext) (*domain.SourceAnalysis, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, src, parser.ParseComments)
	if err != nil {
		// Non-fatal: return partial analysis on syntax errors.
		return partialAnalysis(path, ctx, string(src)), nil //nolint:nilerr
	}

	imports := extractImports(file)
	publicAPI := extractPublicAPI(file)

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

	modulePath, _ := deriveModulePath(path, ctx)

	return &domain.SourceAnalysis{
		SourcePath: path,
		ModulePath: modulePath,
		Imports:    classified,
		PublicAPI:  publicAPI,
		Project:    ctx,
		RawSource:  string(src),
	}, nil
}

func partialAnalysis(path string, ctx *domain.ProjectContext, src string) *domain.SourceAnalysis {
	modulePath, _ := deriveModulePath(path, ctx)
	return &domain.SourceAnalysis{
		SourcePath: path,
		ModulePath: modulePath,
		Project:    ctx,
		RawSource:  src,
	}
}

// extractImports collects all import paths from the file.
func extractImports(file *ast.File) []domain.ImportInfo {
	var imports []domain.ImportInfo
	for _, imp := range file.Imports {
		path := strings.Trim(imp.Path.Value, `"`)
		alias := ""
		if imp.Name != nil {
			alias = imp.Name.Name
		}
		imports = append(imports, domain.ImportInfo{Module: path, Alias: alias})
	}
	return imports
}

// extractPublicAPI returns exported top-level functions, types, and their methods.
func extractPublicAPI(file *ast.File) []domain.PublicMember {
	// Collect methods keyed by receiver type name.
	methodsByType := collectMethods(file)

	var members []domain.PublicMember

	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if d.Recv != nil {
				continue // methods handled via methodsByType
			}
			if !d.Name.IsExported() {
				continue
			}
			params := extractParams(d.Type.Params)
			members = append(members, domain.PublicMember{
				Name:       d.Name.Name,
				Kind:       domain.KindFunction,
				Parameters: params,
			})

		case *ast.GenDecl:
			if d.Tok != token.TYPE {
				continue
			}
			for _, spec := range d.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok || !ts.Name.IsExported() {
					continue
				}
				switch ts.Type.(type) {
				case *ast.StructType, *ast.InterfaceType:
					methods := methodsByType[ts.Name.Name]
					members = append(members, domain.PublicMember{
						Name:    ts.Name.Name,
						Kind:    domain.KindClass,
						Methods: methods,
					})
				}
			}
		}
	}
	return members
}

// collectMethods maps receiver type name → []MethodInfo for all exported methods.
func collectMethods(file *ast.File) map[string][]domain.MethodInfo {
	result := make(map[string][]domain.MethodInfo)
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv == nil || !fn.Name.IsExported() {
			continue
		}
		typeName := receiverTypeName(fn.Recv)
		if typeName == "" {
			continue
		}
		params := extractParams(fn.Type.Params)
		result[typeName] = append(result[typeName], domain.MethodInfo{
			Name:       fn.Name.Name,
			Parameters: params,
			IsPublic:   true,
		})
	}
	return result
}

func receiverTypeName(recv *ast.FieldList) string {
	if recv == nil || len(recv.List) == 0 {
		return ""
	}
	switch t := recv.List[0].Type.(type) {
	case *ast.StarExpr:
		if ident, ok := t.X.(*ast.Ident); ok {
			return ident.Name
		}
	case *ast.Ident:
		return t.Name
	}
	return ""
}

func extractParams(fields *ast.FieldList) []domain.ParamInfo {
	if fields == nil {
		return nil
	}
	var params []domain.ParamInfo
	for _, field := range fields.List {
		typeStr := typeString(field.Type)
		if len(field.Names) == 0 {
			// Unnamed parameter (e.g. in interface methods).
			params = append(params, domain.ParamInfo{TypeHint: typeStr})
			continue
		}
		for _, name := range field.Names {
			params = append(params, domain.ParamInfo{Name: name.Name, TypeHint: typeStr})
		}
	}
	return params
}

// typeString renders an ast.Expr as a readable type string.
func typeString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + typeString(t.X)
	case *ast.ArrayType:
		return "[]" + typeString(t.Elt)
	case *ast.MapType:
		return "map[" + typeString(t.Key) + "]" + typeString(t.Value)
	case *ast.SelectorExpr:
		return typeString(t.X) + "." + t.Sel.Name
	case *ast.InterfaceType:
		return "interface{}"
	case *ast.Ellipsis:
		return "..." + typeString(t.Elt)
	default:
		return "any"
	}
}
