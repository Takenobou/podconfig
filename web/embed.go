package web

import (
	"embed"
	"io/fs"
)

//go:embed static/*
//go:embed templates/*
var files embed.FS

// Static returns a sub filesystem for static assets.
func Static() (fs.FS, error) {
	return fs.Sub(files, "static")
}

// Templates returns a sub filesystem for HTML templates.
func Templates() (fs.FS, error) {
	return fs.Sub(files, "templates")
}
