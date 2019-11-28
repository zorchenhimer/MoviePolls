package moviepoll

import (
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"
	//"time"
)

type Options struct {
	Listen string // eg, "127.0.0.1:8080" or ":8080" (defaults to 0.0.0.0:8080)
	Debug  bool   // debug logging to console
}

type Server struct {
	templates map[string]*template.Template
	s         *http.Server
	debug     bool // turns on debug things (eg, reloading templates on each page request)
	data      DataConnector
}

func NewServer(options Options) (*Server, error) {
	if options.Listen == "" {
		options.Listen = ":8080"
	}

	data, err := NewJsonConnector("db/data.json")
	if err != nil {
		return nil, fmt.Errorf("Unable to load json data: %v", err)
	}

	hs := &http.Server{
		Addr: options.Listen,
	}

	server := &Server{
		debug: options.Debug,
		data: data,
	}

	mux := http.NewServeMux()
	mux.Handle("/api/", apiHandler{})
	mux.HandleFunc("/", server.handler_Root)
	mux.HandleFunc("/movie/", server.handler_Movie)
	mux.HandleFunc("/data/", server.handler_Data)

	hs.Handler = mux
	server.s = hs

	err = server.registerTemplates()
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
	data := dataCycleOther{
		dataPageBase: dataPageBase{
			PageTitle: "Current Cycle",
		},

		Cycle: &Cycle{},//s.data.GetCurrentCycle(),
		Movies: s.data.GetActiveMovies(),
	}

	fmt.Printf("cycle: %v\n", data.Cycle)

	if err := s.executeTemplate(w, "cyclevotes", data); err != nil {
		fmt.Printf("Error rendering template: %v\n", err)
		//http.Error(w, fmt.Sprintf("%v", err), http.StatusInternalServerError)
	}
}
func (s *Server) handler_Movie(w http.ResponseWriter, r *http.Request) {
	var movieId int
	var command string
	n, err := fmt.Sscanf(r.URL.String(), "/movie/%d/%s", &movieId, &command)
	if err != nil && n == 0 {
		dataError := dataMovieError{
			dataPageBase: dataPageBase{PageTitle: "Error"},
			ErrorMessage: "Missing movie ID",
		}

		if err := s.executeTemplate(w, "movieError", dataError); err != nil {
			http.Error(w, fmt.Sprintf("%v", err), http.StatusInternalServerError)
			fmt.Println(err)
		}
		return
	}

	movie, err := s.data.GetMovie(movieId)
	if err != nil {
		dataError := dataMovieError{
			dataPageBase: dataPageBase{PageTitle: "Error"},
			ErrorMessage: "Movie not found",
		}

		if err := s.executeTemplate(w, "movieError", dataError); err != nil {
			http.Error(w, fmt.Sprintf("%v", err), http.StatusInternalServerError)
			fmt.Println(err)
		}
		return
	}

	data := dataMovieInfo{
		dataPageBase: dataPageBase{PageTitle: movie.Name},
		Movie: movie,
	}

	if err := s.executeTemplate(w, "movieinfo", data); err != nil {
		http.Error(w, fmt.Sprintf("%v", err), http.StatusInternalServerError)
	}
}
