"""
LLM-based test body generation using Anthropic API.
"""
import os
import re
from typing import Any

from testsmith.support.models import AnalysisResult, LLMConfig
from testsmith.support.exceptions import TestSmithError

try:
    import anthropic
except ImportError:
    anthropic = None


def build_prompt(member_name: str, member_kind: str, source_code: str, fixtures: list[tuple]) -> str:
    """
    Construct the prompt for the LLM.
    """
    fixture_names = [f[2] for f in fixtures]
    
    prompt = f"""You are an expert Python testing assistant. Your task is to write a comprehensive test body for a specific {member_kind} named `{member_name}`.

Here is the source code of the module:
```python
{source_code}
```

The test file already has necessary imports and fixtures.
Available fixtures: {', '.join(fixture_names)}

Please write the Python code for the test function(s) or method(s) to test `{member_name}`. 
- Include a happy path test.
- Include edge case tests if applicable.
- Use `pytest`.
- Do NOT wrap the code in a class if it's a function test, just write the `def test_...` functions.
- If it's a class method, write the test methods (e.g. `def test_method_name(self, ...)`).
- Use `assert` statements.
- Do NOT include imports or `if __name__ == "__main__":`.
- Output ONLY the python code for the tests, wrapped in a markdown code block.
"""
    return prompt


def call_llm(prompt: str, config: LLMConfig) -> str:
    """
    Call Anthropic API to generate text.
    """
    if anthropic is None:
        raise TestSmithError("Anthropic library not installed. Run 'pip install anthropic'.")

    api_key = os.environ.get(config.api_key_env_var)
    if not api_key:
        raise TestSmithError(f"API key not found in environment variable {config.api_key_env_var}.")

    client = anthropic.Anthropic(api_key=api_key)

    try:
        message = client.messages.create(
            model=config.model,
            max_tokens=config.max_tokens_per_function,
            temperature=0.0,
            system="You are a strict code generation assistant. You only output valid Python code in code blocks.",
            messages=[
                {"role": "user", "content": prompt}
            ]
        )
        return message.content[0].text
    except Exception as e:
        raise TestSmithError(f"Anthropic API call failed: {e}")


def parse_llm_response(response: str) -> list[str]:
    """
    Extract code blocks from response.
    Returns a list of lines (clean code).
    """
    # Regex to find code blocks
    match = re.search(r"```python\n(.*?)\n```", response, re.DOTALL)
    if not match:
        match = re.search(r"```\n(.*?)\n```", response, re.DOTALL)
    
    if match:
        code = match.group(1)
        return code.splitlines()
    
    # Fallback: if no blocks, assume whole text is code but warn/clean?
    # For now, return empty if no code block found to avoid garbage
    return []


def generate_test_bodies(analysis: AnalysisResult, config: LLMConfig) -> dict[str, list[str]]:
    """
    Generate test bodies for all public members.
    Returns dict mapping member name to list of code lines.
    """
    if not config.enabled:
        return {}

    source_code = analysis.source_path.read_text()
    # We might want to pass fixtures info. 
    # But `process_file` in CLI has fixture info. 
    # `analysis` has imports but not the decided fixture names strictly.
    # We can approximate or standard fixture naming.
    # The prompt builder uses `fixtures` list. 
    # refactor: generate_test_bodies should assume knowledge of fixtures?
    # For V1.2, let's just pass empty list or inferred ones.
    # Actually `generate_test` knows the fixtures.
    # But we need bodies BEFORE `generate_test` or INSIDE it.
    
    # We will pass this map to `generate_test`.
    
    bodies = {}
    
    print(f"Generating test bodies using {config.model}...")
    
    for member in analysis.public_api:
        # Simplification: We assume we mocked all external deps as fixtures named after modules.
        # We can extract that from analysis.imports.external
        fixtures = [] # placeholder for now, or infer from imports
        
        prompt = build_prompt(member.name, member.kind, source_code, fixtures)
        try:
            response = call_llm(prompt, config)
            code_lines = parse_llm_response(response)
            if code_lines:
                bodies[member.name] = code_lines
            else:
                 print(f"Warning: No code found in LLM response for {member.name}")
        except TestSmithError as e:
            print(f"LLM Error for {member.name}: {e}")
            
    return bodies
