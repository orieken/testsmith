import pytest
import ast
from pathlib import Path
from testsmith.core.source_analyzer import extract_imports, analyze_file
from testsmith.support.models import ImportInfo, ProjectContext, ClassifiedImports
from testsmith.support.exceptions import SourceParseError

def test_extract_imports_simple():
    code = "import os"
    tree = ast.parse(code)
    imports = extract_imports(tree)
    assert len(imports) == 1
    assert imports[0].module == "os"
    assert imports[0].is_from is False

def test_extract_imports_alias():
    code = "import pandas as pd"
    tree = ast.parse(code)
    imports = extract_imports(tree)
    assert len(imports) == 1
    assert imports[0].module == "pandas"
    assert imports[0].alias == "pd"

def test_extract_imports_from():
    code = "from os import path, sep"
    tree = ast.parse(code)
    imports = extract_imports(tree)
    assert len(imports) == 1
    assert imports[0].module == "os"
    assert imports[0].is_from is True
    assert imports[0].names == ["path", "sep"]

def test_extract_imports_star():
    code = "from math import *"
    tree = ast.parse(code)
    imports = extract_imports(tree)
    assert len(imports) == 1
    assert imports[0].names == ["*"]

def test_extract_imports_relative():
    code = "from . import utils"
    tree = ast.parse(code)
    imports = extract_imports(tree)
    assert len(imports) == 1
    assert imports[0].module == "."
    
    code2 = "from ..sub import foo"
    tree2 = ast.parse(code2)
    imports2 = extract_imports(tree2)
    assert imports2[0].module == "..sub"

def test_extract_imports_nested():
    code = """
try:
    import json
except ImportError:
    import simplejson as json
    
if True:
    import sys
"""
    tree = ast.parse(code)
    imports = extract_imports(tree)
    modules = {i.module for i in imports}
    assert "json" in modules
    assert "simplejson" in modules
    assert "sys" in modules

def test_analyze_file_end_to_end(tmp_path):
    source_file = tmp_path / "test_module.py"
    source_file.write_text("import os\ndef foo(): pass", encoding="utf-8")
    
    ctx = ProjectContext(
        root=tmp_path,
        package_map={},
        conftest_path=None,
        existing_paths=[]
    )
    
    result = analyze_file(source_file, ctx)
    
    assert result.module_name == "test_module"
    assert result.source_path == source_file
    # Check imports (stdlib os)
    assert len(result.imports.stdlib) == 1
    assert result.imports.stdlib[0].module == "os"
    # Check public api
    assert len(result.public_api) == 1
    assert result.public_api[0].name == "foo"

def test_analyze_file_not_found(tmp_path):
    ctx = ProjectContext(root=tmp_path, package_map={}, conftest_path=None, existing_paths=[])
    with pytest.raises(FileNotFoundError):
        analyze_file(tmp_path / "nonexistent.py", ctx)

def test_analyze_syntax_error(tmp_path):
    source_file = tmp_path / "broken.py"
    source_file.write_text("def broken(", encoding="utf-8")
    ctx = ProjectContext(root=tmp_path, package_map={}, conftest_path=None, existing_paths=[])
    
    with pytest.raises(SourceParseError):
        analyze_file(source_file, ctx)
