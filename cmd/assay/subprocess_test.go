package main

import (
	"os"
	"strings"
	"testing"
)

// init seeds the driver registry for all package-main tests.
// main() is not called by the test runner, so we fill the gap here.
func init() {
	initRegistry()
}

// TestSubprocessMain is the TestHelperProcess entry point used by cli_test.go.
// When GO_ASSAY_SUBPROCESS=1 is set, it reconstructs os.Args from
// GO_ASSAY_ARGS and drives the cobra command tree in-process.
//
// Benefit: running the test binary as its own subprocess (os.Args[0]) means
// coverage counters are in the same instrumented binary and can be collected
// via GOCOVERDIR without building a separate, uninstrumented binary.
func TestSubprocessMain(t *testing.T) {
	if os.Getenv("GO_ASSAY_SUBPROCESS") != "1" {
		return // normal test run — skip
	}

	rawArgs := os.Getenv("GO_ASSAY_ARGS")
	if rawArgs != "" {
		os.Args = append([]string{"assay"}, strings.Split(rawArgs, "\x1e")...)
	}

	// execute() calls os.Exit(1) on cobra error, otherwise returns normally.
	execute()
	os.Exit(0)
}
