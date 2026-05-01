from pathlib import Path
from testsmith.support.models import ClassifiedImports, ProjectContext


def test_classified_imports_defaults():
    ci = ClassifiedImports()
    assert ci.stdlib == []
    assert ci.internal == []
    assert ci.external == []


def test_project_context_creation():
    pc = ProjectContext(
        root=Path("/tmp"), package_map={}, conftest_path=None, existing_paths=[]
    )
    assert pc.root == Path("/tmp")
