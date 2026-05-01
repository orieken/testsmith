"""
Unit tests for fixture_pruner module.
"""

import pytest
from pathlib import Path
from testsmith.maintenance.fixture_pruner import (
    scan_used_dependencies,
    scan_existing_fixtures,
    identify_unused_fixtures,
    prune_fixtures,
    update_test_imports,
)
from testsmith.support.config import TestSmithConfig as Config


@pytest.fixture
def sample_project(tmp_path):
    """Create a minimal project for testing."""
    project_root = tmp_path / "project"
    project_root.mkdir()

    # Create pyproject.toml
    (project_root / "pyproject.toml").write_text("[tool.testsmith]\n", encoding="utf-8")

    # Create src structure
    src = project_root / "src" / "myapp"
    src.mkdir(parents=True)
    (src / "__init__.py").write_text("", encoding="utf-8")

    # Module with external deps
    (src / "api.py").write_text(
        """
import requests
from stripe import Charge

def fetch_data():
    pass
""",
        encoding="utf-8",
    )

    # Module with different external dep
    (src / "cache.py").write_text(
        """
import redis

def get_cache():
    pass
""",
        encoding="utf-8",
    )

    # Create fixtures directory
    fixtures_dir = project_root / "tests" / "fixtures"
    fixtures_dir.mkdir(parents=True)

    # Create some fixture files
    (fixtures_dir / "requests.fixture.py").write_text(
        "# mock requests", encoding="utf-8"
    )
    (fixtures_dir / "stripe.fixture.py").write_text("# mock stripe", encoding="utf-8")
    (fixtures_dir / "redis.fixture.py").write_text("# mock redis", encoding="utf-8")
    (fixtures_dir / "boto3.fixture.py").write_text(
        "# mock boto3 (unused)", encoding="utf-8"
    )

    return project_root


def test_scan_used_dependencies(sample_project):
    """Test scanning for used dependencies."""
    config = Config()
    used = scan_used_dependencies(sample_project, config)

    assert "requests" in used
    assert "stripe" in used
    assert "redis" in used
    assert "boto3" not in used  # Not imported anywhere


def test_scan_existing_fixtures(sample_project):
    """Test scanning for existing fixtures."""
    config = Config()
    fixture_dir = sample_project / "tests" / "fixtures"

    existing = scan_existing_fixtures(fixture_dir, config)

    assert "requests" in existing
    assert "stripe" in existing
    assert "redis" in existing
    assert "boto3" in existing
    assert len(existing) == 4


def test_identify_unused_fixtures(sample_project):
    """Test identifying unused fixtures."""
    config = Config()

    used = scan_used_dependencies(sample_project, config)
    fixture_dir = sample_project / "tests" / "fixtures"
    existing = scan_existing_fixtures(fixture_dir, config)

    unused = identify_unused_fixtures(used, existing)

    # boto3 is not used
    assert len(unused) == 1
    assert unused[0][0] == "boto3"


def test_prune_fixtures_dry_run(sample_project):
    """Test pruning in dry-run mode."""
    fixture_dir = sample_project / "tests" / "fixtures"

    unused = [("boto3", fixture_dir / "boto3.fixture.py")]
    results = prune_fixtures(unused, dry_run=True)

    assert len(results) == 1
    assert results[0] == ("boto3", "would_delete")

    # File should still exist
    assert (fixture_dir / "boto3.fixture.py").exists()


def test_prune_fixtures_confirm(sample_project):
    """Test pruning with confirmation."""
    fixture_dir = sample_project / "tests" / "fixtures"

    unused = [("boto3", fixture_dir / "boto3.fixture.py")]
    results = prune_fixtures(unused, dry_run=False)

    assert len(results) == 1
    assert results[0] == ("boto3", "deleted")

    # File should be deleted
    assert not (fixture_dir / "boto3.fixture.py").exists()


def test_update_test_imports(sample_project):
    """Test updating test imports."""
    # Create a test file with imports
    test_file = sample_project / "tests" / "test_api.py"
    test_file.parent.mkdir(parents=True, exist_ok=True)
    test_file.write_text(
        """
import pytest
from fixtures.boto3_fixture import mock_boto3
from fixtures.requests_fixture import mock_requests

def test_something(mock_boto3, mock_requests):
    pass
""",
        encoding="utf-8",
    )

    # Update imports for deleted boto3 fixture
    modified = update_test_imports(sample_project, ["boto3"])

    assert len(modified) == 1
    assert modified[0] == test_file

    # Check that boto3 import is commented out
    content = test_file.read_text(encoding="utf-8")
    assert "# PRUNED by TestSmith" in content
    assert "boto3" in content
    # requests import should still be there
    assert "from fixtures.requests_fixture import mock_requests" in content


def test_scan_existing_fixtures_no_directory(tmp_path):
    """Test scanning when fixture directory doesn't exist."""
    config = Config()
    non_existent = tmp_path / "nonexistent"

    existing = scan_existing_fixtures(non_existent, config)

    assert existing == {}


def test_identify_unused_fixtures_all_used():
    """Test when all fixtures are used."""
    used = {"requests", "stripe", "redis"}
    existing = {
        "requests": Path("/fake/requests.fixture.py"),
        "stripe": Path("/fake/stripe.fixture.py"),
        "redis": Path("/fake/redis.fixture.py"),
    }

    unused = identify_unused_fixtures(used, existing)

    assert len(unused) == 0
