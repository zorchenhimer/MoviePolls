package moviepoll

import (
	"fmt"
	"html/template"
	"net/http"
	//"time"
)

const TEMPLATE_DIR = "templates/"
const TEMPLATE_BASE = TEMPLATE_DIR + "base.html"

//var templates map[string]*template.Template

// templateDefs is static throughout the life of the server process
var templateDefs map[string][]string = map[string][]string{
	"movieinfo":   []string{"movie-info.html"},
	"cyclevotes":  []string{"cycle.html", "vote.html"},
	"movieError":  []string{"movie-error.html"},
	"simplelogin": []string{"plain-login.html"},
	"addmovie":    []string{"add-movie.html"},
	"account":     []string{"account.html"},
}

func (s *Server) registerTemplates() error {
	s.templates = make(map[string]*template.Template)

	for key, files := range templateDefs {
		fpth := []string{TEMPLATE_BASE}
		for _, f := range files {
			fpth = append(fpth, TEMPLATE_DIR+f)
		}

		t, err := template.ParseFiles(fpth...)
		if err != nil {
			return fmt.Errorf("Error parsing template %s: %v", fpth, err)
		}

		fmt.Printf("Registering template %q\n", key)
		s.templates[key] = t
	}
	return nil
}

func (s *Server) executeTemplate(w http.ResponseWriter, key string, data interface{}) error {
	// for deugging only
	if s.debug {
		err := s.registerTemplates()
		if err != nil {
			return err
		}
	}

	t, ok := s.templates[key]
	if !ok {
		return fmt.Errorf("Template with key %q does not exist", key)
	}

	return t.Execute(w, data)
}

func (s *Server) newPageBase(title string, r *http.Request) dataPageBase {
	return dataPageBase{
		PageTitle: title,
		IsAuthed:  s.getSessionBool("authed", r),
		IsAdmin:   s.getSessionBool("admin", r),
	}
}

type dataPageBase struct {
	PageTitle string
	IsAuthed  bool
	IsAdmin   bool
}

type dataCycleOther struct {
	dataPageBase

	Cycle  *Cycle
	Movies []*Movie
}

type dataMovieInfo struct {
	dataPageBase

	Movie *Movie
}

type dataMovieError struct {
	dataPageBase
	ErrorMessage string
}

type dataLoginForm struct {
	dataPageBase
	ErrorMessage string
	Authed       bool
}

type dataAddMovie struct {
	dataPageBase
	ErrorMessage []string

	// Offending input
	ErrTitle       bool
	ErrDescription bool
	ErrLinks       bool
	ErrPoster      bool

	// Values for input if error
	ValTitle       string
	ValDescription string
	ValLinks       string
	//ValPoster      bool
}

func (d dataAddMovie) isError() bool {
	return d.ErrTitle || d.ErrDescription || d.ErrLinks || d.ErrPoster
}
