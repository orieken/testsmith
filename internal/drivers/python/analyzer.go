package python

import (
	"os"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/python"

	"github.com/orieken/testsmith/internal/domain"
)

// analyzeFile parses a Python source file and returns its full SourceAnalysis.
func analyzeFile(path string, ctx *domain.ProjectContext) (*domain.SourceAnalysis, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	parser := sitter.NewParser()
	parser.SetLanguage(python.GetLanguage())
	tree := parser.Parse(nil, src)
	root := tree.RootNode()

	if root.HasError() {
		// Non-fatal: log the error but continue with partial results.
		// A syntax error in one node shouldn't abort the whole analysis.
	}

	imports := extractImports(root, src)
	publicAPI := extractPublicAPI(root, src)

	// Classify each import using the driver's classifier.
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

// extractImports walks the AST and returns all ImportInfo values.
func extractImports(root *sitter.Node, src []byte) []domain.ImportInfo {
	var imports []domain.ImportInfo

	var walk func(n *sitter.Node)
	walk = func(n *sitter.Node) {
		switch n.Type() {
		case "import_statement":
			imports = append(imports, parseImportStatement(n, src)...)
		case "import_from_statement":
			imp := parseImportFromStatement(n, src)
			if imp != nil {
				imports = append(imports, *imp)
			}
		}
		for i := 0; i < int(n.ChildCount()); i++ {
			walk(n.Child(i))
		}
	}
	walk(root)
	return imports
}

// parseImportStatement handles: import os  /  import os as o  /  import os, sys
func parseImportStatement(n *sitter.Node, src []byte) []domain.ImportInfo {
	var out []domain.ImportInfo
	for i := 0; i < int(n.ChildCount()); i++ {
		child := n.Child(i)
		switch child.Type() {
		case "dotted_name":
			out = append(out, domain.ImportInfo{
				Module:     child.Content(src),
				IsFrom:     false,
				LineNumber: int(n.StartPoint().Row) + 1,
			})
		case "aliased_import":
			module := ""
			alias := ""
			for j := 0; j < int(child.ChildCount()); j++ {
				gc := child.Child(j)
				switch gc.Type() {
				case "dotted_name":
					module = gc.Content(src)
				case "identifier":
					alias = gc.Content(src)
				}
			}
			if module != "" {
				out = append(out, domain.ImportInfo{
					Module:     module,
					Alias:      alias,
					IsFrom:     false,
					LineNumber: int(n.StartPoint().Row) + 1,
				})
			}
		}
	}
	return out
}

// parseImportFromStatement handles: from os import path  /  from . import foo  /  from os import *
func parseImportFromStatement(n *sitter.Node, src []byte) *domain.ImportInfo {
	imp := &domain.ImportInfo{
		IsFrom:     true,
		LineNumber: int(n.StartPoint().Row) + 1,
	}

	for i := 0; i < int(n.ChildCount()); i++ {
		child := n.Child(i)
		switch child.Type() {
		case "dotted_name":
			imp.Module = child.Content(src)
		case "relative_import":
			// e.g. ".utils" or "..models"
			imp.Module = child.Content(src)
		case "import_from_names":
			imp.Names = extractImportNames(child, src)
		case "wildcard_import":
			imp.Names = []string{"*"}
		}
	}

	if imp.Module == "" && len(imp.Names) == 0 {
		return nil
	}
	return imp
}

func extractImportNames(n *sitter.Node, src []byte) []string {
	var names []string
	for i := 0; i < int(n.ChildCount()); i++ {
		child := n.Child(i)
		switch child.Type() {
		case "identifier":
			names = append(names, child.Content(src))
		case "aliased_import":
			// from x import y as z — capture "y"
			for j := 0; j < int(child.ChildCount()); j++ {
				gc := child.Child(j)
				if gc.Type() == "identifier" {
					names = append(names, gc.Content(src))
					break
				}
			}
		}
	}
	return names
}

// extractPublicAPI walks the AST for top-level public functions and classes.
func extractPublicAPI(root *sitter.Node, src []byte) []domain.PublicMember {
	var members []domain.PublicMember

	for i := 0; i < int(root.ChildCount()); i++ {
		child := root.Child(i)
		switch child.Type() {
		case "function_definition", "async_function_def":
			if m := parseFunctionDef(child, src); m != nil {
				members = append(members, *m)
			}
		case "class_definition":
			if m := parseClassDef(child, src); m != nil {
				members = append(members, *m)
			}
		case "decorated_definition":
			// unwrap @decorator\ndef foo or @decorator\nclass Bar
			for j := 0; j < int(child.ChildCount()); j++ {
				inner := child.Child(j)
				switch inner.Type() {
				case "function_definition", "async_function_def":
					if m := parseFunctionDef(inner, src); m != nil {
						members = append(members, *m)
					}
				case "class_definition":
					if m := parseClassDef(inner, src); m != nil {
						members = append(members, *m)
					}
				}
			}
		}
	}
	return members
}

func parseFunctionDef(n *sitter.Node, src []byte) *domain.PublicMember {
	name := childByType(n, "identifier")
	if name == nil {
		return nil
	}
	nameStr := name.Content(src)
	if strings.HasPrefix(nameStr, "_") {
		return nil // private
	}

	params := extractParams(childByType(n, "parameters"), src)
	docstring := extractDocstring(childByType(n, "block"), src)

	return &domain.PublicMember{
		Name:       nameStr,
		Kind:       domain.KindFunction,
		Parameters: params,
		Docstring:  docstring,
	}
}

func parseClassDef(n *sitter.Node, src []byte) *domain.PublicMember {
	name := childByType(n, "identifier")
	if name == nil {
		return nil
	}
	nameStr := name.Content(src)
	if strings.HasPrefix(nameStr, "_") {
		return nil
	}

	body := childByType(n, "block")
	methods := extractMethods(body, src)
	docstring := extractDocstring(body, src)

	return &domain.PublicMember{
		Name:      nameStr,
		Kind:      domain.KindClass,
		Methods:   methods,
		Docstring: docstring,
	}
}

func extractMethods(block *sitter.Node, src []byte) []domain.MethodInfo {
	if block == nil {
		return nil
	}
	var methods []domain.MethodInfo
	for i := 0; i < int(block.ChildCount()); i++ {
		child := block.Child(i)
		var fn *sitter.Node
		switch child.Type() {
		case "function_definition", "async_function_def":
			fn = child
		case "decorated_definition":
			for j := 0; j < int(child.ChildCount()); j++ {
				if t := child.Child(j).Type(); t == "function_definition" || t == "async_function_def" {
					fn = child.Child(j)
					break
				}
			}
		}
		if fn == nil {
			continue
		}
		nameNode := childByType(fn, "identifier")
		if nameNode == nil {
			continue
		}
		methodName := nameNode.Content(src)
		// Skip dunder methods except __init__ (useful for constructor params).
		if strings.HasPrefix(methodName, "__") && methodName != "__init__" {
			continue
		}
		isPublic := !strings.HasPrefix(methodName, "_")
		params := extractParams(childByType(fn, "parameters"), src)
		methods = append(methods, domain.MethodInfo{
			Name:       methodName,
			Parameters: params,
			IsPublic:   isPublic,
		})
	}
	return methods
}

func extractParams(params *sitter.Node, src []byte) []domain.ParamInfo {
	if params == nil {
		return nil
	}
	var out []domain.ParamInfo
	for i := 0; i < int(params.ChildCount()); i++ {
		child := params.Child(i)
		switch child.Type() {
		case "identifier":
			name := child.Content(src)
			if name == "self" || name == "cls" {
				continue
			}
			out = append(out, domain.ParamInfo{Name: name})
		case "typed_parameter":
			// name: type
			nameNode := child.Child(0)
			if nameNode == nil {
				continue
			}
			name := nameNode.Content(src)
			if name == "self" || name == "cls" {
				continue
			}
			typeHint := ""
			for j := 0; j < int(child.ChildCount()); j++ {
				gc := child.Child(j)
				if gc.Type() == "type" {
					typeHint = gc.Content(src)
				}
			}
			out = append(out, domain.ParamInfo{Name: name, TypeHint: typeHint})
		case "default_parameter":
			nameNode := child.Child(0)
			if nameNode != nil {
				name := nameNode.Content(src)
				if name != "self" && name != "cls" {
					out = append(out, domain.ParamInfo{Name: name})
				}
			}
		}
	}
	return out
}

func extractDocstring(block *sitter.Node, src []byte) string {
	if block == nil {
		return ""
	}
	// First statement in a block that is a string literal is the docstring.
	for i := 0; i < int(block.ChildCount()); i++ {
		child := block.Child(i)
		if child.Type() == "expression_statement" {
			for j := 0; j < int(child.ChildCount()); j++ {
				gc := child.Child(j)
				if gc.Type() == "string" {
					raw := gc.Content(src)
					return strings.Trim(raw, `"'`)
				}
			}
		}
		break // only check first statement
	}
	return ""
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
