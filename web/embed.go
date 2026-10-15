// Package web embeds the static frontend (HTML/CSS/JS) so the compiled
// server binary can serve the UI without needing the files on disk.
package web

import "embed"

//go:embed static
var FS embed.FS
