"""
Project structure detection.
"""
import ast
from pathlib import Path
from testsmith.support.exceptions import ProjectRootNotFoundError
from testsmith.support.config import TestSmithConfig
from testsmith.support.models import ProjectContext


def find_project_root(start_path: Path) -> Path:
    """
    Find the project root by walking up looking for markers.
    Priority: pyproject.toml > setup.py > setup.cfg > .git > conftest.py
    """
    current = start_path.resolve()
    if current.is_file():
        current = current.parent

    # Markers in priority order
    markers = ["pyproject.toml", "setup.py", "setup.cfg", ".git", "conftest.py"]
    
    # We walk up to root
    for parent in [current] + list(current.parents):
        for marker in markers:
            if (parent / marker).exists():
                return parent
    
    raise ProjectRootNotFoundError(
        f"Could not find project root starting from {start_path}. "
        "Markers searched: " + ", ".join(markers)
    )


def scan_packages(root: Path, exclude_dirs: list[str]) -> dict[str, Path]:
    """
    Recursively scan from root for Python packages and return {name: absolute_path}.
    """
    package_map = {}
    exclude_set = set(exclude_dirs)
    
    # Check for src/ layout first? 
    # The prompt says: "Handle src/ layout: if a src/ directory exists containing packages, scan inside it"
    # But usually we scan the whole tree. If we scan root, we find src/foo as foo?
    # Yes, typically `src/foo` means package `foo`.
    # `root/foo` means package `foo`.
    # We need to detect "roots of packages".
    
    # Implementation strategy: walk top-down. 
    # If a directory has __init__.py, it's a package.
    # If a directory does NOT have __init__.py but has subdirectories that ARE packages (namespace packages), 
    # we might need to handle that. Prompt says: "Handle namespace packages (no __init__.py) by checking directory existence"
    # Actually, the prompt says "Handle namespace packages... by checking directory existence" in the context of 
    # `import_classifier` checking the map. 
    # Here in `scan_packages`, we need to populate the map.
    
    # If we find `src/`, we should probably look inside it primarily?
    # Or just scan everything and ignore non-package structure.
    
    search_dirs = [root]
    if (root / "src").is_dir():
        search_dirs.append(root / "src")
        
    # We'll walk using os.walk logic or pathlib rglob?
    # os.walk is safer for exclusions.
    
    import os
    
    for search_root in search_dirs:
        for dirpath, dirnames, filenames in os.walk(search_root):
            # Modify dirnames in-place to skip excludes
            dirnames[:] = [d for d in dirnames if d not in exclude_set and not d.startswith(".")]
            
            path = Path(dirpath)
            
            # Is this a package?
            # 1. Has __init__.py -> YES
            if (path / "__init__.py").exists():
                # Determine package name relative to search_root?
                # If search_root is project/src, and path is project/src/foo, name is foo.
                # If search_root is project, and path is project/foo, name is foo.
                # If path is project/src/foo, and we walked from project, name is src.foo? No.
                # If src layout is used, packages inside src are top-level.
                
                # Logic:
                # If we are inside `src`, the relative path from `src` is the package name.
                # If we are NOT inside `src`, relative path from `root` is package name.
                
                if (root / "src").exists() and (root / "src") in path.parents:
                    rel_path = path.relative_to(root / "src")
                elif (root / "src").exists() and path == (root / "src"):
                    # src folder itself is not a package usually
                    continue
                else:
                    rel_path = path.relative_to(root)
                    if str(rel_path) == ".":
                        continue
                
                # Convert path to dotted name? 
                # Wait, generic import mapping usually wants Root Package Name -> Path.
                # Prompt says: "Build mapping: {root_package_name: absolute_path}"
                # e.g. {"components": Path("/project/src/components")}
                # So we only want ROOT packages?
                # "Recursively scan... Build mapping {root_package_name: absolute_path}"
                # If we have `foo.bar`, do we map `foo` or `foo.bar`?
                # Usually ImportClassifier needs `extract_root_package`.
                # So we mostly care about top-level packages.
                # But we should probably map everything just in case, or at least top-level ones.
                # "Recursively scan from root for Python packages... Build mapping {root_package_name...}"
                # It implies we iterate everything but the key is "root_package_name".
                # If we find `src/foo/bar`, `foo` is the root package.
                # We should map `foo` -> `src/foo`.
                # `foo.bar` is part of `foo`.
                
                first_part = rel_path.parts[0]
                # If we haven't seen this root package yet, add it.
                if first_part not in package_map:
                    # We need to trace back to the actual root directory for this package.
                    # If rel_path is foo/bar/baz, root pkg is foo at .../foo
                    # Calculate path to 'foo'
                    if (root / "src").exists() and (root / "src") in path.parents:
                         pkg_root = root / "src" / first_part
                    else:
                         pkg_root = root / first_part
                    
                    package_map[first_part] = pkg_root
            
            # Handle standalone modules? "and standalone .py modules"
            for f in filenames:
                if f.endswith(".py") and f != "__init__.py" and f != "conftest.py" and not f.startswith("test_") and f != "setup.py":
                    # Standalone module `utils.py` -> package `utils`
                    # If inside a package, it's a submodule, we ignore (covered by root package).
                    # If at top level (or inside src at top level), it's a root module.
                    
                     file_path = path / f
                     
                     # Check if it's already inside a known package?
                     # If `path` has __init__.py, we already handled the package.
                     if (path / "__init__.py").exists():
                         continue

                     # It's a standalone file in a folder without init.
                     # Treat as root module?
                     if (root / "src").exists() and (root / "src") in path.parents:
                        rel_path = file_path.relative_to(root / "src")
                     elif (root / "src").exists() and path == (root / "src"):
                        rel_path = file_path.relative_to(root / "src")
                     else:
                        rel_path = file_path.relative_to(root)
                        
                     # If it's just `utils.py`, name is `utils`.
                     # If it's `scripts/do_thing.py` (no init in scripts), name is `scripts.do_thing`? 
                     # Or just top level `utils`?
                     # Python treats it as module `utils`.
                     # If inside a folder without init, it's not importable as a package usually, 
                     # unless namespace package. 
                     # Simple logic: map `filename_stem` to file_path if it's at conceptual root.
                     
                     # We only care about root imports.
                     # If I import `utils`, and `utils.py` exists at src root, that matches.
                     root_name = rel_path.parts[0]
                     if root_name.endswith(".py"):
                         root_name = root_name[:-3]
                         if root_name not in package_map:
                             package_map[root_name] = file_path
                     else:
                         # It's in a subfolder `folder/script.py`. 
                         # If `folder` is namespace package, `folder` is root.
                         if root_name not in package_map:
                             # Map the directory as the root of namespace package?
                             if (root / "src").exists() and (root / "src") in path.parents:
                                package_map[root_name] = root / "src" / root_name
                             else:
                                package_map[root_name] = root / root_name
                                
    return package_map


def detect_conftest(root: Path) -> tuple[Path | None, list[str]]:
    """
    Look for conftest.py at project root and parse paths_to_add.
    """
    conftest = root / "conftest.py"
    if not conftest.exists():
        return None, []
    
    try:
        source = conftest.read_text(encoding="utf-8")
        tree = ast.parse(source)
        
        # Look for assignment: paths_to_add = [...]
        for node in ast.walk(tree):
            if isinstance(node, ast.Assign):
                # Check targets
                for target in node.targets:
                    if isinstance(target, ast.Name) and target.id == "paths_to_add":
                        # Check value is a list
                        if isinstance(node.value, ast.List):
                            paths = []
                            for elt in node.value.elts:
                                if isinstance(elt, ast.Constant) and isinstance(elt.value, str):
                                    paths.append(elt.value)
                            return conftest, paths
    except Exception:
        # If parse fails, ignore
        pass
        
    return conftest, []


def build_project_context(source_path: Path, config: TestSmithConfig) -> ProjectContext:
    """
    Build the full project context.
    """
    abs_source = source_path.resolve()
    
    try:
        root = find_project_root(abs_source)
    except ProjectRootNotFoundError:
        # Fallback if source is loose file? Or just fail?
        # Requires config to potentially hint root, but here we just pass config.
        # If we can't find root, we can't build context.
        raise

    # Verify source is inside root
    # Note: behave nicely if source is symlinked or funky, strictly `is_relative_to`
    try:
        abs_source.relative_to(root)
    except ValueError:
       # If source is outside project root, we warn? 
       # Or maybe we found a nested root (e.g. root/tests/conftest.py) but source is root/src/app.py?
       # `find_project_root` walks UP from source. So `root` must contain `source`.
       pass

    package_map = scan_packages(root, config.exclude_dirs)
    conftest_path, existing_paths = detect_conftest(root)
    
    return ProjectContext(
        root=root,
        package_map=package_map,
        conftest_path=conftest_path,
        existing_paths=existing_paths
    )
