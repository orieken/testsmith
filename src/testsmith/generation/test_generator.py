"""
Generates test skeleton files.
"""

from pathlib import Path
from testsmith.support.config import TestSmithConfig
from testsmith.support.models import AnalysisResult, PublicMember, ImportInfo
from testsmith.support.templates import render_test_file
from testsmith.support.file_operations import safe_write, ensure_init_files
from testsmith.generation.fixture_generator import derive_fixture_name


def derive_test_path(
    source_path: Path, project_root: Path, config: TestSmithConfig
) -> Path:
    """
    Derive the test file path for a given source file.
    e.g. src/services/payment.py -> tests/src/services/test_payment.py
    """
    # If source_path is already relative, use it directly if it doesn't start with /
    # relative_to fails if paths are not on same drive or if source is not subpath
    if not source_path.is_absolute():
        rel_path = source_path
    else:
        try:
            rel_path = source_path.relative_to(project_root)
        except ValueError:
            # Fallback
            rel_path = Path(source_path.name)

    # config.test_root is "tests/"
    test_root = project_root / config.test_root

    # Mirror structure: tests / [rel_path.parent] / test_[name]
    # If rel_path is "src/foo.py", parent is "src".
    # Result: tests/src/test_foo.py

    parent = rel_path.parent
    name = rel_path.stem
    filename = f"test_{name}.py"

    return test_root / parent / filename


def determine_fixture_params(
    member: PublicMember, external_imports: list[ImportInfo]
) -> list[str]:
    """
    Determine which fixture parameters a test method needs.
    Based on the file's external imports.
    """
    # Simplification: If file imports a dep, all tests get the mock.
    # Logic: Convert import root packages to fixture names.
    # external_imports contains ImportInfo objects. import_classifier classified them.
    # But AnalysisResult has `imports: ClassifiedImports`.
    # We expect `external_imports` passed here to be the list of external ones.

    fixtures = []
    seen = set()

    for imp in external_imports:
        # We need the root package name to match the fixture generator logic.
        # ImportInfo has `module`. `testsmith.core.import_classifier.extract_root_package` logic?
        # We don't have that function exposed here easily unless we import it or assume ImportInfo
        # matches. ImportInfo.module is "stripe.checkout". Root is "stripe".
        # We should use the same logic or helper.
        root = imp.module.split(".")[0]
        if root not in seen:
            fixtures.append(derive_fixture_name(root))
            seen.add(root)

    return sorted(fixtures)


def generate_test_file(
    analysis: AnalysisResult,
    fixture_imports: list[tuple[str, str]],
    config: TestSmithConfig,
    test_bodies: dict[str, list[str]] | None = None,
) -> str:
    """
    Generate test file content.
    """
    # Prepare data for template

    # 1. Internal Imports (The module under test)
    # We need to import the classes/functions we are testing.
    # from path.to.module import Foo, bar
    # AnalysisResult has `analysis.module_name` but that is just "payment".
    # We need the full python path for import.
    # How to derive full python path from file path?
    # src/services/payment.py -> services.payment? or src.services.payment?
    # It depends on where the python path root is.
    # Usually `src` is in pythonpath.
    # `project_context` has `package_map`?
    # Heuristic:
    # If file is in `src/package/module.py`, and `src` is a root?
    # ProjectContext doesn't explicitly store "source roots".
    # Implementation 2 (project_detector) `scan_packages` works.
    # But simplifying: we use relative imports if possible? No, tests to src usually need absolute.
    # If we assume `src` layout or flat layout.
    #
    # Let's try to derive dotted path from `analysis.source_path` relative to `project.root`.
    # And handle `src` stripping if common convention.
    # `src/testsmith/core/source_analyzer.py` -> `testsmith.core.source_analyzer`

    # If source_path is absolute, make relative to root. If relative, use as is.
    if analysis.source_path.is_absolute():
        try:
            rel = analysis.source_path.relative_to(analysis.project.root)
        except ValueError:
            rel = Path(analysis.source_path.name)
    else:
        rel = analysis.source_path
    parts = list(rel.parts)
    # Remove extension from last part
    parts[-1] = rel.stem

    # Heuristic: if first part is "src", remove it?
    # Poetry projects: `packages = [{include = "testsmith", from = "src"}]`
    # So `src` is NOT part of the module name.
    if parts[0] == "src":
        parts = parts[1:]

    module_path = ".".join(parts)

    # Internal imports list
    # "from module_path import Name1, Name2"
    internal_imports = []
    names_to_import = [m.name for m in analysis.public_api]
    if names_to_import:
        internal_imports.append(
            f"from {module_path} import {', '.join(sorted(names_to_import))}"
        )

    # 2. Public Members & Fixtures
    # We need to structure `public_members` for the template.
    # Template expects: {name, kind, methods: [...], params: [...]}
    # `params` here refers to TEST params (fixtures), NOT function params.

    external_deps = analysis.imports.external
    # Get fixture params valid for this file
    # We calc once per file for v1 approximation
    fixture_params = determine_fixture_params(None, external_deps)

    template_members = []
    for member in analysis.public_api:
        m_dict = {
            "name": member.name,
            "kind": member.kind,  # "class" or "function"
            "params": fixture_params,  # For functions
        }

        if test_bodies and member.name in test_bodies:
            m_dict["body"] = test_bodies[member.name]

        if member.kind == "class" and "body" not in m_dict:
            # For class, we have methods.
            # PublicMember model (src/testsmith/support/models.py):
            # name, kind, line_number, docstring, parameters, methods (list[str] or list of object?)
            # Checking `models.py`: `methods: list[str] = field(default_factory=list)`
            # It stores method NAMES only.
            # Template expects `methods: list[dict]` where dict has `name` and `params`.

            methods_list = []
            for method_name in member.methods:
                methods_list.append(
                    {
                        "name": method_name,
                        "params": fixture_params,  # methods get fixtures too
                    }
                )
            m_dict["methods"] = methods_list

        template_members.append(m_dict)

    return render_test_file(
        module_name=module_path,
        public_members=template_members,
        fixture_imports=fixture_imports,
        internal_imports=internal_imports,
    )


def generate_test(
    analysis: AnalysisResult,
    config: TestSmithConfig,
    test_bodies: dict[str, list[str]] | None = None,
) -> tuple[Path, str]:
    """
    Orchestrate test generation.
    """
    test_path = derive_test_path(analysis.source_path, analysis.project.root, config)

    if test_path.exists():
        return test_path, "skipped"

    # Ensure directories and __init__
    # test_path.parent
    # We need to ensure __init__.py exists in all parents up to test root?
    # `ensure_init_files` does that.
    # ensure_init_files(test_path.parent) # It works on the directory
    # We need to ensure __init__.py in all parents up to project root (or test root)
    # ensuring it makes the test suite discoverable as a package.
    current = test_path.parent
    while True:
        ensure_init_files(current)
        if current == analysis.project.root:
            break
        try:
            # If we go past root (e.g. into system root), stop
            if not current.is_relative_to(analysis.project.root):
                break
        except Exception:
            # is_relative_to might fail? Or python < 3.9
            pass

        if current == current.parent:  # root system
            break
        current = current.parent

    content = generate_test_file(analysis, [], config, test_bodies)

    if safe_write(test_path, content):
        return test_path, "created"

    return test_path, "error"  # Should fail in safe_write
