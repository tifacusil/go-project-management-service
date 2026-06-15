// Package migrations embeds the goose SQL files so the binary carries its own
// schema — no migrations dir needed at runtime (works in distroless).
package migrations

import "embed"

// FS holds every .sql migration, applied by goose at startup (goose = fuente
// de verdad del schema; CLAUDE.md §2.4).
//
//go:embed *.sql
var FS embed.FS
