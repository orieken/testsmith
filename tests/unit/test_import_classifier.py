from pathlib import Path
from testsmith.core.import_classifier import get_stdlib_modules, extract_root_package, classify_import, classify_all, ImportInfo

def test_stdlib_detection():
    stdlib = get_stdlib_modules()
    assert "os" in stdlib
    assert "sys" in stdlib
    assert "json" in stdlib

def test_extract_root_pkg():
    assert extract_root_package("os.path") == "os"
    assert extract_root_package("stripe") == "stripe"
    assert extract_root_package("a.b.c") == "a"
    assert extract_root_package("") == ""

def test_classify_import_stdlib():
    stdlib = get_stdlib_modules()
    pkg_map = {}
    info = ImportInfo("os.path", [], False, None, 1)
    
    assert classify_import(info, pkg_map, stdlib) == "stdlib"

def test_classify_import_internal():
    stdlib = get_stdlib_modules()
    pkg_map = {"my_pkg": Path("/tmp/my_pkg")}
    info = ImportInfo("my_pkg.utils", [], False, None, 1)
    
    assert classify_import(info, pkg_map, stdlib) == "internal"

def test_classify_import_external():
    stdlib = get_stdlib_modules()
    pkg_map = {"my_pkg": Path("/tmp/my_pkg")}
    info = ImportInfo("requests", [], False, None, 1)
    
    assert classify_import(info, pkg_map, stdlib) == "external"

def test_classify_relative():
    stdlib = get_stdlib_modules()
    pkg_map = {}
    info = ImportInfo(".utils", [], True, None, 1)
    
    assert classify_import(info, pkg_map, stdlib) == "internal"

def test_classify_all():
    info_stdlib = ImportInfo("os", [], False, None, 1)
    info_internal = ImportInfo("my_pkg", [], False, None, 2)
    info_external = ImportInfo("requests", [], False, None, 3)
    
    pkg_map = {"my_pkg": Path("/tmp")}
    
    classified = classify_all([info_stdlib, info_internal, info_external], pkg_map)
    
    assert len(classified.stdlib) == 1
    assert classified.stdlib[0].module == "os"
    assert len(classified.internal) == 1
    assert classified.internal[0].module == "my_pkg"
    assert len(classified.external) == 1
    assert classified.external[0].module == "requests"
