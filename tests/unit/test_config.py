from pathlib import Path
from testsmith.support.config import TestSmithConfig, load_config

def test_default_config():
    cfg = TestSmithConfig()
    assert cfg.test_root == "tests/"
    assert "node_modules" in cfg.exclude_dirs

def test_load_config_defaults(tmp_path):
    # No pyproject.toml
    cfg = load_config(tmp_path)
    assert cfg.test_root == "tests/"

def test_load_config_from_file(tmp_path):
    pyproject = tmp_path / "pyproject.toml"
    # We rely on tomli/tomllib logic which expects valid TOML
    pyproject.write_text("""
[tool.testsmith]
test_root = "mytests/"
exclude_dirs = ["foo"]
""", encoding="utf-8")
    
    cfg = load_config(tmp_path)
    assert cfg.test_root == "mytests/"
    assert cfg.exclude_dirs == ["foo"]
