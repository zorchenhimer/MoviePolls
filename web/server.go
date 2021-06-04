package web

import (
	"context"
	"fmt"
	"html/template"

	//"io/ioutil"
	"net/http"
	"os"

	//"path/filepath"
	//"regexp"
	//"strconv"
	//"strings"

	"github.com/gorilla/sessions"

	"github.com/zorchenhimer/MoviePolls/logic"
	"github.com/zorchenhimer/MoviePolls/models"
	//"github.com/zorchenhimer/MoviePolls/data"
)

const SessionName string = "moviepoll-session"

type Server interface {
	ListenAndServe() error
	Shutdown() error
}

type Options struct {
	Listen string // eg, "127.0.0.1:8080" or ":8080" (defaults to 0.0.0.0:8080)
	Debug  bool   // debug logging to console
}

type webServer struct {
	templates map[string]*template.Template
	s         *http.Server
	debug     bool // turns on debug things (eg, reloading templates on each page request)
	backend   logic.Logic

	cookies      *sessions.CookieStore
	passwordSalt string

	l *models.Logger
}

func New(options Options, backend logic.Logic, log *models.Logger) (Server, error) {
	if options.Listen == "" {
		options.Listen = ":8090"
	}

	err := os.MkdirAll("posters", 0755)
	if err != nil {
		return nil, fmt.Errorf("Unable to create posters directory: %v", err)
	}

	hs := &http.Server{
		Addr: options.Listen,
	}

	authKey, encryptKey, passwordSalt, err := backend.GetKeys()
	if err != nil {
		return nil, fmt.Errorf("Unable to get keys: %v", err)
	}

	server := &webServer{
		debug:        options.Debug,
		passwordSalt: passwordSalt,

		cookies: sessions.NewCookieStore([]byte(authKey), []byte(encryptKey)),
		l:       log,
	}

	err = server.initOauth()
	if err != nil {
		return nil, err
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/movie/", server.pageMovie)
	mux.HandleFunc("/static/", server.handlerStatic)
	mux.HandleFunc("/posters/", server.handlerStatic)
	mux.HandleFunc("/add", server.pageAddMovie)

	// list of past cycles
	mux.HandleFunc("/history", server.handlerHistory)

	mux.HandleFunc("/oauth/twitch", server.handlerTwitchOAuth)
	mux.HandleFunc("/oauth/twitch/callback", server.handlerTwitchOAuthCallback)

	mux.HandleFunc("/oauth/discord", server.handlerDiscordOAuth)
	mux.HandleFunc("/oauth/discord/callback", server.handlerDiscordOAuthCallback)

	mux.HandleFunc("/oauth/patreon", server.handlerPatreonOAuth)
	mux.HandleFunc("/oauth/patreon/callback", server.handlerPatreonOAuthCallback)

	mux.HandleFunc("/user", server.handlerUser)
	mux.HandleFunc("/user/login", server.handlerUserLogin)
	mux.HandleFunc("/user/logout", server.handlerUserLogout)
	mux.HandleFunc("/user/new", server.handlerUserNew)
	mux.HandleFunc("/user/remove/local", server.handlerLocalAuthRemove)

	mux.HandleFunc("/vote/", server.handlerVote)
	mux.HandleFunc("/", server.pageMain)
	mux.HandleFunc("/favicon.ico", server.handlerFavicon)

	mux.HandleFunc("/auth/", server.handlerAuth)
	mux.HandleFunc("/admin/", server.handlerAdmin)
	mux.HandleFunc("/admin/config", server.handlerAdminConfig)
	mux.HandleFunc("/admin/cycles", server.handlerAdminCycles)
	mux.HandleFunc("/admin/cyclepost", server.handlerAdminCycles_Post)
	// mux.HandleFunc("/admin/nextcycle", server.handlerAdminNextCycle)
	mux.HandleFunc("/admin/user/", server.handlerAdminUserEdit)
	mux.HandleFunc("/admin/users", server.handlerAdminUsers)
	mux.HandleFunc("/admin/movies", server.handlerAdminMovies)
	mux.HandleFunc("/admin/movie/", server.handlerAdminMovieEdit)

	hs.Handler = mux
	server.s = hs

	err = server.registerTemplates()
	if err != nil {
		return nil, err
	}

	return server, nil
	return nil, fmt.Errorf("not implemented")
}

func (s *webServer) ListenAndServe() error {
	return s.s.ListenAndServe()
}

func (s *webServer) Shutdown(ctx context.Context) error {
	//s.backend.Close()
	return s.s.Shutdown(ctx)
}

func (s *webServer) doError(code int, message string, w http.ResponseWriter, r *http.Request) {
	s.l.Debug("%d for %q", code, r.URL.Path)
	dataErr := dataError{
		dataPageBase: s.newPageBase("Error", w, r),
		Message:      message,
		Code:         code,
	}

	w.WriteHeader(http.StatusNotFound)
	if err := s.executeTemplate(w, "error", dataErr); err != nil {
		s.l.Error("Error rendering template: %v", err)
	}
}
