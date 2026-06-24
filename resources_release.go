//go:build release

package main

import (
	"embed"
	"io/fs"
)

// webDist embeds the built web UI. The frontend (web/dist) must be built before
// compiling with the "release" build tag, otherwise the build fails with
// "no matching files found".
//
//go:embed all:web/dist
var webDist embed.FS

func init() {
	sub, err := fs.Sub(webDist, "web/dist")
	if err != nil {
		panic(err)
	}
	webFS = sub
}
