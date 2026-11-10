// Package scripts embeds the bundled sample fight scripts (*.pf).
package scripts

import "embed"

// FS holds every sample bot, at the root of the filesystem as "<name>.pf".
//
//go:embed *.pf
var FS embed.FS
