package generation

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Verifier checks whether a generated test file compiles / parses correctly.
// Implementations are language-specific and run after the Executor writes a file.
// A non-nil error means the generated file is syntactically or semantically
// invalid and the caller should warn the user — but not delete the file.
type Verifier interface {
	Verify(testFilePath string) error
}

// VerifierFor returns the Verifier for the given language, or nil when no
// compile check is available (Java, C# require full build toolchains that are
// not guaranteed to be present on all developer machines).
func VerifierFor(language string) Verifier {
	switch language {
	case "go":
		return GoVerifier{}
	case "typescript", "javascript":
		return TypeScriptVerifier{}
	case "python":
		return PythonVerifier{}
	}
	return nil
}

// GoVerifier runs `go vet` on the package containing the test file.
// go vet catches type errors and common mistakes without a full build.
type GoVerifier struct{}

func (v GoVerifier) Verify(testFilePath string) error {
	pkgDir := filepath.Dir(testFilePath)
	cmd := exec.Command("go", "vet", "./...")
	cmd.Dir = pkgDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("go vet: %w\n%s", err, trimOutput(out))
	}
	return nil
}

// TypeScriptVerifier runs `tsc --noEmit` from the directory containing
// the test file, picking up the nearest tsconfig.json automatically.
// Falls back gracefully when tsc is not installed.
type TypeScriptVerifier struct{}

func (v TypeScriptVerifier) Verify(testFilePath string) error {
	tsc, err := exec.LookPath("tsc")
	if err != nil {
		return nil //nolint:nilerr // tsc not installed — skip verification silently
	}

	// Walk up to find the tsconfig.json so tsc gets the right project settings.
	tsconfig := findTsConfig(filepath.Dir(testFilePath))
	args := []string{"--noEmit"}
	if tsconfig != "" {
		args = append(args, "--project", tsconfig)
	} else {
		// No tsconfig found — check just the single file.
		args = append(args, testFilePath)
	}

	cmd := exec.Command(tsc, args...)
	cmd.Dir = filepath.Dir(testFilePath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("tsc --noEmit: %w\n%s", err, trimOutput(out))
	}
	return nil
}

// findTsConfig walks up from dir looking for a tsconfig.json.
// Returns empty string when none is found within 5 levels.
func findTsConfig(dir string) string {
	const maxLevels = 5
	cur := dir
	for i := 0; i < maxLevels; i++ {
		candidate := filepath.Join(cur, "tsconfig.json")
		if fileExists(candidate) {
			return candidate
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			break
		}
		cur = parent
	}
	return ""
}

// PythonVerifier uses `python -m py_compile` to check syntax.
// This catches parse errors without executing the file.
type PythonVerifier struct{}

func (v PythonVerifier) Verify(testFilePath string) error {
	// Try python3 first, fall back to python.
	python := "python3"
	if _, err := exec.LookPath(python); err != nil {
		python = "python"
		if _, err2 := exec.LookPath(python); err2 != nil {
			return nil //nolint:nilerr // no python interpreter — skip verification silently
		}
	}

	cmd := exec.Command(python, "-m", "py_compile", testFilePath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("py_compile: %w\n%s", err, trimOutput(out))
	}
	return nil
}

// trimOutput trims and truncates command output for error messages.
func trimOutput(out []byte) string {
	s := strings.TrimSpace(string(bytes.TrimSpace(out)))
	const maxChars = 400
	if len(s) > maxChars {
		return s[:maxChars] + "…"
	}
	return s
}

// fileExists is a lightweight existence check used within this package.
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return !errors.Is(err, os.ErrNotExist)
}
