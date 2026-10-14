// Package migrations embeds the SQL migrations so the binary can migrate on start.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
