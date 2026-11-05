package server

import (
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strings"

	"github.com/uiautodev/weektodo/frontend"
)

func (s *Server) handleStatic(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/api/") || r.URL.Path == "/api" {
		writeError(w, http.StatusNotFound, "not_found", "not found")
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	sub, err := fs.Sub(frontend.BuildFS, "build")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "assets_unavailable", "frontend assets unavailable")
		return
	}
	file := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
	if file == "" || file == "." {
		file = "index.html"
	}
	if strings.HasPrefix(file, "../") {
		writeError(w, http.StatusNotFound, "not_found", "not found")
		return
	}
	data, err := fs.ReadFile(sub, file)
	if err != nil {
		if path.Ext(file) != "" {
			writeError(w, http.StatusNotFound, "not_found", "not found")
			return
		}
		file = "index.html"
		data, err = fs.ReadFile(sub, file)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "frontend_not_built", "frontend is not built")
			return
		}
	}
	if contentType := mime.TypeByExtension(path.Ext(file)); contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}
	if file == "index.html" {
		w.Header().Set("Cache-Control", "no-cache")
	} else if strings.HasPrefix(file, "_app/") {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	} else {
		w.Header().Set("Cache-Control", "public, max-age=3600")
	}
	_, _ = w.Write(data)
}
