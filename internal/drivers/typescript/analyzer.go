package typescript

import (
	"os"
	"path/filepath"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/typescript/tsx"
	"github.com/smacker/go-tree-sitter/typescript/typescript"

	"github.com/orieken/testsmith/internal/domain"
)

func analyzeFile(path string, ctx *domain.ProjectContext) (*domain.SourceAnalysis, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	lang := tsLanguage(path)
	parser := sitter.NewParser()
	parser.SetLanguage(lang)
	tree := parser.Parse(nil, src)
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

func tsLanguage(path string) *sitter.Language {
	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".tsx" {
		return tsx.GetLanguage()
	}
	return typescript.GetLanguage()
}

// extractImports walks the AST collecting all ESM imports and CJS require() calls.
func extractImports(root *sitter.Node, src []byte) []domain.ImportInfo {
	var imports []domain.ImportInfo

	var walk func(n *sitter.Node)
	walk = func(n *sitter.Node) {
		switch n.Type() {
		case "import_statement":
			if imp := parseImportStatement(n, src); imp != nil {
				imports = append(imports, *imp)
			}
		case "call_expression":
			if imp := parseRequireCall(n, src); imp != nil {
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

func parseImportStatement(n *sitter.Node, src []byte) *domain.ImportInfo {
	// The module specifier is the string node after the "from" keyword.
	for i := int(n.ChildCount()) - 1; i >= 0; i-- {
		child := n.Child(i)
		if child.Type() == "string" {
			// Extract the string_fragment child (the actual path without quotes).
			if frag := childByType(child, "string_fragment"); frag != nil {
				return &domain.ImportInfo{Module: frag.Content(src)}
			}
			return &domain.ImportInfo{Module: trimQuotes(child.Content(src))}
		}
	}
	return nil
}

func parseRequireCall(n *sitter.Node, src []byte) *domain.ImportInfo {
	// require("module") — callee is identifier "require", first arg is string.
	callee := childByType(n, "identifier")
	if callee == nil || callee.Content(src) != "require" {
		return nil
	}
	args := childByType(n, "arguments")
	if args == nil {
		return nil
	}
	for i := 0; i < int(args.ChildCount()); i++ {
		child := args.Child(i)
		if child.Type() == "string" {
			if frag := childByType(child, "string_fragment"); frag != nil {
				return &domain.ImportInfo{Module: frag.Content(src)}
			}
			return &domain.ImportInfo{Module: trimQuotes(child.Content(src))}
		}
	}
	return nil
}

// extractPublicAPI walks the AST collecting all top-level exported declarations.
func extractPublicAPI(root *sitter.Node, src []byte) []domain.PublicMember {
	var members []domain.PublicMember

	for i := 0; i < int(root.ChildCount()); i++ {
		child := root.Child(i)
		if child.Type() != "export_statement" {
			continue
		}
		members = append(members, parseExportStatement(child, src)...)
	}
	return members
}

func parseExportStatement(n *sitter.Node, src []byte) []domain.PublicMember {
	for i := 0; i < int(n.ChildCount()); i++ {
		child := n.Child(i)
		switch child.Type() {
		case "function_declaration", "generator_function_declaration":
			if m := parseFunctionDecl(child, src); m != nil {
				return []domain.PublicMember{*m}
			}
		case "class_declaration":
			if m := parseClassDecl(child, src); m != nil {
				return []domain.PublicMember{*m}
			}
		case "interface_declaration":
			if m := parseNamedDecl(child, src, domain.KindClass, "type_identifier"); m != nil {
				return []domain.PublicMember{*m}
			}
		case "type_alias_declaration":
			if m := parseNamedDecl(child, src, domain.KindClass, "type_identifier"); m != nil {
				return []domain.PublicMember{*m}
			}
		case "enum_declaration":
			if m := parseNamedDecl(child, src, domain.KindClass, "identifier"); m != nil {
				return []domain.PublicMember{*m}
			}
		case "lexical_declaration":
			return parseLexicalDeclaration(child, src)
		}
	}
	return nil
}

func parseFunctionDecl(n *sitter.Node, src []byte) *domain.PublicMember {
	nameNode := childByType(n, "identifier")
	if nameNode == nil {
		return nil
	}
	name := nameNode.Content(src)
	if !isPublicName(name) {
		return nil
	}
	params := extractFormalParams(childByType(n, "formal_parameters"), src)
	return &domain.PublicMember{
		Name:   name,
		Kind:   domain.KindFunction,
		Parameters: params,
	}
}

func parseClassDecl(n *sitter.Node, src []byte) *domain.PublicMember {
	nameNode := childByType(n, "type_identifier")
	if nameNode == nil {
		nameNode = childByType(n, "identifier")
	}
	if nameNode == nil {
		return nil
	}
	name := nameNode.Content(src)
	body := childByType(n, "class_body")
	methods := extractMethods(body, src)
	return &domain.PublicMember{
		Name:    name,
		Kind:    domain.KindClass,
		Methods: methods,
	}
}

func parseNamedDecl(n *sitter.Node, src []byte, kind domain.MemberKind, nameType string) *domain.PublicMember {
	nameNode := childByType(n, nameType)
	if nameNode == nil {
		return nil
	}
	return &domain.PublicMember{Name: nameNode.Content(src), Kind: kind}
}

func parseLexicalDeclaration(n *sitter.Node, src []byte) []domain.PublicMember {
	var members []domain.PublicMember
	for i := 0; i < int(n.ChildCount()); i++ {
		child := n.Child(i)
		if child.Type() != "variable_declarator" {
			continue
		}
		nameNode := child.Child(0)
		if nameNode == nil {
			continue
		}
		name := nameNode.Content(src)
		if !isPublicName(name) {
			continue
		}
		// Only emit if the value is an arrow function.
		for j := 0; j < int(child.ChildCount()); j++ {
			gc := child.Child(j)
			if gc.Type() == "arrow_function" {
				params := extractFormalParams(childByType(gc, "formal_parameters"), src)
				members = append(members, domain.PublicMember{
					Name:   name,
					Kind:   domain.KindFunction,
					Parameters: params,
				})
				break
			}
		}
	}
	return members
}

func extractMethods(body *sitter.Node, src []byte) []domain.MethodInfo {
	if body == nil {
		return nil
	}
	var methods []domain.MethodInfo
	for i := 0; i < int(body.ChildCount()); i++ {
		child := body.Child(i)
		if child.Type() != "method_definition" {
			continue
		}
		// Method name is a property_identifier node (distinct from identifier).
		nameNode := childByType(child, "property_identifier")
		if nameNode == nil {
			continue
		}
		name := nameNode.Content(src)
		// Skip constructor and private (#name) methods.
		if name == "constructor" || strings.HasPrefix(name, "#") {
			continue
		}
		isPublic := !strings.HasPrefix(name, "_") && !strings.HasPrefix(name, "#")
		// Check for TypeScript access modifiers.
		isPublic = isPublic && !hasModifier(child, src, "private") && !hasModifier(child, src, "protected")

		params := extractFormalParams(childByType(child, "formal_parameters"), src)
		methods = append(methods, domain.MethodInfo{
			Name:       name,
			Parameters: params,
			IsPublic:   isPublic,
		})
	}
	return methods
}

func hasModifier(n *sitter.Node, src []byte, modifier string) bool {
	for i := 0; i < int(n.ChildCount()); i++ {
		child := n.Child(i)
		if child.Type() == "accessibility_modifier" && child.Content(src) == modifier {
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
		switch child.Type() {
		case "identifier":
			out = append(out, domain.ParamInfo{Name: child.Content(src)})
		case "required_parameter", "optional_parameter":
			// TypeScript typed params: name: Type
			nameNode := child.Child(0)
			if nameNode == nil {
				continue
			}
			name := nameNode.Content(src)
			typeHint := ""
			for j := 0; j < int(child.ChildCount()); j++ {
				gc := child.Child(j)
				if gc.Type() == "type_annotation" {
					typeHint = strings.TrimPrefix(gc.Content(src), ": ")
				}
			}
			out = append(out, domain.ParamInfo{Name: name, TypeHint: typeHint})
		case "rest_pattern":
			nameNode := childByType(child, "identifier")
			if nameNode != nil {
				out = append(out, domain.ParamInfo{Name: "..." + nameNode.Content(src)})
			}
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

func trimQuotes(s string) string {
	s = strings.TrimPrefix(s, `"`)
	s = strings.TrimSuffix(s, `"`)
	s = strings.TrimPrefix(s, `'`)
	s = strings.TrimSuffix(s, `'`)
	s = strings.TrimPrefix(s, "`")
	s = strings.TrimSuffix(s, "`")
	return s
}

func isPublicName(name string) bool {
	return name != "" && !strings.HasPrefix(name, "_") && !strings.HasPrefix(name, "#")
}
