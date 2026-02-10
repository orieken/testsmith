"""
Updates conftest.py to include necessary paths for test discovery.
"""
from pathlib import Path
import ast
from testsmith.support.config import TestSmithConfig
from testsmith.support.templates import render_conftest_pytest_configure
from testsmith.support.file_operations import safe_write

def parse_paths_to_add(conftest_content: str, var_name: str) -> tuple[list[str], int, int]:
    """
    Parse conftest content to find existing paths_to_add list.
    Returns (existing_paths, start_line_index, end_line_index).
    Lines are 0-indexed for string manipulation.
    If not found, returns ([], -1, -1).
    """
    try:
        tree = ast.parse(conftest_content)
    except SyntaxError:
        return [], -1, -1
        
    for node in ast.walk(tree):
        if isinstance(node, ast.Assign):
            for target in node.targets:
                if isinstance(target, ast.Name) and target.id == var_name:
                    if isinstance(node.value, ast.List):
                        # Found the list assignment
                        # Extract string values
                        paths = []
                        for elt in node.value.elts:
                            if isinstance(elt, ast.Constant) and isinstance(elt.value, str):
                                paths.append(elt.value)
                        
                        # Determine exact lines for replacement
                        # node.lineno is 1-indexed start
                        # node.end_lineno is 1-indexed end
                        # We want the content inside the list if possible, or replace the whole list?
                        # Replacing the whole list is safer for formatting if we reconstruct it.
                        # But prompt says "Preserve ALL existing content... comments".
                        # If we replace `paths_to_add = [...]` block, comments *inside* might be lost 
                        # if we just dump new list.
                        # However, for V1 we assume standard format or append.
                        # If we just appending, we need the END of the list.
                        # `end_lineno` gives the closing bracket line usually.
                        
                        return paths, node.lineno - 1, node.end_lineno - 1
                        
    return [], -1, -1

def compute_required_paths(
    source_path: Path, 
    test_path: Path, 
    fixture_dir: Path, 
    project_root: Path
) -> list[str]:
    """
    Compute required paths relative to project root.
    Returns list of strings like 'src', 'tests/src', 'tests/fixtures'.
    """
    paths = set()
    
    # 1. Source parent (e.g. src or src/pkg)
    # If source is `src/pkg/mod.py`, we usually add `src` to path?
    # Or strict parent `src/pkg`?
    # Common pattern: add `src` to pythonpath.
    # If using `src` layout, add `src`.
    # logic: if path starts with `src`, add `src`.
    # simpler: add the parent directory of the file?
    # If we add `src/pkg`, then `import mod` works.
    # If we add `src`, then `import pkg.mod` works.
    # Our generated tests do `from pkg.mod import ...`.
    # So we need `src` in path.
    
    # Heuristic: Find the "root" of the package structure.
    # If `source_path` is `root/src/pkg/mod.py`:
    # relative: `src/pkg/mod.py`
    # top-level dir: `src`
    
    try:
        rel_source = source_path.relative_to(project_root)
        if len(rel_source.parts) > 1:
            top_level = rel_source.parts[0]
            paths.add(top_level) # e.g. "src" or "lib"
            # What if flat layout? `root/pkg/mod.py` -> `pkg` is top level?
            # If flat, usually we add current dir `.`? 
            # Or we add `.` to path?
            if top_level != "src":
                # For flat layout, we might need to add Project Root `.`?
                # Actually, standard python behavior adds CWD. 
                # Be specific: add the top folder if it's a package container?
                pass
    except ValueError:
        pass
        
    # 2. Test parent
    # `tests/src/pkg/test_mod.py`
    # We might need `tests` in path if valid? 
    # Usually we don't need tests in path unless importing helpers.
    # But prompt says "registers... test directories".
    # We'll add the test root `tests`.
    try:
        rel_test = test_path.relative_to(project_root)
        if len(rel_test.parts) > 0:
             # tests/src/.... -> "tests"
             paths.add(rel_test.parts[0])
    except ValueError:
        pass

    # 3. Fixtures dir
    # `tests/fixtures` - we definitely need this if we import from fixtures 
    # (though typically conftest discovery handles it, explicit imports need path)
    try:
        rel_fix = fixture_dir.relative_to(project_root)
        paths.add(str(rel_fix))
    except ValueError:
        pass
        
    return sorted(list(paths))

def diff_paths(existing: list[str], required: list[str]) -> list[str]:
    """Return paths in required that are not in existing."""
    # Normalize: strip trailing slashes, generic cleaning
    norm_existing = {p.rstrip("/") for p in existing}
    missing = []
    for p in required:
        norm_p = p.rstrip("/")
        # also handle "./src" vs "src"?
        if norm_p not in norm_existing:
            missing.append(p)
    return missing

def update_conftest(
    conftest_path: Path, 
    new_paths: list[str], 
    config: TestSmithConfig
) -> tuple[Path, str]:
    """
    Create or update conftest.py to include new paths.
    """
    if not new_paths:
         return conftest_path, "skipped"

    if not conftest_path.exists():
        # Create new
        # We need a template for "pytest_configure" and "sys.path" setup
        # `render_conftest_pytest_configure` does exactly this?
        # Check support.templates
        content = render_conftest_pytest_configure(new_paths)
        safe_write(conftest_path, content)
        return conftest_path, "created"
        
    # Update existing
    content = conftest_path.read_text(encoding="utf-8")
    var_name = "paths_to_add" # As per config default, hardcoded ok for now or use config.paths_to_add_var?
    # Config has `paths_to_add_var`
    var_name = config.paths_to_add_var
    
    existing_paths, start, end = parse_paths_to_add(content, var_name)
    
    missing = diff_paths(existing_paths, new_paths)
    if not missing:
        return conftest_path, "skipped"
        
    lines = content.splitlines()
    
    if start == -1:
        # Variable not found, append to end
        # We need to append the whole block:
        # paths_to_add = [...]
        # ... logic ...
        # But wait, `render_conftest_pytest_configure` generates the WHOLE function.
        # If conftest exists, it might have other fixtures.
        # we can't just append a function that might conflict (pytest_configure).
        # Prompt says: "If conftest exists but has no paths_to_add -> append the variable and pytest_configure code"
        # We will append the rendered block.
        
        block = render_conftest_pytest_configure(missing) # This generates variable AND logic
        # Wait, render_conftest_pytest_configure generates `paths_to_add = [...]` INSIDE the function?
        # Checking templates.py...
        # Yes: 
        # def pytest_configure(config):
        #    ...
        #    paths_to_add = [...]
        #    for p in paths_to_add: sys.path.append...
        
        # If we append this, and `pytest_configure` already exists, we break it (double def).
        # "If conftest exists... append...".
        # We assume if `paths_to_add` is missing, we can append the hook. 
        # (Risk: hook exists but implemented differently. V1 acceptable limitation).
        
        # Check if pytest_configure exists?
        if "def pytest_configure" in content:
            # Hard case: hook exists but no paths_to_add. 
            # We skip modification to avoid breaking user code in V1.
            return conftest_path, "skipped"
            
        lines.append("")
        lines.append(block)
        safe_write(conftest_path, "\n".join(lines), overwrite=True)
        return conftest_path, "updated"
        
    else:
        # Variable exists, insert into list
        # We found `paths_to_add = [...]`
        # We want to insert `    "new/path", # Added by TestSmith`
        # We insert before the closing bracket line (end).
        # Actually `end` is the line index of the closing bracket (usually).
        
        # Limitation: AST gives line range of the whole assignment.
        # `end_lineno` is the last line.
        # For `l = [a, b]`, end is line of `]`.
        
        insertion_idx = end 
        # Check if `lines[end]` contains `]`.
        while insertion_idx >= start:
            if "]" in lines[insertion_idx]:
                break
            insertion_idx -= 1
            
        # If we didn't find `]`, fall back (weird formatting)
        if insertion_idx < start:
             return conftest_path, "skipped"
             
        # Insert before the line with `]`?
        # Or if `]` is on separate line.
        # If `paths = ["a", "b"]`, inserting before `]` (same line) is hard with simple string split.
        # We need to handle inline list `[...]`.
        
        line_with_bracket = lines[insertion_idx]
        pre, post = line_with_bracket.rsplit("]", 1)
        
        # Construct insertion
        # If multiline list:
        # [
        #   "a",
        # ]
        # We insert at `insertion_idx`.
        
        # If single line: `l = ["a"]`
        # pre=`l = ["a"`, post=``
        # We want `l = ["a", "new"]`
        
        # Simplest specific approach:
        # Regex replacement of `]` with `, "new", ... ]`?
        
        # Let's try inserting new lines if it looks multiline (indentation detected).
        # Else line insertion.
        
        to_insert = []
        for p in missing:
            to_insert.append(f'"{p}", # Added by TestSmith')
            
        if "," in line_with_bracket:
             # Likely single line or compact
             # Join with commas
             joined = ", ".join(to_insert)
             # Add leading comma if needed
             if not pre.strip().endswith("[") and not pre.strip().endswith(","):
                  joined = ", " + joined
             
             new_line = f"{pre}{joined}]{post}"
             lines[insertion_idx] = new_line
        else:
             # Likely multiline, `]` is alone or with whitespace
             # Insert lines with indentation
             # Detect indent from `lines[insertion_idx]`
             indent = line_with_bracket[:len(line_with_bracket) - len(line_with_bracket.lstrip())]
             
             # If `]` was on a line with content ` "a" ]`, logic above would catch it?
             # If `]` is alone property indented.
             # We assume standard formatting: 4 spaces indent relative to list start?
             # Or match indent of previous line?
             
             # Use 4 spaces + indent?
             # Just usage of `indent + "    "`
             base_indent = "    " # default
             
             new_lines = []
             for item in to_insert:
                 new_lines.append(f"{indent}{base_indent}{item},")
                 
             lines[insertion_idx:insertion_idx] = new_lines # Insert before bracket
             
        safe_write(conftest_path, "\n".join(lines), overwrite=True)
        return conftest_path, "updated"
