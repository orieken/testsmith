"""
Inspects Python modules to extract their public API.
"""
import ast
from testsmith.support.models import PublicMember
from testsmith.support.exceptions import SourceParseError


def extract_public_functions(tree: ast.Module) -> list[PublicMember]:
    """
    Extract public top-level functions from the AST.
    """
    members = []
    
    for node in tree.body:
        if isinstance(node, (ast.FunctionDef, ast.AsyncFunctionDef)):
            if node.name.startswith("_"):
                continue
            
            # Extract parameters
            params = []
            for arg in node.args.args:
                # Typically skip self/cls for methods, but these are top-level functions
                # Top-level should keep all args usually? 
                # Prompt says: "Extract parameter names from ast.arguments (skip self, cls)"
                # Just in case someone names a top-level func arg 'self', we skip it.
                if arg.arg not in ("self", "cls"):
                    params.append(arg.arg)
            
            # Extract docstring
            docstring = ast.get_docstring(node)
            
            members.append(PublicMember(
                name=node.name,
                kind="function",
                parameters=params,
                methods=[],
                docstring=docstring
            ))
            
    return members


def extract_public_classes(tree: ast.Module) -> list[PublicMember]:
    """
    Extract public top-level classes and their public methods.
    """
    members = []
    
    for node in tree.body:
        if isinstance(node, ast.ClassDef):
            if node.name.startswith("_"):
                continue
                
            public_methods = []
            init_params = []
            
            for item in node.body:
                if isinstance(item, (ast.FunctionDef, ast.AsyncFunctionDef)):
                    method_name = item.name
                    # Include __init__, skip other _ prefixed methods
                    if method_name.startswith("_") and method_name != "__init__":
                        continue
                    
                    # Extract params for method
                    m_params = []
                    for arg in item.args.args:
                        if arg.arg not in ("self", "cls"):
                            m_params.append(arg.arg)
                    
                    if method_name == "__init__":
                        # Class parameters are init parameters
                        init_params = m_params
                    else:
                        public_methods.append(method_name)
            
            # Extract class docstring
            docstring = ast.get_docstring(node)
            
            members.append(PublicMember(
                name=node.name,
                kind="class",
                parameters=init_params,
                methods=public_methods,
                docstring=docstring
            ))
            
    return members


def inspect_module(source_code: str, file_path: str = "<unknown>") -> list[PublicMember]:
    """
    Parse source code and extract public API.
    """
    try:
        tree = ast.parse(source_code)
    except SyntaxError as e:
        raise SourceParseError(file_path, e.lineno or 0, str(e))
        
    classes = extract_public_classes(tree)
    functions = extract_public_functions(tree)
    
    return classes + functions
