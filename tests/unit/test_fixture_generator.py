import pytest
from pathlib import Path
from testsmith.generation.fixture_generator import (
    derive_fixture_name,
    derive_fixture_filename,
    parse_existing_fixture,
    generate_fixture,
    generate_or_update_fixture,
    generate_fixtures_conftest,
)
from testsmith.support.config import TestSmithConfig as Config


@pytest.fixture
def config():
    return Config()


def test_derive_fixture_name():
    assert derive_fixture_name("stripe") == "mock_stripe"
    assert derive_fixture_name("my-lib") == "mock_my_lib"
    assert derive_fixture_name("os.path") == "mock_os_path"


def test_derive_fixture_filename(config):
    path = derive_fixture_filename("stripe", config)
    # Default config fixture_dir is 'tests/fixtures'
    # Our impl uses '_fixture.py' suffix
    assert path == Path("tests/fixtures/stripe_fixture.py")


def test_parse_existing_fixture(tmp_path):
    f = tmp_path / "f.py"
    f.write_text(
        """
def mock_foo(mocker):
    mocker.patch.dict("sys.modules", {"foo": mocker.Mock(), "foo.bar": mocker.Mock()})
""",
        encoding="utf-8",
    )

    res = parse_existing_fixture(f)
    assert "foo" in res["sub_modules"]
    assert "foo.bar" in res["sub_modules"]


def test_generate_fixture(config):
    # Just check it returns a string with expected content
    content = generate_fixture("stripe", ["stripe.checkout"], {}, config)
    assert "def mock_stripe(mocker):" in content
    # The template generates `mock.checkout = mocker.Mock()` and maps "stripe.checkout": mock.checkout
    assert "mock.checkout = mocker.Mock()" in content
    assert '"stripe.checkout": mock.checkout' in content


def test_generate_or_update_create(tmp_path, config):
    # Set config root
    # generate_or_update takes project_root. path is project_root / derived.
    # We need to ensure fixture dir exists or logic handles it.
    # Our templates/file_operations creates dirs.

    path, action = generate_or_update_fixture("stripe", [], {}, tmp_path, config)
    assert action == "created"
    assert path.exists()
    assert (tmp_path / "tests/fixtures/stripe_fixture.py").exists()


def test_generate_or_update_update(tmp_path, config):
    # Create existing
    f_path = tmp_path / "tests/fixtures/stripe_fixture.py"
    f_path.parent.mkdir(parents=True)
    f_path.write_text(
        """
import pytest
def mock_stripe(mocker):
    mocker.patch.dict("sys.modules", {
        "stripe": mocker.Mock(),
    })
""",
        encoding="utf-8",
    )

    # Update with new module
    path, action = generate_or_update_fixture(
        "stripe", ["stripe.checkout"], {}, tmp_path, config
    )
    assert action == "updated"

    content = f_path.read_text()
    # The update logic (regex) INSERTS `"": mocker.Mock()` directly into the dict for new modules
    # because it doesn't know how to add the attribute assignment lines easily.
    # See generate_or_update_fixture implementation:
    # lines_to_add.append(f'\n        "{mod}": mocker.Mock(),')

    assert '"stripe.checkout": mocker.Mock()' in content
    assert '"stripe": mocker.Mock()' in content


def test_generate_or_update_skip(tmp_path, config):
    # Create existing
    f_path = tmp_path / "tests/fixtures/stripe_fixture.py"
    f_path.parent.mkdir(parents=True)
    f_path.write_text(
        """
import pytest
def mock_stripe(mocker):
    mocker.patch.dict("sys.modules", {
        "stripe": mocker.Mock(),
    })
""",
        encoding="utf-8",
    )

    path, action = generate_or_update_fixture(
        "stripe", ["stripe"], {}, tmp_path, config
    )
    assert action == "skipped"


def test_generate_conftest():
    files = [
        Path("tests/fixtures/stripe_fixture.py"),
        Path("tests/fixtures/auth_fixture.py"),
    ]
    content = generate_fixtures_conftest(Path("."), files)

    assert "from .stripe_fixture import mock_stripe" in content
    assert "from .auth_fixture import mock_auth" in content
