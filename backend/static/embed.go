package static

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var dist embed.FS

// FS returns the embedded frontend build rooted at dist/.
func FS() (fs.FS, error) {
	return fs.Sub(dist, "dist")
}
