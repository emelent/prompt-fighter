// Package docs embeds the player-facing documentation.
package docs

import _ "embed"

// PlayerGuide is docs/PLAYER_GUIDE.md.
//
//go:embed PLAYER_GUIDE.md
var PlayerGuide string

// Rules is docs/RULES.md.
//
//go:embed RULES.md
var Rules string
