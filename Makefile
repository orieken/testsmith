# ──────────────────────────────────────────────────────────────────────────────
# TestSmith — top-level Makefile
#
# ── Build ────────────────────────────────────────────────────────────────────
#   make build            compile dist/testsmith (stamped with git version)
#   make install          copy dist/testsmith → $GOPATH/bin/testsmith
#   make clean            remove dist/
#
# ── Quality ──────────────────────────────────────────────────────────────────
#   make test             go test ./...
#   make vet              go vet ./...
#   make lint             golangci-lint (skips if not installed)
#   make check            vet + test (fast CI path)
#
# ── Generate example projects (files left on disk to inspect) ─────────────
#   make example-go           generate stub tests in examples/go-service/
#   make example-python       …examples/python-service/
#   make example-typescript   …examples/typescript-service/
#   make example-java         …examples/java-service/
#   make example-csharp       …examples/csharp-service/
#   make examples             all five
#
# ── LLM variants (require ANTHROPIC_API_KEY) ─────────────────────────────
#   make example-go-llm / example-python-llm / … / examples-llm
#
# ── Cleanup (remove generated test files, leave source untouched) ─────────
#   make clean-example-go / clean-example-python / … / clean-examples
#
# ── CI smoke-test (generate + verify + clean in one step) ────────────────
#   make examples-ci          all five: generate, report, then clean
# ──────────────────────────────────────────────────────────────────────────────

BINARY := dist/testsmith
CMD    := ./cmd/testsmith
# Absolute path so sub-shell cd's can still reach the binary.
TS     := $(CURDIR)/$(BINARY)

# ── Build ─────────────────────────────────────────────────────────────────────

.PHONY: build
build: $(BINARY)

$(BINARY):
	@mkdir -p dist
	go build \
	    -ldflags "-X main.Version=$$(git describe --tags --always --dirty 2>/dev/null || echo dev)" \
	    -o $(BINARY) $(CMD)
	@echo "Built $(BINARY)"

.PHONY: install
install: build
	cp $(BINARY) $$(go env GOPATH)/bin/testsmith
	@echo "Installed to $$(go env GOPATH)/bin/testsmith"

# ── Quality ───────────────────────────────────────────────────────────────────

.PHONY: test
test:
	go test ./...

.PHONY: vet
vet:
	go vet ./...

.PHONY: lint
lint: vet
	@if command -v golangci-lint >/dev/null 2>&1; then \
	    golangci-lint run; \
	else \
	    echo "golangci-lint not found — skipping (install from https://golangci-lint.run)"; \
	fi

.PHONY: check
check: vet test

# ── Clean build artefacts ─────────────────────────────────────────────────────

.PHONY: clean
clean:
	rm -rf dist/
	@echo "Removed dist/"

# ──────────────────────────────────────────────────────────────────────────────
# Example targets
#
# generate-example-*   run testsmith inside the project; files stay on disk
# clean-example-*      delete the generated test files; source stays untouched
# example-*            alias for generate-example-* (for muscle-memory)
# example-*-llm        same but with --llm (requires ANTHROPIC_API_KEY)
# examples             generate all five
# examples-llm         generate all five with LLM bodies
# clean-examples       clean all five
# examples-ci          generate all five then clean (CI smoke-test)
# ──────────────────────────────────────────────────────────────────────────────

# ── go-service ────────────────────────────────────────────────────────────────

.PHONY: generate-example-go
generate-example-go: build
	@echo "\n── go-service (stubs) ──"
	cd examples/go-service && $(TS) generate --all
	@echo "Files written to examples/go-service — run 'make clean-example-go' when done."

.PHONY: generate-example-go-llm
generate-example-go-llm: build
	@echo "\n── go-service (LLM) ──"
	cd examples/go-service && $(TS) generate --all --llm
	@echo "Files written to examples/go-service — run 'make clean-example-go' when done."

.PHONY: clean-example-go
clean-example-go:
	@find examples/go-service -name '*_test.go' -print -delete
	@echo "go-service cleaned"

# Convenience aliases
.PHONY: example-go
example-go: generate-example-go

.PHONY: example-go-llm
example-go-llm: generate-example-go-llm

# ── python-service ────────────────────────────────────────────────────────────

.PHONY: generate-example-python
generate-example-python: build
	@echo "\n── python-service (stubs) ──"
	cd examples/python-service && $(TS) generate --all
	@echo "Files written to examples/python-service — run 'make clean-example-python' when done."

.PHONY: generate-example-python-llm
generate-example-python-llm: build
	@echo "\n── python-service (LLM) ──"
	cd examples/python-service && $(TS) generate --all --llm
	@echo "Files written to examples/python-service — run 'make clean-example-python' when done."

.PHONY: clean-example-python
clean-example-python:
	@find examples/python-service -name 'test_*.py' -print -delete
	@find examples/python-service -name '*_fixture.py' -print -delete
	@find examples/python-service -maxdepth 1 -name 'conftest.py' -print -delete
	@find examples/python-service -name '__init__.py' -print -delete
	@echo "python-service cleaned"

.PHONY: example-python
example-python: generate-example-python

.PHONY: example-python-llm
example-python-llm: generate-example-python-llm

# ── typescript-service ────────────────────────────────────────────────────────

.PHONY: generate-example-typescript
generate-example-typescript: build
	@echo "\n── typescript-service (stubs) ──"
	cd examples/typescript-service && $(TS) generate --all
	@echo "Files written to examples/typescript-service — run 'make clean-example-typescript' when done."

.PHONY: generate-example-typescript-llm
generate-example-typescript-llm: build
	@echo "\n── typescript-service (LLM) ──"
	cd examples/typescript-service && $(TS) generate --all --llm
	@echo "Files written to examples/typescript-service — run 'make clean-example-typescript' when done."

.PHONY: clean-example-typescript
clean-example-typescript:
	@find examples/typescript-service/src -name '*.test.ts' -print -delete
	@find examples/typescript-service -maxdepth 1 \( -name 'vitest.setup.ts' -o -name 'jest.setup.ts' \) -print -delete
	@echo "typescript-service cleaned"

.PHONY: example-typescript
example-typescript: generate-example-typescript

.PHONY: example-typescript-llm
example-typescript-llm: generate-example-typescript-llm

# ── java-service ──────────────────────────────────────────────────────────────

.PHONY: generate-example-java
generate-example-java: build
	@echo "\n── java-service (stubs) ──"
	cd examples/java-service && $(TS) generate --all
	@echo "Files written to examples/java-service — run 'make clean-example-java' when done."

.PHONY: generate-example-java-llm
generate-example-java-llm: build
	@echo "\n── java-service (LLM) ──"
	cd examples/java-service && $(TS) generate --all --llm
	@echo "Files written to examples/java-service — run 'make clean-example-java' when done."

.PHONY: clean-example-java
clean-example-java:
	@find examples/java-service/src/test -name '*Test.java' -print -delete 2>/dev/null || true
	@echo "java-service cleaned"

.PHONY: example-java
example-java: generate-example-java

.PHONY: example-java-llm
example-java-llm: generate-example-java-llm

# ── csharp-service ────────────────────────────────────────────────────────────

.PHONY: generate-example-csharp
generate-example-csharp: build
	@echo "\n── csharp-service (stubs) ──"
	cd examples/csharp-service && $(TS) generate --all
	@echo "Files written to examples/csharp-service — run 'make clean-example-csharp' when done."

.PHONY: generate-example-csharp-llm
generate-example-csharp-llm: build
	@echo "\n── csharp-service (LLM) ──"
	cd examples/csharp-service && $(TS) generate --all --llm
	@echo "Files written to examples/csharp-service — run 'make clean-example-csharp' when done."

.PHONY: clean-example-csharp
clean-example-csharp:
	@find examples/csharp-service -name '*Tests.cs' -o -name '*Test.cs' | xargs rm -f 2>/dev/null || true
	@echo "csharp-service cleaned"

.PHONY: example-csharp
example-csharp: generate-example-csharp

.PHONY: example-csharp-llm
example-csharp-llm: generate-example-csharp-llm

# ── Aggregate targets ─────────────────────────────────────────────────────────

.PHONY: examples
examples: generate-example-go generate-example-python generate-example-typescript generate-example-java generate-example-csharp
	@echo "\nAll example test files generated. Inspect them, then run 'make clean-examples' to remove."

.PHONY: examples-llm
examples-llm: generate-example-go-llm generate-example-python-llm generate-example-typescript-llm generate-example-java-llm generate-example-csharp-llm
	@echo "\nAll example test files generated (LLM). Run 'make clean-examples' to remove."

.PHONY: clean-examples
clean-examples: clean-example-go clean-example-python clean-example-typescript clean-example-java clean-example-csharp

# CI smoke-test: generate + immediately clean (proves the tool runs without leaving debris).
.PHONY: examples-ci
examples-ci: examples clean-examples
