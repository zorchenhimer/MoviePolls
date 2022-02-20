package web

import (
	"net/http"
	"path/filepath"
	"strings"

	"github.com/zorchenhimer/MoviePolls/models"
)

func (s *webServer) handlerStatic(w http.ResponseWriter, r *http.Request) {
	file := strings.TrimLeft(filepath.Clean("/"+r.URL.Path), "/\\")
	file = "web/" + file
	if s.debug {
		s.l.Info("Attempting to serve file %q", file)
	}
	http.ServeFile(w, r, file)
}

func (s *webServer) handlerPosters(w http.ResponseWriter, r *http.Request) {
	file := strings.TrimLeft(filepath.Clean("/"+r.URL.Path), "/\\")
	if s.debug {
		s.l.Info("Attempting to serve file %q", file)
	}

	// Dirty but might work out fine
	if file != "/posters" {
		http.ServeFile(w, r, file)
	} else {
		http.Error(w, "Nothing to see here. Go away!", http.StatusForbidden)
	}
}

func (s *webServer) handlerFavicon(w http.ResponseWriter, r *http.Request) {
	if models.FileExists("data/favicon.ico") {
		http.ServeFile(w, r, "data/favicon.ico")
	} else {
		http.NotFound(w, r)
	}
}
