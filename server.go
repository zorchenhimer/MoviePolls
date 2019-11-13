package moviepoll

import (
    "fmt"
    "net/http"
    "html/template"
)

type Options struct {
    Listen string // eg, "127.0.0.1:8080" or ":8080" (defaults to 0.0.0.0:80)
    Debug bool  // debug logging to console
}

type Server struct {
    templates map[string]*template.Template
    s *http.Server
}

func NewServer(options Options) *Server {
    hs := &http.Server{
        Addr: options.Listen,
    }

    server := &Server{}

    mux := http.NewServeMux()
    mux.HandleFunc("/", server.handler_Root)
    mux.Handle("/api/", apiHandler{})

    hs.Handler = mux
    server.s = hs

    return server
}

func (s *Server) handler_Root(w http.ResponseWriter, r *http.Request) {
    if err := s.executeTemplate(w, "movieinfo", nil); err != nil {
        http.Error(w, fmt.Sprintf("%v", err), http.StatusInternalServerError)
    }
}

