package web

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed all:dist
var staticFiles embed.FS

// Handler returns an http.Handler that serves the embedded React SPA.
// All non-file requests fall back to index.html for client-side routing.
func Handler() http.Handler {
	sub, err := fs.Sub(staticFiles, "dist")
	if err != nil {
		panic("web: failed to sub dist: " + err.Error())
	}
	fileServer := http.FileServer(http.FS(sub))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try to serve the exact file; fall back to index.html for SPA routes.
		f, err := sub.Open(r.URL.Path[1:])
		if err != nil || r.URL.Path == "/" {
			// Serve index.html for all unknown paths.
			r2 := *r
			r2.URL.Path = "/"
			fileServer.ServeHTTP(w, &r2)
			return
		}
		f.Close()
		fileServer.ServeHTTP(w, r)
	})
}
