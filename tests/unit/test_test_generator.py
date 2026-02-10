import pytest
from pathlib import Path
from testsmith.generation.test_generator import (
    derive_test_path,
    determine_fixture_params,
    generate_test_file,
    generate_test
)
from testsmith.support.config import TestSmithConfig
from testsmith.support.models import (
    AnalysisResult, ProjectContext, PublicMember, ClassifiedImports, ImportInfo
)

@pytest.fixture
def config():
    return TestSmithConfig()

def test_derive_test_path(config, tmp_path):
    root = tmp_path
    
    # Case 1: src layout
    src_file = root / "src/pkg/module.py"
    expected = root / "tests/src/pkg/test_module.py"
    assert derive_test_path(src_file, root, config) == expected
    
    # Case 2: flat layout
    flat_file = root / "pkg/module.py"
    expected_flat = root / "tests/pkg/test_module.py"
    assert derive_test_path(flat_file, root, config) == expected_flat

def test_determine_fixture_params():
    imports = [
        ImportInfo(module="stripe.checkout", names=[], is_from=False, alias=None, line_number=1),
        ImportInfo(module="requests", names=[], is_from=False, alias=None, line_number=2),
    ]
    # Assuming PublicMember is not used in v1 logic as we apply to all
    # PublicMember(name, kind, parameters, methods, docstring)
    member = PublicMember("Foo", "class", [], [], None)
    
    fixtures = determine_fixture_params(member, imports)
    assert "mock_stripe" in fixtures
    assert "mock_requests" in fixtures
    assert len(fixtures) == 2

def test_generate_test_file(config, tmp_path):
    # Mock AnalysisResult
    project = ProjectContext(root=tmp_path, package_map={}, conftest_path=None, existing_paths=[])
    
    # Fake source path
    source_path = tmp_path / "src/myapp/logic.py"
    
    # Public API
    api = [
        PublicMember("process", "function", ["x"], [], None),
        PublicMember("Handler", "class", [], ["handle"], None)
    ]
    
    # Imports
    imports = ClassifiedImports(
        stdlib=[],
        internal=[],
        external=[ImportInfo(module="stripe", names=[], is_from=False, alias=None, line_number=1)]
    )
    
    analysis = AnalysisResult(
        source_path=source_path,
        module_name="logic",
        imports=imports,
        public_api=api,
        project=project
    )
    
    content = generate_test_file(analysis, [], config)
    
    # Verify content
    assert "import pytest" in content
    expected_import = "from myapp.logic import Handler, process"
    assert expected_import in content, f"Expected '{expected_import}' in content:\n{content}"
    
    # Check fixtures in function
    # mock_stripe should be in params
    assert "def test_process(self, mock_stripe):" in content
    
    # Check class and method
    assert "class TestHandler:" in content
    assert "def test_handle(self, mock_stripe):" in content

def test_generate_test_orchestrator(config, tmp_path):
    # Setup
    src_dir = tmp_path / "src/pkg"
    src_dir.mkdir(parents=True)
    source_path = src_dir / "mod.py"
    source_path.touch()
    
    project = ProjectContext(root=tmp_path, package_map={}, conftest_path=None, existing_paths=[])
    analysis = AnalysisResult(
        source_path=source_path, 
        module_name="mod", 
        imports=ClassifiedImports([], [], []), 
        public_api=[], 
        project=project
    )
    
    # Generate
    path, action = generate_test(analysis, config)
    
    assert action == "created", f"Expected 'created' but got '{action}'"
    assert path.exists()
    assert path.name == "test_mod.py"
    # Check init files
    assert (tmp_path / "tests/__init__.py").exists()
    assert (tmp_path / "tests/src/pkg/__init__.py").exists()
    
    # Run again -> skip
    path2, action2 = generate_test(analysis, config)
    assert action2 == "skipped"

def test_generate_test_file_with_bodies(config, tmp_path):
    """Test injection of LLM-generated bodies."""
    project = ProjectContext(root=tmp_path, package_map={}, conftest_path=None, existing_paths=[])
    source_path = tmp_path / "src/logic.py"
    
    api = [
        PublicMember("process", "function", [], [], None),
        PublicMember("Handler", "class", [], ["handle"], None)
    ]
    imports = ClassifiedImports([], [], [])
    analysis = AnalysisResult(source_path, "logic", imports, api, project)
    
    bodies = {
        "process": ["def test_process():", "    assert True"],
        "Handler": ["def test_handle(self):", "    assert True"]
    }
    
    content = generate_test_file(analysis, [], config, test_bodies=bodies)
    
    assert "def test_process():" in content
    assert "    assert True" in content
    # Should not have default TODO for these
    # Actually templates put TODO if default. If body, no TODO.
    assert "# TODO: Implement test" not in content 
    
    assert "class TestHandler:" in content
    assert "def test_handle(self):" in content
