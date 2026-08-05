package java

import (
	"context"
	"fmt"
	"os"

	sitter "github.com/smacker/go-tree-sitter"
	sitterjava "github.com/smacker/go-tree-sitter/java"

	"github.com/orieken/testsmith/internal/domain"
)

func analyzeFile(path string, ctx *domain.ProjectContext) (*domain.SourceAnalysis, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	parser := sitter.NewParser()
	parser.SetLanguage(sitterjava.GetLanguage())
	tree, err := parser.ParseCtx(context.Background(), nil, src)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	root := tree.RootNode()

	imports := extractImports(root, src)
	publicAPI := extractPublicAPI(root, src)

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

func extractImports(root *sitter.Node, src []byte) []domain.ImportInfo {
	var imports []domain.ImportInfo
	for i := 0; i < int(root.ChildCount()); i++ {
		child := root.Child(i)
		if child.Type() != "import_declaration" {
			continue
		}
		// The scoped_identifier child holds the full dotted package path.
		for j := 0; j < int(child.ChildCount()); j++ {
			gc := child.Child(j)
			if gc.Type() == "scoped_identifier" || gc.Type() == "identifier" {
				path := gc.Content(src)
				// Strip trailing class name for package-level classification.
				imports = append(imports, domain.ImportInfo{Module: path})
				break
			}
		}
	}
	return imports
}

func extractPublicAPI(root *sitter.Node, src []byte) []domain.PublicMember {
	var members []domain.PublicMember
	for i := 0; i < int(root.ChildCount()); i++ {
		child := root.Child(i)
		if child.Type() != "class_declaration" {
			continue
		}
		if !hasPublicModifier(child, src) {
			continue
		}
		members = append(members, parseClass(child, src)...)
	}
	return members
}

func parseClass(n *sitter.Node, src []byte) []domain.PublicMember {
	nameNode := childByType(n, "identifier")
	if nameNode == nil {
		return nil
	}
	className := nameNode.Content(src)

	body := childByType(n, "class_body")
	if body == nil {
		return nil
	}

	methods := extractMethods(body, src)

	// Emit the class itself.
	members := []domain.PublicMember{
		{Name: className, Kind: domain.KindClass, Methods: methods},
	}

	// Also emit public static methods as standalone functions for coverage purposes.
	for _, m := range methods {
		if isStaticMethod(body, m.Name, src) {
			members = append(members, domain.PublicMember{
				Name:       className + "." + m.Name,
				Kind:       domain.KindFunction,
				Parameters: m.Parameters,
			})
		}
	}

	return members
}

func extractMethods(body *sitter.Node, src []byte) []domain.MethodInfo {
	var methods []domain.MethodInfo
	for i := 0; i < int(body.ChildCount()); i++ {
		child := body.Child(i)
		if child.Type() != "method_declaration" {
			continue
		}
		if !hasPublicModifier(child, src) {
			continue
		}
		nameNode := childByType(child, "identifier")
		if nameNode == nil {
			continue
		}
		params := extractFormalParams(childByType(child, "formal_parameters"), src)
		methods = append(methods, domain.MethodInfo{
			Name:       nameNode.Content(src),
			Parameters: params,
			IsPublic:   true,
		})
	}
	return methods
}

func isStaticMethod(body *sitter.Node, methodName string, src []byte) bool {
	for i := 0; i < int(body.ChildCount()); i++ {
		child := body.Child(i)
		if child.Type() != "method_declaration" {
			continue
		}
		nameNode := childByType(child, "identifier")
		if nameNode == nil || nameNode.Content(src) != methodName {
			continue
		}
		mods := childByType(child, "modifiers")
		if mods == nil {
			return false
		}
		for j := 0; j < int(mods.ChildCount()); j++ {
			if mods.Child(j).Content(src) == "static" {
				return true
			}
		}
	}
	return false
}

func hasPublicModifier(n *sitter.Node, src []byte) bool {
	mods := childByType(n, "modifiers")
	if mods == nil {
		return false
	}
	for i := 0; i < int(mods.ChildCount()); i++ {
		if mods.Child(i).Content(src) == "public" {
			return true
		}
	}
	return false
}

func extractFormalParams(params *sitter.Node, src []byte) []domain.ParamInfo {
	if params == nil {
		return nil
	}
	var out []domain.ParamInfo
	for i := 0; i < int(params.ChildCount()); i++ {
		child := params.Child(i)
		if child.Type() != "formal_parameter" {
			continue
		}
		// formal_parameter: <type> <identifier>
		var typeName, paramName string
		for j := 0; j < int(child.ChildCount()); j++ {
			gc := child.Child(j)
			switch gc.Type() {
			case "type_identifier", "integral_type", "floating_point_type",
				"boolean_type", "void_type", "generic_type":
				typeName = gc.Content(src)
			case "identifier":
				paramName = gc.Content(src)
			}
		}
		if paramName != "" {
			out = append(out, domain.ParamInfo{Name: paramName, TypeHint: typeName})
		}
	}
	return out
}

func childByType(n *sitter.Node, typ string) *sitter.Node {
	if n == nil {
		return nil
	}
	for i := 0; i < int(n.ChildCount()); i++ {
		if child := n.Child(i); child.Type() == typ {
			return child
		}
	}
	return nil
}
