// Package migrations embeds the SQL migrations of the metrics database.
package migrations

import "embed"

// FS holds the *.sql migration files.
//
//go:embed *.sql
var FS embed.FS
