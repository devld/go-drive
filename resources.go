package main

import (
	"embed"
	"io/fs"
)

// langFiles embeds the backend i18n translations.
//
//go:embed docs/lang/*.yml
var langFiles embed.FS

var webFS fs.FS = emptyFS{}

// webResourceFS returns the embedded web UI rooted at the dist directory.
func webResourceFS() fs.FS {
	return webFS
}

// langResourceFS returns the embedded i18n files rooted at the lang directory.
func langResourceFS() fs.FS {
	sub, err := fs.Sub(langFiles, "docs/lang")
	if err != nil {
		panic(err)
	}
	return sub
}

type emptyFS struct{}

func (emptyFS) Open(string) (fs.File, error) {
	return nil, fs.ErrNotExist
}
