package domain

import "errors"

var (
	// ErrProjectNotFound is returned by DetectProject when no root marker is found.
	ErrProjectNotFound = errors.New("could not detect project root; ensure a marker file (go.mod, pyproject.toml, package.json, pom.xml, Gemfile, .git) exists")

	// ErrNoDriverForFile is returned when no registered driver claims a file's extension.
	ErrNoDriverForFile = errors.New("no language driver registered for this file type")

	// ErrNoDriverForLanguage is returned when a language name is specified but unregistered.
	ErrNoDriverForLanguage = errors.New("no language driver registered for this language")

	// ErrSourceParse is returned when a driver cannot parse a source file.
	ErrSourceParse = errors.New("failed to parse source file")

	// ErrTestFileExists is returned when a test file already exists and overwrite is disabled.
	ErrTestFileExists = errors.New("test file already exists; use --overwrite to regenerate")

	// ErrLLMUnavailable is returned when LLM generation is requested but no provider is configured.
	ErrLLMUnavailable = errors.New("LLM provider not configured; set the api_key_env_var or disable llm.enabled")
)

// ParseError wraps a parse failure with source location context.
type ParseError struct {
	Path   string
	Line   int
	Reason string
}

func (e *ParseError) Error() string {
	return "parse error in " + e.Path + " at line " + itoa(e.Line) + ": " + e.Reason
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	buf := [20]byte{}
	pos := len(buf)
	for n > 0 {
		pos--
		buf[pos] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[pos:])
}
