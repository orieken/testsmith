import ast
import pytest
from testsmith.support.templates import (
    render_test_file,
    render_fixture_file,
    render_conftest_pytest_configure,
)


def assert_valid_python(code: str):
    try:
        ast.parse(code)
    except SyntaxError as e:
        pytest.fail(f"Generated code invalid: {e}\nCode:\n{code}")


def test_render_test_file_structure():
    code = render_test_file(
        module_name="my_mod",
        public_members=[
            {
                "name": "MyClass",
                "kind": "class",
                "methods": [{"name": "foo", "params": []}],
            },
            {"name": "my_func", "kind": "function", "params": ["mock_stripe"]},
        ],
        fixture_imports=[],
        internal_imports=["from src.my_mod import MyClass, my_func"],
    )
    assert_valid_python(code)
    assert "class TestMyClass:" in code
    assert "def test_foo(" in code
    assert "class TestMyFunc:" in code


def test_render_fixture_file():
    code = render_fixture_file("stripe", ["stripe.checkout"], {})
    assert_valid_python(code)
    assert "@pytest.fixture" in code
    assert "def mock_stripe(mocker):" in code
    assert "mock.checkout = mocker.Mock()" in code


def test_render_conftest():
    code = render_conftest_pytest_configure(["src/", "tests/"])
    assert_valid_python(code)
    assert "paths_to_add = [" in code
    assert '"src/",' in code
