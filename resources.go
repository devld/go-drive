package main

import (
	"embed"
	"io/fs"
)

// webDist embeds the built web UI. The frontend (web/dist) must be built before
// compiling, otherwise the build fails with "no matching files found".
//
//go:embed all:web/dist
var webDist embed.FS

// langFiles embeds the backend i18n translations.
//
//go:embed docs/lang/*.yml
var langFiles embed.FS

// webResourceFS returns the embedded web UI rooted at the dist directory.
func webResourceFS() fs.FS {
	sub, err := fs.Sub(webDist, "web/dist")
	if err != nil {
		panic(err)
	}
	return sub
}

// langResourceFS returns the embedded i18n files rooted at the lang directory.
func langResourceFS() fs.FS {
	sub, err := fs.Sub(langFiles, "docs/lang")
	if err != nil {
		panic(err)
	}
	return sub
}
