"""
Classification of imports into stdlib, internal, or external.
"""
import sys
from pathlib import Path
from testsmith.support.models import ImportInfo, ClassifiedImports


def get_stdlib_modules() -> frozenset[str]:
    """
    Return a frozenset of standard library module names.
    """
    # PEP 635 - sys.stdlib_module_names (Python 3.10+)
    if sys.version_info >= (3, 10):
        modules = set(sys.stdlib_module_names)
    else:
        # Fallback for older python if needed, though we require 3.10+
        # minimal fallback
        modules = {"os", "sys", "pathlib", "json", "ast", "re", "math", "datetime", "typing"}
        
    # Add common submodules that might be imported directly but not in the root list
    # e.g. email.mime is not in stdlib_module_names, only email is.
    # But usually we check ROOT package.
    # "Include common stdlib sub-packages that might not be in the set" (from prompt)
    # The classification methodology uses `extract_root_package`.
    # `email.mime` -> `email`. `email` is in stdlib_names.
    # So we strictly speaking don't need submodules if we reliably check root packages.
    # Only if `extract_root_package` returns `email` and we check `email`.
    
    return frozenset(modules)


def extract_root_package(module_name: str) -> str:
    """
    Extract the root package name from a dotted module path.
    e.g. "stripe.checkout.Session" -> "stripe"
    """
    if not module_name:
        return ""
    return module_name.split(".")[0]


def classify_import(
    import_info: ImportInfo, 
    package_map: dict[str, Path], 
    stdlib_modules: frozenset[str]
) -> str:
    """
    Classify a single import as 'stdlib', 'internal', or 'external'.
    """
    # 1. Relative imports are always internal
    if import_info.module.startswith("."):
        return "internal"
        
    root_pkg = extract_root_package(import_info.module)
    
    # 2. Check stdlib
    if root_pkg in stdlib_modules:
        return "stdlib"
        
    # 3. Check project package map
    if root_pkg in package_map:
        return "internal"
        
    # 4. Default to external
    return "external"


def classify_all(imports: list[ImportInfo], package_map: dict[str, Path]) -> ClassifiedImports:
    """
    Classify a list of imports.
    """
    stdlib_modules = get_stdlib_modules()
    classified = ClassifiedImports()
    
    for imp in imports:
        category = classify_import(imp, package_map, stdlib_modules)
        
        if category == "stdlib":
            classified.stdlib.append(imp)
        elif category == "internal":
            classified.internal.append(imp)
        else:
            classified.external.append(imp)
            
    return classified
