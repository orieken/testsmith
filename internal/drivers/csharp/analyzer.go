package csharp

import (
	"context"
	"fmt"
	"os"

	sitter "github.com/smacker/go-tree-sitter"
	sittercsharp "github.com/smacker/go-tree-sitter/csharp"

	"github.com/orieken/assay/internal/domain"
)

func analyzeFile(path string, ctx *domain.ProjectContext) (*domain.SourceAnalysis, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	parser := sitter.NewParser()
	parser.SetLanguage(sittercsharp.GetLanguage())
	tree, err := parser.ParseCtx(context.Background(), nil, src)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	root := tree.RootNode()

	imports := extractUsings(root, src)
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

// extractUsings collects all `using` directives from the compilation unit.
func extractUsings(root *sitter.Node, src []byte) []domain.ImportInfo {
	var imports []domain.ImportInfo
	for i := 0; i < int(root.ChildCount()); i++ {
		child := root.Child(i)
		if child.Type() != "using_directive" {
			continue
		}
		// using_directive children: "using" keyword, then identifier or qualified_name, then ";"
		for j := 0; j < int(child.ChildCount()); j++ {
			gc := child.Child(j)
			switch gc.Type() {
			case "identifier", "qualified_name":
				imports = append(imports, domain.ImportInfo{Module: gc.Content(src)})
			}
		}
	}
	return imports
}

// extractPublicAPI collects public class declarations from the compilation unit
// and from inside namespace declarations.
func extractPublicAPI(root *sitter.Node, src []byte) []domain.PublicMember {
	var members []domain.PublicMember
	var walk func(n *sitter.Node)
	walk = func(n *sitter.Node) {
		switch n.Type() {
		case "class_declaration":
			if isPublicNode(n, src) {
				members = append(members, parseClass(n, src)...)
			}
			return // don't recurse into class body for nested classes
		case "namespace_declaration", "file_scoped_namespace_declaration":
			// recurse into namespace body
			for i := 0; i < int(n.ChildCount()); i++ {
				walk(n.Child(i))
			}
			return
		}
		for i := 0; i < int(n.ChildCount()); i++ {
			walk(n.Child(i))
		}
	}
	walk(root)
	return members
}

func parseClass(n *sitter.Node, src []byte) []domain.PublicMember {
	nameNode := childByType(n, "identifier")
	if nameNode == nil {
		return nil
	}
	className := nameNode.Content(src)

	body := childByType(n, "declaration_list")
	if body == nil {
		return nil
	}

	methods := extractMethods(body, src)
	return []domain.PublicMember{
		{Name: className, Kind: domain.KindClass, Methods: methods},
	}
}

func extractMethods(body *sitter.Node, src []byte) []domain.MethodInfo {
	var methods []domain.MethodInfo
	for i := 0; i < int(body.ChildCount()); i++ {
		child := body.Child(i)
		if child.Type() != "method_declaration" {
			continue
		}
		if !isPublicNode(child, src) {
			continue
		}
		nameNode := childByType(child, "identifier")
		if nameNode == nil {
			continue
		}
		params := extractParams(childByType(child, "parameter_list"), src)
		methods = append(methods, domain.MethodInfo{
			Name:       nameNode.Content(src),
			Parameters: params,
			IsPublic:   true,
		})
	}
	return methods
}

func isPublicNode(n *sitter.Node, src []byte) bool {
	for i := 0; i < int(n.ChildCount()); i++ {
		child := n.Child(i)
		if child.Type() == "modifier" && child.Content(src) == "public" {
			return true
		}
	}
	return false
}

func extractParams(paramList *sitter.Node, src []byte) []domain.ParamInfo {
	if paramList == nil {
		return nil
	}
	var out []domain.ParamInfo
	for i := 0; i < int(paramList.ChildCount()); i++ {
		child := paramList.Child(i)
		if child.Type() != "parameter" {
			continue
		}
		// parameter: <type_node> <identifier>
		var typeName, paramName string
		for j := 0; j < int(child.ChildCount()); j++ {
			gc := child.Child(j)
			switch gc.Type() {
			case "identifier":
				// Last identifier is the param name; earlier one may be type.
				if paramName == "" {
					paramName = gc.Content(src)
				} else {
					typeName = paramName
					paramName = gc.Content(src)
				}
			case "predefined_type", "generic_name", "nullable_type",
				"array_type", "qualified_name":
				typeName = gc.Content(src)
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
