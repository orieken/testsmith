// Package agents embeds the bundled Claude Code agent files that testsmith
// can write into a consumer project's .claude/agents/ directory via
// testsmith init --with-agents.
package agents

import (
	"embed"
	"io/fs"
)

//go:embed files/*.md
var agentFS embed.FS

// All returns a map of filename → content for every bundled agent file.
func All() (map[string][]byte, error) {
	out := make(map[string][]byte)
	err := fs.WalkDir(agentFS, "files", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, readErr := agentFS.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		out[d.Name()] = data
		return nil
	})
	return out, err
}
