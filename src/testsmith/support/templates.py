"""
Code generation templates.
"""
from typing import Any


def render_test_file(
    module_name: str,
    public_members: list[dict[str, Any]],
    fixture_imports: list[tuple[str, str]],
    internal_imports: list[str],
) -> str:
    """
    Render a test file content.

    Args:
        module_name: Name of the module under test.
        public_members: List of dicts describing members to test.
                        Each dict should have:
                        - name: str
                        - kind: "class" or "function"
                        - methods: list[dict] (for classes) -> each has 'name', 'params'
                        - params: list[str] (for functions)
        fixture_imports: List of (module, name) tuples for fixtures.
        internal_imports: List of internal import lines (strings).
        
    Returns:
        String containing the complete Python test file.
    """
    lines = []
    lines.append(f'"""Tests for {module_name} module."""')
    lines.append("import pytest")
    
    # Internal imports
    if internal_imports:
        lines.append("")
        for imp in internal_imports:
            lines.append(imp)
            
    # Fixture imports (if we needed explicit imports, but typically conftest handles this)
    # The prompt says "fixture_imports" is passed. 
    # If using conftest re-exports, we might not need explicit imports here?
    # Architecture says: "Fixture Discovery by Test Files: ... via conftest ... explicit imports not needed"
    # But later: "Alternatively... re-export... so test files just declare fixture parameters"
    # Prompt 7 says: "Generate test file... Internal imports... A test class..."
    # It lists "fixture_imports" argument. 
    # Let's assume we might need them if not using the conftest trick yet, 
    # OR the prompt implies we should put them if configured.
    # For now, I'll add them if valid.
    if fixture_imports:
         lines.append("")
         for mod, name in fixture_imports:
             lines.append(f"from {mod} import {name}")

    lines.append("")
    lines.append("")

    for member in public_members:
        name = member.get("name")
        kind = member.get("kind")
        body = member.get("body") # Custom body from LLM
        
        if kind == "class":
            lines.append(f"class Test{name}:")
            lines.append(f'    """Tests for {name}."""')
            lines.append("")
            
            if body:
                # Body contains the test methods. We just indent it.
                # Assuming body is list of strings.
                for line in body:
                    lines.append(f"    {line}")
                lines.append("")
                continue

            methods = member.get("methods", [])
            if not methods:
                lines.append("    pass")
                lines.append("")
                continue

            for method in methods:
                m_name = method["name"]
                m_params = method.get("params", []) # fixtures
                
                # Setup params string: self, fixture1, fixture2
                param_str = ", ".join(["self"] + m_params)
                
                lines.append(f"    def test_{m_name}({param_str}):")
                lines.append(f'        """Test {name}.{m_name}."""')
                lines.append("        # TODO: Implement test")
                lines.append("        pass")
                lines.append("")
        
        elif kind == "function":
            # Standalone function -> Test class or just function?
            camel_name = "".join(x.capitalize() or "_" for x in name.split("_"))
            class_name = f"Test{camel_name}"

            lines.append(f"class {class_name}:")
            lines.append(f'    """Tests for {name}."""')
            lines.append("")
            
            if body:
                for line in body:
                    lines.append(f"    {line}")
                lines.append("")
                continue
            
            params = member.get("params", [])
            param_str = ", ".join(["self"] + params)
            
            lines.append(f"    def test_{name}({param_str}):")
            lines.append(f'        """Test {name}."""')
            lines.append("        # TODO: Implement test")
            lines.append("        pass")
            lines.append("")

    # Ensure single newline at end
    return "\n".join(lines).strip() + "\n"


def render_fixture_file(
    dependency_name: str, 
    sub_modules: list[str], 
    imported_names: dict[str, list[str]]
) -> str:
    """
    Render a shared fixture file.
    
    Args:
        dependency_name: Root package name (e.g. "stripe")
        sub_modules: List of mocked submodules (e.g. ["stripe.checkout"])
        imported_names: Dict of specific classes/funcs mocked (unused in basic skeleton but good for future)
    """
    lines = []
    lines.append(f'"""Shared mock fixtures for the {dependency_name} external dependency."""')
    lines.append("import pytest")
    lines.append("")
    lines.append("")
    lines.append("@pytest.fixture")
    
    fixture_name = f"mock_{dependency_name.replace('.', '_').replace('-', '_')}"
    
    lines.append(f"def {fixture_name}(mocker):")
    lines.append(f'    """Mock for {dependency_name} and its sub-modules."""')
    lines.append("    mock = mocker.Mock()")
    
    # Sort sub-modules to ensure deterministic output
    sorted_subs = sorted(sub_modules)
    
    # We need to handle nested mocking. 
    # e.g. mock.checkout.Session = mocker.Mock()
    # For now, straightforward implementation based on Prompt 6 example
    
    for sub in sorted_subs:
        if sub == dependency_name:
            continue
        # stripe.checkout -> mock.checkout
        # we assume dependency_name is the prefix
        if sub.startswith(dependency_name + "."):
            suffix = sub[len(dependency_name)+1:]
            # handle multi-level? stripe.checkout.session -> mock.checkout.session
            # yes, standard mock attributes work that way
            lines.append(f"    mock.{suffix} = mocker.Mock()")
    
    lines.append("    mocker.patch.dict(\"sys.modules\", {")
    lines.append(f'        "{dependency_name}": mock,')
    for sub in sorted_subs:
        if sub == dependency_name:
            continue
        if sub.startswith(dependency_name + "."):
             suffix = sub[len(dependency_name)+1:]
             lines.append(f'        "{sub}": mock.{suffix},')
    lines.append("    })")
    
    lines.append("    return mock")
    lines.append("")
    
    return "\n".join(lines)


def render_conftest_pytest_configure(paths: list[str]) -> str:
    """Render pytest_configure hook for conftest.py."""
    lines = []
    lines.append("def pytest_configure(config):")
    lines.append('    """Register project paths for test discovery."""')
    lines.append("    import sys")
    lines.append("    import os")
    lines.append("")
    lines.append("    paths_to_add = [")
    for p in paths:
        lines.append(f'        "{p}",')
    lines.append("    ]")
    lines.append("")
    lines.append("    for p in paths_to_add:")
    lines.append("        sys.path.append(os.path.abspath(p))")
    lines.append("")
    return "\n".join(lines)


def render_init_file() -> str:
    """Render content for __init__.py."""
    return ""
