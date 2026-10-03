// Package web serves the built frontend from inside the binary.
package web

import (
	"embed"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

// The all: prefix matters: plain embed skips files beginning with _ or .,
// which would drop some Vite assets.
//
//go:embed all:dist
var dist embed.FS

// Handler serves the SPA, falling back to index.html for unknown paths so
// client-side routes survive a refresh.
func Handler() (http.Handler, error) {
	sub, err := fs.Sub(dist, "dist")
	if err != nil {
		return nil, err
	}
	files := http.FileServer(http.FS(sub))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if name == "" {
			name = "index.html"
		}
		if _, err := fs.Stat(sub, name); err != nil {
			if _, err := fs.Stat(sub, "index.html"); err != nil {
				// A Go build without the frontend (go build before
				// ./build.sh): say so instead of a bare 404.
				http.Error(w, "The web UI isn't built into this binary. Build with ./build.sh, build.ps1 or build.cmd.", http.StatusNotFound)
				return
			}
			r = r.Clone(r.Context())
			r.URL.Path = "/"
		}
		files.ServeHTTP(w, r)
	}), nil
}
