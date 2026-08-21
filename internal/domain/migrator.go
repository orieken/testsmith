package domain

// Migrator transforms the content of an existing test file from one framework
// convention to another (e.g. jest → vitest, junit4 → junit5).
type Migrator interface {
	// From returns the source framework or mock-library name (e.g. "jest", "junit4").
	From() string
	// To returns the target framework or mock-library name (e.g. "vitest", "junit5").
	To() string
	// MigrateFile applies all transformation rules to the given file content and
	// returns the modified result. The original is never written to disk here.
	MigrateFile(content string) (string, error)
}
