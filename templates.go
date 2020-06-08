package moviepoll

import (
	"fmt"
	"html/template"
	"net/http"

	"github.com/zorchenhimer/MoviePolls/common"
)

const TEMPLATE_DIR = "templates/"
const TEMPLATE_BASE = TEMPLATE_DIR + "base.html"

//var templates map[string]*template.Template

// templateDefs is static throughout the life of the server process
var templateDefs map[string][]string = map[string][]string{
	"movieinfo":   []string{"movie-info.html"},
	"cyclevotes":  []string{"cycle.html"},
	"movieError":  []string{"movie-error.html"},
	"simplelogin": []string{"plain-login.html"},
	"addmovie":    []string{"add-movie.html"},
	"account":     []string{"account.html"},
	"newaccount":  []string{"newaccount.html"},
	"error":       []string{"error.html"},
	"history":     []string{"history.html"},
	"auth":        []string{"auth.html"},

	"adminHome":     []string{"admin/base.html", "admin/home.html"},
	"adminConfig":   []string{"admin/base.html", "admin/config.html"},
	"adminUsers":    []string{"admin/base.html", "admin/users.html"},
	"adminUserEdit": []string{"admin/base.html", "admin/user-edit.html"},
	"adminCycles":   []string{"admin/base.html", "admin/cycles.html"},
	"adminEndCycle": []string{"admin/base.html", "admin/endcycle.html"},
	"adminMovies":   []string{"admin/base.html", "admin/movies.html"},
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

	if err := t.Execute(w, data); err != nil {
		return fmt.Errorf("[%s] %v", key, err)
	}

	return nil
}

func (s *Server) newPageBase(title string, w http.ResponseWriter, r *http.Request) dataPageBase {
	cycle, err := s.data.GetCurrentCycle()
	if err != nil {
		fmt.Printf("[newPageBase] Unable to get current sessinon: %v\n", err)
	}

	return dataPageBase{
		PageTitle:    title,
		User:         s.getSessionUser(w, r),
		CurrentCycle: cycle,
	}
}

type dataPageBase struct {
	PageTitle string

	User         *common.User
	CurrentCycle *common.Cycle
}

type dataMovieInfo struct {
	dataPageBase

	Movie *common.Movie
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
	ErrAutofill    bool

	// Values for input if error
	ValTitle       string
	ValDescription string
	ValLinks       string
	//ValPoster      bool
}

func (d dataAddMovie) isError() bool {
	return d.ErrTitle || d.ErrDescription || d.ErrLinks || d.ErrPoster || d.ErrAutofill
}

type dataAccount struct {
	dataPageBase

	CurrentVotes   []*common.Movie
	TotalVotes     int
	AvailableVotes int

	SuccessMessage string

	PassError   []string
	NotifyError []string
	EmailError  []string

	ErrCurrentPass bool
	ErrNewPass     bool
	ErrEmail       bool
}

func (a dataAccount) IsErrored() bool {
	return a.ErrCurrentPass || a.ErrNewPass || a.ErrEmail
}

type dataNewAccount struct {
	dataPageBase

	ErrorMessage []string
	ErrName      bool
	ErrPass      bool
	ErrEmail     bool

	ValName           string
	ValEmail          string
	ValNotifyEnd      bool
	ValNotifySelected bool
}

type dataError struct {
	dataPageBase

	Message string
	Code    int
}
