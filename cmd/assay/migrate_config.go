package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

type renameOp struct {
	from string
	to   string
	kind string // "file" or "dir"
}

func newMigrateConfigCmd() *cobra.Command {
	var (
		fromName string
		toName   string
	)

	cmd := &cobra.Command{
		Use:   "migrate-config",
		Short: "Rename project config files to match a new binary name",
		Long: `Rename .assay.yaml, .assay/, ASSAY.md files, and bundled
agent files to match a new binary name. Run this once in a consumer project
after upgrading to a renamed binary.

Artifacts renamed:
  .assay.yaml              →  .<newname>.yaml
  .assay/                  →  .<newname>/
  ASSAY.md                 →  <NEWNAME>.md  (root and all subdirectories)
  .claude/agents/assay-*.md  →  .claude/agents/<newname>-*.md

Examples:
  assay migrate-config --to myname
  assay migrate-config --to myname --dry-run`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if toName == "" {
				return errors.New("--to is required: provide the new binary name")
			}
			if fromName == toName {
				return errors.New("--from and --to must be different")
			}
			return runMigrateConfig(fromName, toName)
		},
	}

	cmd.Flags().StringVar(&fromName, "from", "assay", "current name to migrate from")
	cmd.Flags().StringVar(&toName, "to", "", "new binary name to migrate to")

	return cmd
}

func runMigrateConfig(fromName, toName string) error {
	root, err := os.Getwd()
	if err != nil {
		return err
	}

	ops := collectRenameOps(root, fromName, toName)
	if len(ops) == 0 {
		fmt.Printf("No %s config artifacts found under %s\n", fromName, root)
		return nil
	}

	for _, op := range ops {
		rel := func(p string) string {
			r, _ := filepath.Rel(root, p)
			return r
		}
		if dryRun {
			fmt.Printf("  [dry-run] %s  %s  →  %s\n", op.kind, rel(op.from), rel(op.to))
			continue
		}
		if err := os.Rename(op.from, op.to); err != nil {
			return fmt.Errorf("rename %s → %s: %w", rel(op.from), rel(op.to), err)
		}
		fmt.Printf("  ✓ %s  %s  →  %s\n", op.kind, rel(op.from), rel(op.to))
	}

	if !dryRun {
		fmt.Printf("\nMigration complete. Update any references to %q inside your config files and agent descriptions.\n", fromName)
	}
	return nil
}

func collectRenameOps(root, fromName, toName string) []renameOp {
	var ops []renameOp

	// .<fromName>.yaml → .<toName>.yaml
	ops = appendIfExists(ops, renameOp{
		from: filepath.Join(root, "."+fromName+".yaml"),
		to:   filepath.Join(root, "."+toName+".yaml"),
		kind: "file",
	})

	// .<fromName>/ → .<toName>/
	ops = appendIfExists(ops, renameOp{
		from: filepath.Join(root, "."+fromName),
		to:   filepath.Join(root, "."+toName),
		kind: "dir ",
	})

	// ASSAY.md → <NEWNAME>.md in root and all subdirs
	upperFrom := strings.ToUpper(fromName) + ".md"
	upperTo := strings.ToUpper(toName) + ".md"
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		if d.Name() == upperFrom {
			ops = append(ops, renameOp{
				from: path,
				to:   filepath.Join(filepath.Dir(path), upperTo),
				kind: "file",
			})
		}
		return nil
	})

	// .claude/agents/<fromName>-*.md → .claude/agents/<toName>-*.md
	agentsDir := filepath.Join(root, ".claude", "agents")
	entries, _ := os.ReadDir(agentsDir)
	for _, e := range entries {
		if e.IsDir() || !strings.HasPrefix(e.Name(), fromName+"-") || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		newName := toName + e.Name()[len(fromName):]
		ops = append(ops, renameOp{
			from: filepath.Join(agentsDir, e.Name()),
			to:   filepath.Join(agentsDir, newName),
			kind: "file",
		})
	}

	return ops
}

func appendIfExists(ops []renameOp, op renameOp) []renameOp {
	if _, err := os.Stat(op.from); err == nil {
		return append(ops, op)
	}
	return ops
}
