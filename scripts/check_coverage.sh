#!/usr/bin/env bash
# check_coverage.sh — coverage fitness function for Assay CI.
#
# Enforces two constraints:
#   1. Overall project coverage must be >= TOTAL_THRESHOLD (default 70%).
#   2. Each "core" package must stay >= CORE_THRESHOLD (default 85%).
#
# Usage: ./scripts/check_coverage.sh [coverage.out]
# If no profile path is provided, the script runs the test suite itself.

set -euo pipefail

PROFILE="${1:-coverage.out}"
TOTAL_THRESHOLD="${TOTAL_THRESHOLD:-70}"
CORE_THRESHOLD="${CORE_THRESHOLD:-85}"

# Packages that have received significant test investment — held to the higher bar.
CORE_PACKAGES=(
  "github.com/orieken/assay/internal/config"
  "github.com/orieken/assay/internal/analysis"
  "github.com/orieken/assay/internal/llm$"
  "github.com/orieken/assay/internal/llm/anthropic"
  "github.com/orieken/assay/internal/llm/openai"
  "github.com/orieken/assay/internal/llm/ollama"
  "github.com/orieken/assay/internal/llm/factory"
  "github.com/orieken/assay/internal/migration"
  "github.com/orieken/assay/internal/projectknowledge"
)

# ── Generate profile if not supplied ─────────────────────────────────────────

if [[ ! -f "$PROFILE" ]]; then
  echo "No coverage profile found at '$PROFILE'. Running tests..."
  CGO_ENABLED=1 go test -coverprofile="$PROFILE" -covermode=atomic ./... -timeout 120s
fi

# ── Total coverage gate ───────────────────────────────────────────────────────

ACTUAL_TOTAL=$(go tool cover -func="$PROFILE" | awk '/^total:/{gsub(/%/,""); print $3}')
echo "Total coverage: ${ACTUAL_TOTAL}%  (threshold: ${TOTAL_THRESHOLD}%)"

FAILED=0

if awk -v a="$ACTUAL_TOTAL" -v t="$TOTAL_THRESHOLD" \
    'BEGIN { exit (a+0 >= t+0) }'; then
  echo "FAIL: total coverage ${ACTUAL_TOTAL}% is below threshold ${TOTAL_THRESHOLD}%"
  FAILED=1
fi

# ── Per-package gate on core packages ────────────────────────────────────────

echo ""
echo "Core-package coverage (threshold: ${CORE_THRESHOLD}%):"

# Build a per-package summary from the profile.
# go tool cover -func emits: <file>:<line>: <func> <pct>%
# The last line per package is the package summary — we want only the
# "total" line for each package, which looks like:
#   github.com/orieken/assay/internal/config/loader.go:30:	loadYAML	100.0%
# We'll compute per-package averages ourselves from the profile.

# Use `go test -cover ./...` output for per-package numbers; but that
# requires running tests again. Instead, parse the profile directly:
# profile lines are: <file>:<start>,<end> <stmts> <count>
# Build pkg→(total, covered) sums.

check_pkg() {
  local pkg="$1"
  # Grep lines belonging to this package from the profile (skip the "mode:" header).
  local lines
  lines=$(grep "^${pkg}/" "$PROFILE" 2>/dev/null || true)
  if [[ -z "$lines" ]]; then
    echo "  SKIP (no data): ${pkg}"
    return
  fi

  local total covered
  read -r total covered < <(echo "$lines" | awk -F'[,: ]' '
    {
      stmts = $(NF-1)
      count = $NF
      total += stmts
      if (count > 0) covered += stmts
    }
    END { print total, covered }
  ')

  if [[ "$total" -eq 0 ]]; then
    echo "  SKIP (no statements): ${pkg}"
    return
  fi

  local pct
  pct=$(awk -v c="$covered" -v t="$total" 'BEGIN { printf "%.1f", c*100/t }')
  if awk -v p="$pct" -v th="$CORE_THRESHOLD" 'BEGIN { exit (p+0 >= th+0) }'; then
    echo "  FAIL: ${pkg}  ${pct}% < ${CORE_THRESHOLD}%"
    FAILED=1
  else
    echo "  PASS: ${pkg}  ${pct}%"
  fi
}

for pkg in "${CORE_PACKAGES[@]}"; do
  # Strip regex anchor added to disambiguate llm$ from llm/anthropic etc.
  check_pkg "${pkg%$}"
done

# ── Result ────────────────────────────────────────────────────────────────────

echo ""
if [[ "$FAILED" -ne 0 ]]; then
  echo "Coverage gate FAILED."
  exit 1
fi
echo "Coverage gate PASSED."
