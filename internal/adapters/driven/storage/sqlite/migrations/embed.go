// Package migrations embeds SQL migration files for the SQLite store.
package migrations

import "embed"

// FS contains all SQL migration files embedded at compile time.
//
//go:embed *.sql
var FS embed.FS
