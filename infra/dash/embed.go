package dash

import (
	"embed"
	"io/fs"
)

//go:embed dist dist/* dist/assets dist/assets/*
var distFS embed.FS

//go:embed todash.html
var toDashHTML string

//go:embed todash.css
var toDashCSS []byte

func DistFS() (fs.FS, error) {
	return fs.Sub(distFS, "dist")
}

func ToDashHTML() string {
	return toDashHTML
}

func ToDashCSS() []byte {
	return toDashCSS
}
