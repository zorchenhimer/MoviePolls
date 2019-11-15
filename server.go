package moviepoll

import (
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"
)

type Options struct {
	Listen string // eg, "127.0.0.1:8080" or ":8080" (defaults to 0.0.0.0:8080)
	Debug  bool   // debug logging to console
}

type Server struct {
	templates map[string]*template.Template
	s         *http.Server
	debug     bool // turns on debug things (eg, reloading templates on each page request)
}

func NewServer(options Options) (*Server, error) {
	if options.Listen == "" {
		options.Listen = ":8080"
	}

	hs := &http.Server{
		Addr: options.Listen,
	}

	server := &Server{
		debug: options.Debug,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", server.handler_Root)
	mux.Handle("/api/", apiHandler{})
	mux.HandleFunc("/data/", server.handler_Data)

	hs.Handler = mux
	server.s = hs

	err := server.registerTemplates()
	if err != nil {
		return nil, err
	}

	return server, nil
}

func (server *Server) Run() error {
	return server.s.ListenAndServe()
}

func (s *Server) handler_Data(w http.ResponseWriter, r *http.Request) {
	file := "data/" + filepath.Base(r.URL.Path)
	fmt.Printf("Attempting to serve file %q\n", file)
	http.ServeFile(w, r, file)
}

func (s *Server) handler_Root(w http.ResponseWriter, r *http.Request) {
	data := dataMovieInfo{
		PageTitle:   "Movie Info - Some Movie, IDK",
		Description: "A shitty movie about some sombies or something.  You figure it out.",
		MovieTitle:  "Zombie Butts",
		MoviePoster: "data/poster.jpg",
		AddedBy:     "Zorchenhimer",
		Votes: []string{
			"Zorchenhimer",
			"Mia",
			"Someone else",
		},
	}

	if err := s.executeTemplate(w, "movieinfo", data); err != nil {
		http.Error(w, fmt.Sprintf("%v", err), http.StatusInternalServerError)
	}
}
