"""
Configuration management for TestSmith.
"""

from dataclasses import dataclass, field
from pathlib import Path
import sys
from testsmith.support.models import LLMConfig

# Compat for Python < 3.11
if sys.version_info >= (3, 11):
    import tomllib
else:
    import tomli as tomllib


@dataclass
class TestSmithConfig:
    """Configuration settings for TestSmith."""

    test_root: str = "tests/"
    root: str | None = None  # Optional explicit project root
    fixture_dir: str = "tests/fixtures/"
    fixture_suffix: str = ".fixture.py"
    conftest_path: str = "conftest.py"
    paths_to_add_var: str = "paths_to_add"
    exclude_dirs: list[str] = field(
        default_factory=lambda: [
            "node_modules",
            ".venv",
            "venv",
            "__pycache__",
            ".git",
            "build",
            "dist",
            ".tox",
            ".eggs",
        ]
    )
    llm: LLMConfig = field(default_factory=LLMConfig)


def load_config(path: Path | None = None) -> TestSmithConfig:
    """
    Load configuration.
    Args:
        path: Path to config file (pyproject.toml) OR project root directory.
              If None, returns default config.
    """
    if path is None:
        # Default to current working directory
        path = Path.cwd()

    if path.is_dir():
        config_path = path / "pyproject.toml"
    else:
        config_path = path

    if not config_path.exists():
        # If no config file found, return defaults
        return TestSmithConfig()

    try:
        with open(config_path, "rb") as f:
            data = tomllib.load(f)

        config_data = data.get("tool", {}).get("testsmith", {})

        # Handle LLM config
        llm_data = config_data.pop("llm", {})
        llm_config = LLMConfig(**llm_data)

        valid_keys = TestSmithConfig.__annotations__.keys()
        filtered_data = {
            k: v for k, v in config_data.items() if k in valid_keys and k != "llm"
        }

        return TestSmithConfig(llm=llm_config, **filtered_data)
    except Exception:
        # In case of any error (e.g. malformed TOML, missing section), return defaults
        return TestSmithConfig()
