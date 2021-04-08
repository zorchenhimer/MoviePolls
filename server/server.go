package server

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/gorilla/sessions"
	mpd "github.com/zorchenhimer/MoviePolls/data"
	mpm "github.com/zorchenhimer/MoviePolls/models"
)

const SessionName string = "moviepoll-session"

// defaults
const (
	DefaultMaxUserVotes           int    = 5
	DefaultEntriesRequireApproval bool   = false
	DefaultFormfillEnabled        bool   = true
	DefaultVotingEnabled          bool   = false
	DefaultJikanEnabled           bool   = false
	DefaultJikanBannedTypes       string = "TV,music"
	DefaultJikanMaxEpisodes       int    = 1
	DefaultTmdbEnabled            bool   = false
	DefaultTmdbToken              string = ""
	DefaultMaxNameLength          int    = 100
	DefaultMinNameLength          int    = 4
	DefaultUnlimitedVotes         bool   = false

	DefaultMaxTitleLength       int = 100
	DefaultMaxDescriptionLength int = 1000
	DefaultMaxLinkLength        int = 500 // length of all links combined
	DefaultMaxRemarksLength     int = 200

	DefaultMaxMultEpLength int = 120 // length of multiple episode entries in minutes

	DefaultLocalSignupEnabled        bool   = true
	DefaultTwitchOauthEnabled        bool   = false
	DefaultTwitchOauthSignupEnabled  bool   = false
	DefaultTwitchOauthClientID       string = ""
	DefaultTwitchOauthClientSecret   string = ""
	DefaultDiscordOauthEnabled       bool   = false
	DefaultDiscordOauthSignupEnabled bool   = false
	DefaultDiscordOauthClientID      string = ""
	DefaultDiscordOauthClientSecret  string = ""
	DefaultPatreonOauthEnabled       bool   = false
	DefaultPatreonOauthSignupEnabled bool   = false
	DefaultPatreonOauthClientID      string = ""
	DefaultPatreonOauthClientSecret  string = ""
)

// configuration keys
const (
	ConfigVotingEnabled          string = "VotingEnabled"
	ConfigMaxUserVotes           string = "MaxUserVotes"
	ConfigEntriesRequireApproval string = "EntriesRequireApproval"
	ConfigFormfillEnabled        string = "FormfillEnabled"
	ConfigTmdbToken              string = "TmdbToken"
	ConfigJikanEnabled           string = "JikanEnabled"
	ConfigJikanBannedTypes       string = "JikanBannedTypes"
	ConfigJikanMaxEpisodes       string = "JikanMaxEpisodes"
	ConfigTmdbEnabled            string = "TmdbEnabled"
	ConfigMaxNameLength          string = "MaxNameLength"
	ConfigMinNameLength          string = "MinNameLength"
	ConfigNoticeBanner           string = "NoticeBanner"
	ConfigHostAddress            string = "HostAddress"
	ConfigUnlimitedVotes         string = "UnlimitedVotes"

	ConfigMaxTitleLength       string = "MaxTitleLength"
	ConfigMaxDescriptionLength string = "MaxDescriptionLength"
	ConfigMaxLinkLength        string = "MaxLinkLength"
	ConfigMaxRemarksLength     string = "MaxRemarksLength"

	ConfigMaxMultEpLength string = "ConfigMaxMultEpLength"

	ConfigLocalSignupEnabled        string = "LocalSignupEnabled"
	ConfigTwitchOauthEnabled        string = "TwitchOauthEnabled"
	ConfigTwitchOauthSignupEnabled  string = "TwitchOauthSignupEnabled"
	ConfigTwitchOauthClientID       string = "TwitchOauthClientID"
	ConfigTwitchOauthClientSecret   string = "TwitchOauthSecret"
	ConfigDiscordOauthEnabled       string = "DiscordOauthEnabled"
	ConfigDiscordOauthSignupEnabled string = "DiscordOauthSignupEnabled"
	ConfigDiscordOauthClientID      string = "DiscordOauthClientID"
	ConfigDiscordOauthClientSecret  string = "DiscordOauthClientSecret"
	ConfigPatreonOauthEnabled       string = "PatreonOauthEnabled"
	ConfigPatreonOauthSignupEnabled string = "PatreonOauthSignupEnabled"
	ConfigPatreonOauthClientID      string = "PatreonOauthClientID"
	ConfigPatreonOauthClientSecret  string = "PatreonOauthClientSecret"
)

var ReleaseVersion string

type Options struct {
	Listen   string // eg, "127.0.0.1:8080" or ":8080" (defaults to 0.0.0.0:8080)
	Debug    bool   // debug logging to console
	LogLevel mpm.LogLevel
	LogFile  string
}

type Server struct {
	templates map[string]*template.Template
	s         *http.Server
	debug     bool // turns on debug things (eg, reloading templates on each page request)
	data      mpd.DataConnector

	cookies      *sessions.CookieStore
	passwordSalt string

	l *mpm.Logger

	urlKeys map[string]*mpm.UrlKey
}

func NewServer(options Options) (*Server, error) {
	if options.Listen == "" {
		options.Listen = ":8090"
	}

	l, err := mpm.NewLogger(options.LogLevel, options.LogFile)
	if err != nil {
		return nil, fmt.Errorf("Unable to setup logger: %v", err)
	}

	err = os.MkdirAll("posters", 0755)
	if err != nil {
		return nil, fmt.Errorf("Unable to create posters directory: %v", err)
	}

	data, err := mpd.GetDataConnector("json", "db/data.json", l)
	if err != nil {
		return nil, fmt.Errorf("Unable to load json data: %v", err)
	}

	hs := &http.Server{
		Addr: options.Listen,
	}

	authKey, err := data.GetCfgString("SessionAuth", "")
	if err != nil || authKey == "" {
		authKey = getCryptRandKey(64)
		data.SetCfgString("SessionAuth", authKey)
	}

	encryptKey, err := data.GetCfgString("SessionEncrypt", "")
	if err != nil || encryptKey == "" {
		encryptKey = getCryptRandKey(32)
		data.SetCfgString("SessionEncrypt", encryptKey)
	}

	l.Info("Running version: %s", ReleaseVersion)
	if options.Debug {
		l.Info("Debug mode turned on")
	}

	server := &Server{
		debug: options.Debug,
		data:  data,

		cookies: sessions.NewCookieStore([]byte(authKey), []byte(encryptKey)),
		l:       l,
		urlKeys: make(map[string]*mpm.UrlKey),
	}

	server.passwordSalt, err = server.data.GetCfgString("PassSalt", "")
	if err != nil || server.passwordSalt == "" {
		server.passwordSalt = getCryptRandKey(32)
		server.data.SetCfgString("PassSalt", server.passwordSalt)
	}

	adminExists, err := server.CheckAdminExists()
	if err != nil {
		return nil, err
	}

	if !adminExists {
		urlKey, err := mpm.NewAdminAuth()
		if err != nil {
			return nil, fmt.Errorf("Unable to get Url/Key pair for admin auth: %v", err)
		}

		server.urlKeys[urlKey.Url] = urlKey

		host, err := server.data.GetCfgString(ConfigHostAddress, "")
		if err != nil {
			return nil, fmt.Errorf("Unable to get host: %v", err)
		}

		if host == "" {
			host = "http://<host>"
		}
		host = strings.ToLower(host)

		if !strings.HasPrefix(host, "http") {
			host = "http://" + host
		}

		// Print directly to the console, not through the logger.
		fmt.Printf("Claim admin: %s/auth/%s Password: %s\n", host, urlKey.Url, urlKey.Key)
	}

	err = server.initOauth()
	if err != nil {
		return nil, err
	}

	mux := http.NewServeMux()
	mux.Handle("/api/", apiHandler{})
	mux.HandleFunc("/movie/", server.handlerMovie)
	mux.HandleFunc("/static/", server.handlerStatic)
	mux.HandleFunc("/posters/", server.handlerPoster)
	mux.HandleFunc("/add", server.handlerAddMovie)

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
	mux.HandleFunc("/", server.handlerRoot)
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
}

func (s *Server) Run() error {
	s.l.Info("Listening on address %s", s.s.Addr)
	return s.s.ListenAndServe()
}

func (s *Server) CheckAdminExists() (bool, error) {
	found, end := false, false

	start := 0
	count := 20
	for !found && !end {
		users, err := s.data.GetUsers(start, 20)
		if err != nil {
			return false, err
		}
		start += count

		if err != nil {
			return false, nil
		}

		if len(users) == 0 {
			return false, nil
		}

		for _, u := range users {
			if u.IsAdmin() {
				return true, nil
			}
		}
	}

	s.l.Debug("[CheckAdminExists] end of loop")
	return false, nil
}

func (s *Server) doError(code int, message string, w http.ResponseWriter, r *http.Request) {
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

func (s *Server) uploadFile(r *http.Request, name string) (string, error) {
	s.l.Debug("[uploadFile] Start")
	var err error
	// 10 MB upload limit
	r.ParseMultipartForm(10 << 20)

	file, handler, err := r.FormFile("PosterFile")

	if err != nil {
		s.l.Error(err.Error())
		return "", fmt.Errorf("Unable to retrive the file")
	}

	defer file.Close()

	s.l.Info("Uploaded File: %v - Size %v", handler.Filename, handler.Size)

	tempFile, err := ioutil.TempFile("posters", name+"-*.png")

	if err != nil {
		return "", fmt.Errorf("Error while saving file to disk: %v", err)
	}
	defer tempFile.Close()

	fileBytes, err := ioutil.ReadAll(file)

	if err != nil {
		return "", err
	}

	tempFile.Write(fileBytes)

	s.l.Debug("[uploadFile] Filename: %v", tempFile.Name())

	return tempFile.Name(), nil
}
