import pytest
import ast
from testsmith.core.module_inspector import extract_public_functions, extract_public_classes, inspect_module
from testsmith.support.exceptions import SourceParseError
from testsmith.support.models import PublicMember

def test_extract_functions():
    code = """
def foo(a, b):
    '''Docstring'''
    pass
def _bar():
    pass
async def baz(c):
    pass
"""
    tree = ast.parse(code)
    funcs = extract_public_functions(tree)
    
    assert len(funcs) == 2
    f1 = next(f for f in funcs if f.name == "foo")
    assert f1.parameters == ["a", "b"]
    assert f1.docstring == "Docstring"
    
    f2 = next(f for f in funcs if f.name == "baz")
    assert f2.parameters == ["c"]

def test_extract_classes():
    code = """
class MyClass:
    '''Class doc'''
    def __init__(self, x):
        pass
    def method(self, y):
        pass
    def _internal(self):
        pass
class _Private:
    pass
"""
    tree = ast.parse(code)
    classes = extract_public_classes(tree)
    
    assert len(classes) == 1
    c1 = classes[0]
    assert c1.name == "MyClass"
    assert c1.docstring == "Class doc"
    assert c1.parameters == ["x"]  # from __init__
    assert "method" in c1.methods
    assert "_internal" not in c1.methods

def test_inspect_module_syntax_error():
    with pytest.raises(SourceParseError):
        inspect_module("def broken(:")

def test_inspect_module_ordering():
    code = """
def func(): pass
class Cls: pass
"""
    members = inspect_module(code)
    # Expect classes first then functions
    assert members[0].kind == "class"
    assert members[0].name == "Cls"
    assert members[1].kind == "function"
    assert members[1].name == "func"

def test_inspect_sample_files():
    from pathlib import Path
    fixtures_dir = Path(__file__).parent.parent / "fixtures" / "sample_sources"
    
    # Simple module
    simple = (fixtures_dir / "simple_module.py").read_text()
    members = inspect_module(simple)
    assert len(members) == 2
    assert {m.name for m in members} == {"func_one", "func_two"}
    
    # Class heavy
    class_heavy = (fixtures_dir / "class_heavy_module.py").read_text()
    members = inspect_module(class_heavy)
    assert len(members) == 2 # MyClass + standalone_func
    cls = next(m for m in members if m.kind == "class")
    assert cls.name == "MyClass"
    assert "method_one" in cls.methods
    
    # Private members
    private = (fixtures_dir / "private_members.py").read_text()
    members = inspect_module(private)
    assert len(members) == 1
    assert members[0].name == "public_func"
    
    # Async
    async_code = (fixtures_dir / "async_module.py").read_text()
    members = inspect_module(async_code)
    assert len(members) == 2
    assert "async_func" in [m.name for m in members]
    
    # Empty
    empty_code = (fixtures_dir / "empty_module.py").read_text()
    assert inspect_module(empty_code) == []
    
    # Syntax Error
    syntax_error_code = (fixtures_dir / "syntax_error_module.py").read_text()
    with pytest.raises(SourceParseError):
        inspect_module(syntax_error_code, "syntax_error_module.py")

