"""
Orchestrator for analyzing Python source files.
"""
import ast
from pathlib import Path
from testsmith.support.models import ImportInfo, AnalysisResult, ProjectContext
from testsmith.support.exceptions import SourceParseError
from testsmith.core.import_classifier import classify_all
from testsmith.core.module_inspector import inspect_module


def extract_imports(tree: ast.Module) -> list[ImportInfo]:
    """
    Extract import statements from the AST.
    Handles 'import x' and 'from x import y'.
    Walks all nodes to capture imports inside blocks (try/except, if).
    """
    imports = []
    
    for node in ast.walk(tree):
        if isinstance(node, ast.Import):
            for alias in node.names:
                imports.append(ImportInfo(
                    module=alias.name,
                    names=[],  # Regular import has no 'names' list usually, unless we want to track aliases separately
                    is_from=False,
                    alias=alias.asname,
                    line_number=node.lineno
                ))
        elif isinstance(node, ast.ImportFrom):
            if node.module is None and node.level > 0:
                # Relative import like 'from . import foo'
                # module name logic: '.' * level
                module_name = "." * node.level
            elif node.module:
                # 'from foo import bar' or 'from .foo import bar'
                if node.level > 0:
                    module_name = "." * node.level + node.module
                else:
                    module_name = node.module
            else:
                # Should not happen in valid python? 'from import x'? No.
                continue

            names = [alias.name for alias in node.names]
            
            # If there's an alias for a specific name?
            # ImportInfo model: `names` is list[str], `alias` is Optional[str].
            # If `from x import y as z`, we have one ImportFrom but potentially multiple names.
            # And `alias` field in ImportInfo handles the alias of the MODULE usually?
            # Or is it per name?
            # dataclass ImportInfo:
            # module: str
            # names: list[str]
            # is_from: bool
            # alias: Optional[str]
            # line_number: int
            
            # If we have `from x import y as z, a as b`, this is ONE ast.ImportFrom node.
            # But the alias applies to specific names. ImportInfo seems designed for one "statement" 
            # or maybe one "import source".
            # If we create one ImportInfo per statement:
            imports.append(ImportInfo(
                module=module_name,
                names=names,
                is_from=True,
                alias=None, # Aliases are inside names/asnames map really, but simplifying for now or if ImportInfo supports it better.
                # Actually, if the model has `alias`, it is likely for `import x as y`.
                # For `from x import y as z`, the alias is specific to `y`.
                # Given `names` is just a list of strings, we seemingly lose the alias info for `from` imports 
                # unless we change the model or interpretation.
                # Project requirement says: "Handle import x as y -> ImportInfo(..., alias='y')"
                # It doesn't explicitly specify `from x import y as z`.
                # We will stick to collecting names.
                line_number=node.lineno
            ))
            
    return imports


def analyze_file(source_path: Path, project_context: ProjectContext) -> AnalysisResult:
    """
    Analyze a source file to extract imports, public API, and classification.
    """
    try:
        source_code = source_path.read_text(encoding="utf-8")
    except FileNotFoundError:
        # Re-raise with message as per prompt
        raise FileNotFoundError(f"Source file not found: {source_path}")
        
    try:
        tree = ast.parse(source_code)
    except SyntaxError as e:
        raise SourceParseError(str(source_path), e.lineno or 0, str(e))
        
    # Extract raw imports
    raw_imports = extract_imports(tree)
    
    # Classify imports
    classified_imports = classify_all(raw_imports, project_context.package_map)
    
    # Inspect public API
    # We re-use the tree? inspect_module currently takes source_code and re-parses.
    # To be efficient we could refactor inspect_module to take tree, but prompt says:
    # "inspect_module(source_code: str) -> list[PublicMember]".
    # We will follow the interface.
    public_api = inspect_module(source_code, str(source_path))
    
    # Derive module name
    # e.g. src/testsmith/core/source_analyzer.py -> testsmith.core.source_analyzer ?
    # Or just filename stem? Prompt says: "Derive module_name from the filename (e.g., payment_processor.py -> payment_processor)"
    # This implies just the stem.
    module_name = source_path.stem
    
    return AnalysisResult(
        source_path=source_path,
        module_name=module_name,
        imports=classified_imports,
        public_api=public_api,
        project=project_context
    )
