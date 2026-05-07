package server

import (
	"io/fs"
	"net/http"
)

// staticHandler returns an http.Handler that serves files from the given FS
// under the /static/ path prefix.
func staticHandler(frontendFS fs.FS) http.Handler {
	return http.StripPrefix("/static/", http.FileServer(http.FS(frontendFS)))
}
