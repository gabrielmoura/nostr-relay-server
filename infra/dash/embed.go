package dash

import (
	"embed"
	"io/fs"
)

//go:embed dist dist/* dist/assets dist/assets/*
var distFS embed.FS

func DistFS() (fs.FS, error) {
	return fs.Sub(distFS, "dist")
}
