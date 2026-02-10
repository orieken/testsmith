"""
Generates and updates shared pytest fixture files.
"""
import ast
import re
from pathlib import Path
from testsmith.support.config import TestSmithConfig
from testsmith.support.templates import render_fixture_file
from testsmith.support.file_operations import safe_write


def derive_fixture_name(dependency_name: str) -> str:
    """
    Convert dependency name to fixture function name.
    e.g. "stripe" -> "mock_stripe"
    e.g. "my-lib" -> "mock_my_lib"
    """
    safe_name = dependency_name.replace("-", "_").replace(".", "_")
    return f"mock_{safe_name}"


def derive_fixture_filename(dependency_name: str, config: TestSmithConfig) -> Path:
    """
    Generate the expected file path for a dependency fixture.
    Ensures filename is valid python module (replaces - and . with _).
    """
    safe_name = dependency_name.replace("-", "_").replace(".", "_")
    # Prompt example was 'stripe.fixture.py', but dots are bad for imports.
    # We use '_fixture.py' pattern for safety if suffix allows, or just verify valid identifier.
    
    # If suffix starts with '.', we might create invalid module name if we prepend name.
    # e.g. stripe.fixture.py
    # We will enforce underscore separator if suffix doesn't have one, or sanitize.
    # For this implementation, we force `name_fixture.py` style for compatibility.
    filename = f"{safe_name}_fixture.py"
    return Path(config.fixture_dir) / filename


def parse_existing_fixture(fixture_path: Path) -> dict:
    """
    Parse an existing fixture file to find what's already mocked.
    Returns {"sub_modules": set[str], "mock_assignments": set[str]}
    """
    result = {"sub_modules": set(), "mock_assignments": set()}
    
    if not fixture_path.exists():
        return result
        
    try:
        source = fixture_path.read_text(encoding="utf-8")
        tree = ast.parse(source)
    except Exception:
        return result
        
    for node in ast.walk(tree):
        # Look for mocker.patch.dict("sys.modules", {...})
        if isinstance(node, ast.Call):
            if isinstance(node.func, ast.Attribute) and node.func.attr == "dict":
                if node.args and isinstance(node.args[0], ast.Constant) and node.args[0].value == "sys.modules":
                    if len(node.args) > 1 and isinstance(node.args[1], ast.Dict):
                        datadict = node.args[1]
                        for key in datadict.keys:
                            if isinstance(key, ast.Constant) and isinstance(key.value, str):
                                result["sub_modules"].add(key.value)
    return result


def generate_fixture(
    dependency_name: str, 
    sub_modules: list[str], 
    imported_names: dict[str, list[str]], 
    config: TestSmithConfig
) -> str:
    """
    Generate content for a shared fixture.
    """
    # derive fixture name is handled inside render_fixture_file or we need to align.
    # templates.render_fixture_file takes (dependency_name, sub_modules, imported_names)
    # and calculates fixture name itself.
    return render_fixture_file(dependency_name, sub_modules, imported_names)


def generate_or_update_fixture(
    dependency_name: str, 
    sub_modules: list[str], 
    imported_names: dict[str, list[str]], 
    project_root: Path,
    config: TestSmithConfig
) -> tuple[Path, str]:
    """
    Create or update a shared fixture file.
    Returns (path, action) where action is 'created', 'updated', or 'skipped'.
    """
    # Resolve absolute path
    rel_path = derive_fixture_filename(dependency_name, config)
    path = project_root / rel_path
    
    if not path.exists():
        content = generate_fixture(dependency_name, sub_modules, imported_names, config)
        if safe_write(path, content):
            return path, "created"
        return path, "skipped"
        
    # Update existing
    existing = parse_existing_fixture(path)
    existing_modules = existing["sub_modules"]
    
    new_modules = [m for m in sub_modules if m not in existing_modules]
    
    if not new_modules:
        return path, "skipped"
        
    # Attempt to update file using regex injection
    try:
        src = path.read_text(encoding="utf-8")
        # Find the sys.modules dict start
        # Look for `mocker.patch.dict("sys.modules", {`
        pattern = r'(mocker\.patch\.dict\s*\(\s*["\']sys\.modules["\']\s*,\s*\{)'
        match = re.search(pattern, src)
        if match:
            # We found the start. We want to insert new keys after the opening brace.
            insertion_point = match.end()
            
            # Construct new entries
            # default indentation 8 spaces? 
            # We assume standard formatting or try to detect?
            # Inserting: `\n        "module": mocker.Mock(),`
            
            lines_to_add = []
            for mod in new_modules:
                lines_to_add.append(f'\n        "{mod}": mocker.Mock(),')
                
            new_content = src[:insertion_point] + "".join(lines_to_add) + src[insertion_point:]
            safe_write(path, new_content, overwrite=True)
            return path, "updated"
            
    except Exception:
        # Fallback if regex fails (e.g. formatting differs wildy)
        pass
        
    return path, "skipped"


def generate_fixtures_conftest(fixture_dir: Path, fixture_files: list[Path]) -> str:
    """
    Generate conftest.py content importing all fixtures.
    """
    lines = ['"""Auto-generated conftest for fixtures."""', ""]
    
    for f in fixture_files:
        # file: stripe_fixture.py
        # module: stripe_fixture
        # function: mock_stripe (naming convention)
        mod_name = f.stem
        # Heuristic: name is mock_{dep}
        # We can also parse the file to find the function name, safer.
        # But per requirements we use convention derived from filename/dep.
        # if file is stripe_fixture.py, dep is stripe?
        # derive_fixture_name("stripe") -> mock_stripe.
        
        # Reverse dependency name from filename?
        # stripe_fixture -> stripe
        if mod_name.endswith("_fixture"):
            dep_name = mod_name[:-8] # remove _fixture
            func_name = derive_fixture_name(dep_name)
            lines.append(f"from .{mod_name} import {func_name}  # noqa: F401")
            
    return "\n".join(lines) + "\n"
