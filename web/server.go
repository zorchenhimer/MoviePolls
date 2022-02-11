package web

import (
	"context"
	"fmt"
	"html/template"

	"net/http"
	"os"

	"github.com/gorilla/sessions"

	"github.com/zorchenhimer/MoviePolls/logger"
	"github.com/zorchenhimer/MoviePolls/logic"
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

	l *logger.Logger
}

func New(options Options, backend logic.Logic, log *logger.Logger) (*webServer, error) {
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
		backend: backend,
	}

	err = server.initOauth()
	if err != nil {
		return nil, err
	}

	mux := http.NewServeMux()

	handlers := map[string]func(http.ResponseWriter, *http.Request){
		// Static stuff
		"/static/":     server.handlerStatic,
		"/posters/":    server.handlerPosters,
		"/favicon.ico": server.handlerFavicon,

		// Main Page handlers
		"/":        server.handlerPageMain,
		"/add":     server.handlerPageAddMovie,
		"/movie/":  server.handlerPageMovie,
		"/history": server.handlerPageHistory,
		"/user":    server.handlerPageUser,

		// User management
		"/user/login":        server.handlerUserLogin,
		"/user/logout":       server.handlerUserLogout,
		"/user/new":          server.handlerUserNew,
		"/user/remove/local": server.handlerLocalAuthRemove,

		// Functional endpoints (used for page functionality) - not having a page itself
		"/vote/": server.handlerVote,

		"/oauth/twitch":          server.handlerTwitchOAuth,
		"/oauth/twitch/callback": server.handlerTwitchOAuthCallback,

		"/oauth/discord":          server.handlerDiscordOAuth,
		"/oauth/discord/callback": server.handlerDiscordOAuthCallback,

		"/oauth/patreon":          server.handlerPatreonOAuth,
		"/oauth/patreon/callback": server.handlerPatreonOAuthCallback,

		// Admin pages
		"/auth/":           server.handlerAuth,
		"/admin/":          server.handlerAdminHome,
		"/admin/config":    server.handlerAdminConfig,
		"/admin/cycles":    server.handlerAdminCycles,
		"/admin/cyclepost": server.handlerAdminCycles_Post,
		"/admin/user/":     server.handlerAdminUserEdit,
		"/admin/users":     server.handlerAdminUsers,
		"/admin/movies":    server.handlerAdminMovies,
		"/admin/movie/":    server.handlerAdminMovieEdit,

		// "/admin/nextcycle", server.handlerAdminNextCycle)
	}

	for path, handler := range handlers {
		mux.HandleFunc(path, handler)
	}

	hs.Handler = mux
	server.s = hs

	err = server.registerTemplates()
	if err != nil {
		return nil, err
	}

	return server, nil
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
