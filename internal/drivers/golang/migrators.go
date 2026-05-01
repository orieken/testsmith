package golang

import "github.com/orieken/testsmith/internal/domain"

// Go test migration is not supported via regex — the AST-level rewrites
// required (e.g. testify → stdlib assertion rewrites) are too structurally
// complex to express as text substitutions without producing incorrect code.
var goMigrators []domain.Migrator
