package dash

import (
	"embed"
	"io/fs"
)

//go:embed dist dist/* dist/assets dist/assets/*
var distFS embed.FS

//go:embed public-dist public-dist/* public-dist/assets public-dist/assets/*
var publicDistFS embed.FS

//go:embed todash.html
var toDashHTML string

//go:embed todash.css
var toDashCSS []byte

func DistFS() (fs.FS, error) {
	return fs.Sub(distFS, "dist")
}

// PublicDistFS exposes the public-only SPA bundle. It has no admin route or
// GraphQL imports and is served exclusively from the public relay listener.
func PublicDistFS() (fs.FS, error) {
	return fs.Sub(publicDistFS, "public-dist")
}

func ToDashHTML() string {
	return toDashHTML
}

func ToDashCSS() []byte {
	return toDashCSS
}
