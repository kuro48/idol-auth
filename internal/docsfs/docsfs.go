// Package docsfs embeds the pre-built Scalar API reference site.
// The dist directory is populated by "make docs" or the Docker build stage.
package docsfs

import (
	"embed"
	"io/fs"
)

//go:embed dist
var embedded embed.FS

// FS returns an fs.FS rooted at the dist directory.
func FS() fs.FS {
	sub, err := fs.Sub(embedded, "dist")
	if err != nil {
		panic("docsfs: failed to sub into dist: " + err.Error())
	}
	return sub
}
