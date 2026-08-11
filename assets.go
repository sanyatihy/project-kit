// Package assets embeds the templates `kit init` writes into a project.
//
// It lives at the repo root so the templates stay visible and editable there,
// rather than buried next to the code that reads them.
package assets

import "embed"

//go:embed all:templates
var FS embed.FS
