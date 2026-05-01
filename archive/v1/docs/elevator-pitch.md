# TestSmith — Elevator Pitch

> **One command. Zero `ModuleNotFoundError`. Test scaffolding that actually understands your codebase.**

---

## The Problem

Every Python project with non-trivial dependencies has the same testing friction: before you write a single assertion, you spend 20–30 minutes on ceremony. You read source files to figure out what needs mocking. You create fixture files by hand. You update `conftest.py` with paths. You run pytest, hit `ModuleNotFoundError`, fix, repeat.

This isn't a "your project" problem — it's a structural problem with how pytest expects you to wire things up. And it means tests don't get written, coverage stays low, and bugs ship.

## The Solution

**TestSmith** is a project-agnostic Python CLI tool that reads any source file, analyzes its dependency graph via AST parsing, and generates everything you need to start testing — in one command:

```bash
python -m testsmith src/services/payment_processor.py
```

That single invocation:

1. **Parses** the source file and classifies every import — stdlib gets skipped, anything in your project structure is internal, everything else is external
2. **Generates or updates** shared fixture files in `tests/fixtures/` — one fixture per external dependency, reused across all test files that need it
3. **Scaffolds** `tests/src/services/test_payment_processor.py` with proper imports, fixture wiring, and a test skeleton for every public function and class
4. **Updates** `conftest.py` — registers paths for the source directory, the test directory, and the fixtures directory

## Who Is It For?

Any Python developer who uses pytest and deals with:

- Projects where source code lives in nested directory structures
- External dependencies that are heavy, slow, or unavailable in test environments
- A centralized `conftest.py` with path management
- The recurring pain of getting a new test file "wired up" before you can write actual tests

TestSmith is project-agnostic. It doesn't care if you're building a web app, a CLI tool, a game engine, or a data pipeline. It reads your project structure and adapts.

## What Makes It Different?

- **AST-based analysis** — parses Python properly, not regex. Handles `import x`, `from x.y import z`, relative imports, and conditional imports inside `try/except` blocks.
- **Project-structure-aware classification** — doesn't rely on hardcoded lists. If a module resolves to a file in your project, it's internal. If not, it's external. Works on any project layout.
- **Shared fixtures** — external dependencies are mocked once in a shared fixture file. When `payment_processor.py` and `order_service.py` both depend on `stripe`, they share `tests/fixtures/stripe.fixture.py`. No duplication.
- **Idempotent** — run it twice, nothing breaks. Existing fixtures are updated (new mocks appended), existing test files are never overwritten, conftest paths are deduped.
- **Clean code output** — generated files follow PEP 8, include docstrings, and are ready for a craftsman to fill in real assertions.

## Where ML Fits (v2 Roadmap)

TestSmith v1 generates skeletons with `# TODO: Implement test` stubs. In v2, an LLM integration can analyze function signatures, docstrings, and type hints to generate *meaningful* initial assertions — happy path, edge cases, and error conditions. The ML layer is strictly additive: the tool works perfectly without it, and the generated suggestions are always marked as AI-generated for human review.

## The Outcome

What used to take 20–30 minutes of manual scaffolding now takes 5 seconds. Engineers spend their time writing meaningful tests, not fighting import machinery. Coverage goes up because the barrier to writing tests goes down.
