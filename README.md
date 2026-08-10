# Assay

> *assay (v.)* — to test the quality or composition of something; from Old French *assai*, "trial, test." In metallurgy, an assay determines the purity of a metal sample. Here, it determines the test coverage of your code.

**Language-agnostic test scaffold generator.** Point it at any source file and it writes the boilerplate so you can write the assertions.

> **Formerly published as `testsmith`.** Renamed in 2026 following a trade name conflict with [testsmith.io](https://testsmith.io), which has operated under the Testsmith name since 2018. All functionality is identical — only the name changed. See [migrating from testsmith](#migrating-from-testsmith).

---

## Install

```sh
pip install assay-cli
```

## Quick start

```sh
# Generate a test scaffold for one file
assay generate src/services/payment.py

# Generate tests for every untested file
assay generate --all

# Preview without writing
assay generate --all --dry-run

# Watch for changes and auto-regenerate
assay watch
```

## Commands

| Command | Description |
|---------|-------------|
| `assay generate [file]` | Generate test scaffolds |
| `assay generate --all` | Process every untested file |
| `assay generate --path <dir>` | Process untested files under a directory |
| `assay validate` | Check existing tests against adapter conventions |
| `assay migrate --from jest --to vitest` | Rewrite tests between frameworks |
| `assay gaps` | Coverage gap report |
| `assay graph` | Dependency graph |
| `assay watch` | Auto-regenerate on file save |
| `assay init` | Create `.assay.yaml` config |
| `assay learn <file>` | Extract patterns from an existing test (requires LLM) |

## Configuration (`.assay.yaml`)

```yaml
language: python       # override auto-detection
test_root: tests/
fixture_dir: tests/fixtures/

llm:
  enabled: false
  provider: anthropic  # anthropic | openai | ollama
  model: claude-sonnet-4-6
  api_key_env_var: ANTHROPIC_API_KEY
```

Run `assay init` to generate a starter config.

## LLM setup

Assay works offline with TODO stubs. Pass `--llm` to have an LLM write the test bodies.

```sh
export ANTHROPIC_API_KEY=sk-ant-...
assay generate src/payment.py --llm
```

---

## Migrating from testsmith

```sh
pip uninstall testsmith
pip install assay-cli
```

Rename your config file:

```sh
mv .testsmith.yaml .assay.yaml
```

The `assay` command is a drop-in replacement for `testsmith`. All flags and subcommands are identical.

### Why the rename?

Shortly before a wider release we were contacted by [Roy de Kleijn](https://testsmith.io), who has been running a software testing consultancy under the Testsmith trade name since 2018. Under Dutch trade name law, active prior use establishes protection — and given both projects operate in the software testing space, the potential for confusion was real. Roy handled it graciously, and we agreed to rename.

---

## License

MIT
