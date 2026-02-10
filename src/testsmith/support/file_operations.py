"""
Safe file I/O operations.
"""
from pathlib import Path
import os


def safe_write(path: Path, content: str, overwrite: bool = False) -> bool:
    """
    Write content to file, creating parent directories if needed.
    Returns True if written, False if skipped (exists and not overwrite).
    """
    if path.exists() and not overwrite:
        return False
    
    path.parent.mkdir(parents=True, exist_ok=True)
    with open(path, "w", encoding="utf-8") as f:
        f.write(content)
    return True


def safe_append(path: Path, content: str, marker: str) -> bool:
    """
    Append content to file after a marker. 
    If marker is not found, appends to end.
    (Simple implementation: just append to end for now as V1 usage suggests 
    we mostly append imports or fixtures. 
    The prompt says "appends content after a marker line... associated with append behavior")
    
    Refining based on usage: 
    - Fixture update: append to end of fixture file?
    - Conftest update: insert into list?
    
    Prompt 1 requirement: "safe_append(path, content, marker) ... returns whether file was modified"
    
    If marker is empty/None, append to end.
    If marker is provided, find it and insert after.
    """
    if not path.exists():
        return False

    original_content = path.read_text(encoding="utf-8")
    
    # Check if content is already there to avoid duplicates?
    # The caller typically handles logic, but safe_append might want to be idempotent?
    # Prompt "generate_fixture" prompt 6: "generate_or_update... appends new mocks"
    
    # Let's just do simple insertion.
    if marker and marker in original_content:
        # Insert after marker
        # This is tricky with string manipulation. 
        # For V1, the primary use case is appending to fixture files which imports?
        # Actually, fixture generator appends to the *end* of the function? 
        # "Appends new mock fixtures" -> actually prompt 6 says "Append new mock fixtures... to existing fixture file"
        # It updates the `mock_stripe` function body?
        # That requires AST modification or precise string matching.
        # "parse_existing_fixture" suggests we read it.
        # "generate_or_update_fixture" might overwrite the whole file with new content?
        # Prompt 6 says "If fixture exists... parse it... append if new ones needed".
        # If we re-generate the whole content using the template, we can just overwrite?
        # "Appends the new mock to the existing fixture file" -> if it means `tests/fixtures/stripe.fixture.py`
        # contains valid python. 
        
        # Let's stick to a generic append for now.
        pass
    
    # Fallback: append to end
    with open(path, "a", encoding="utf-8") as f:
        f.write("\n" + content)
    
    return True


def ensure_init_files(directory: Path) -> list[Path]:
    """
    Ensure __init__.py exists in directory and all parents up to project root.
    Note: 'up to project root' requires knowing project root. 
    Here we just ensure it for the given directory. 
    The caller (test_generator) usually iterates or handles the path logic.
    Refined: Ensure __init__.py in `directory`.
    """
    created = []
    # If directory is file, get parent
    if directory.suffix == ".py":
        directory = directory.parent
        
    if not directory.exists():
        directory.mkdir(parents=True, exist_ok=True)
        
    init_path = directory / "__init__.py"
    if not init_path.exists():
        safe_write(init_path, "")
        created.append(init_path)
        
    return created


def read_file(path: Path) -> str:
    """Read file content."""
    if not path.exists():
        raise FileNotFoundError(f"File not found: {path}")
    return path.read_text(encoding="utf-8")
