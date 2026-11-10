// Package web embeds the built replay viewer (Phaser 3 + TypeScript + Vite).
//
// The sources live in web/src; `npm run build` (in web/) writes the bundle to
// web/dist, which is committed so that `go build` needs no Node toolchain.
package web

import (
	"embed"
	"io/fs"
)

//go:embed dist
var dist embed.FS

// FS is the built viewer: index.html plus its hashed .js/.css assets, at the
// root of the filesystem.
var FS fs.FS = mustSub(dist, "dist")

func mustSub(f fs.FS, dir string) fs.FS {
	sub, err := fs.Sub(f, dir)
	if err != nil {
		panic(err)
	}
	return sub
}
