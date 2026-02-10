from testsmith.support.file_operations import safe_write, safe_append, ensure_init_files
import pytest
from pathlib import Path
from unittest.mock import patch, mock_open

def test_safe_write(tmp_path):
    p = tmp_path / "foo.txt"
    assert safe_write(p, "hello")
    assert p.read_text(encoding="utf-8") == "hello"
    
    with patch("builtins.open", mock_open()) as mock_file:
        _ = safe_write("test.txt", "content")
    assert p.read_text(encoding="utf-8") == "hello"
    
    # Should not overwrite
    assert not safe_write(p, "world")
    assert p.read_text(encoding="utf-8") == "hello"
    
    # Should overwrite
    assert safe_write(p, "world", overwrite=True)
    assert p.read_text(encoding="utf-8") == "world"

def test_safe_append(tmp_path):
    p = tmp_path / "append.txt"
    p.write_text("line1", encoding="utf-8")
    safe_append(p, "line2", marker=None)
    content = p.read_text(encoding="utf-8")
    assert "line1" in content
    assert "line2" in content

def test_ensure_init_files(tmp_path):
    d = tmp_path / "a/b/c"
    created = ensure_init_files(d)
    assert (d / "__init__.py").exists()
