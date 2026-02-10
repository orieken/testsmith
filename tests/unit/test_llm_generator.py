"""
Unit tests for LLM generator module.
"""

import pytest
from unittest.mock import MagicMock, patch
from pathlib import Path

from testsmith.generation.llm_generator import (
    build_prompt,
    call_llm,
    parse_llm_response,
    generate_test_bodies,
)
from testsmith.support.models import (
    LLMConfig,
    AnalysisResult,
    PublicMember,
    ProjectContext,
    ClassifiedImports,
)
from testsmith.support.exceptions import TestSmithError

# Mock anthropic module for tests
import sys

mock_anthropic = MagicMock()
sys.modules["anthropic"] = mock_anthropic


def test_build_prompt():
    """Test prompt construction."""
    code = "def foo(): pass"
    fixtures = [("path", "created", "mock_stripe")]
    prompt = build_prompt("foo", "function", code, fixtures)

    assert "You are an expert Python testing assistant" in prompt
    assert "def foo(): pass" in prompt
    assert "mock_stripe" in prompt
    assert "happy path" in prompt


def test_parse_llm_response_valid():
    """Test extracting code from markdown block."""
    response = """
Here is the code:
```python
def test_foo():
    assert True
```
Hope it helps.
"""
    lines = parse_llm_response(response)
    assert lines == ["def test_foo():", "    assert True"]


def test_parse_llm_response_no_block():
    """Test fallback when no code block is found."""
    response = "No code here."
    lines = parse_llm_response(response)
    assert lines == []


@patch("testsmith.generation.llm_generator.anthropic", mock_anthropic)
@patch("os.environ.get")
def test_call_llm_success(mock_env):
    """Test successful API call."""
    mock_env.return_value = "fake-key"

    # Mock client and message
    mock_client = MagicMock()
    mock_anthropic.Anthropic.return_value = mock_client

    mock_message = MagicMock()
    mock_message.content = [MagicMock(text="response text")]
    mock_client.messages.create.return_value = mock_message

    config = LLMConfig(enabled=True, api_key_env_var="KEY")
    result = call_llm("prompt", config)

    assert result == "response text"
    mock_client.messages.create.assert_called_once()
    assert (
        mock_client.messages.create.call_args[1]["messages"][0]["content"] == "prompt"
    )


@patch("testsmith.generation.llm_generator.anthropic", None)
def test_call_llm_missing_library():
    """Test error when library is missing."""
    config = LLMConfig(enabled=True)
    with pytest.raises(TestSmithError, match="not installed"):
        call_llm("prompt", config)


@patch("testsmith.generation.llm_generator.call_llm")
def test_generate_test_bodies(mock_call):
    """Test orchestration."""
    mock_call.return_value = "```python\ndef test_x(): pass\n```"

    analysis = AnalysisResult(
        source_path=Path("src/foo.py"),
        module_name="foo",
        imports=ClassifiedImports(),
        public_api=[
            PublicMember("foo", "function", [], [], None),
            PublicMember("Bar", "class", [], ["baz"], None),
        ],
        project=ProjectContext(Path("root"), {}, None, []),
    )
    # Mock read_text to avoid file I/O
    with patch("pathlib.Path.read_text", return_value="source code"):
        config = LLMConfig(enabled=True)
        bodies = generate_test_bodies(analysis, config)

    assert "foo" in bodies
    assert bodies["foo"] == ["def test_x(): pass"]
    assert "Bar" in bodies  # It generates for Bar too
    assert mock_call.call_count == 2
