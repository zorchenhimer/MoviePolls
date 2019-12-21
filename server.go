package moviepoll

import (
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gorilla/sessions"
	"github.com/zorchenhimer/MoviePolls/common"
	mpd "github.com/zorchenhimer/MoviePolls/data"
)

const SessionName string = "moviepoll-session"

type Options struct {
	Listen string // eg, "127.0.0.1:8080" or ":8080" (defaults to 0.0.0.0:8080)
	Debug  bool   // debug logging to console
}

type Server struct {
	templates map[string]*template.Template
	s         *http.Server
	debug     bool // turns on debug things (eg, reloading templates on each page request)
	data      common.DataConnector

	cookies      *sessions.CookieStore
	passwordSalt string
}

func NewServer(options Options) (*Server, error) {
	if options.Listen == "" {
		options.Listen = ":8080"
	}

	data, err := mpd.GetDataConnector("json", "db/data.json")
	if err != nil {
		return nil, fmt.Errorf("Unable to load json data: %v", err)
	}

	hs := &http.Server{
		Addr: options.Listen,
	}

	authKey, err := data.GetCfgString("SessionAuth")
	if err != nil {
		authKey = getCryptRandKey(64)
		data.SetCfgString("SessionAuth", authKey)
	}

	encryptKey, err := data.GetCfgString("SessionEncrypt")
	if err != nil {
		encryptKey = getCryptRandKey(32)
		data.SetCfgString("SessionEncrypt", encryptKey)
	}

	if options.Debug {
		fmt.Println("Debug mode turned on")
	}

	server := &Server{
		debug: options.Debug,
		data:  data,

		cookies: sessions.NewCookieStore([]byte(authKey), []byte(encryptKey)),
	}

	server.passwordSalt, err = server.data.GetCfgString("PassSalt")
	if err != nil {
		server.passwordSalt = getCryptRandKey(32)
		server.data.SetCfgString("PassSalt", server.passwordSalt)
	}

	mux := http.NewServeMux()
	mux.Handle("/api/", apiHandler{})
	mux.HandleFunc("/movie/", server.handlerMovie)
	mux.HandleFunc("/static/", server.handlerStatic)
	mux.HandleFunc("/poster/", server.handlerPoster)
	mux.HandleFunc("/add", server.handlerAddMovie)

	mux.HandleFunc("/user", server.handlerUser)
	mux.HandleFunc("/user/login", server.handlerUserLogin)
	mux.HandleFunc("/user/logout", server.handlerUserLogout)
	mux.HandleFunc("/user/new", server.handlerUserNew)

	mux.HandleFunc("/vote/", server.handlerVote)
	mux.HandleFunc("/", server.handlerRoot)
	mux.HandleFunc("/favicon.ico", server.handlerFavicon)

	mux.HandleFunc("/admin/", server.handlerAdmin)
	mux.HandleFunc("/admin/config", server.handlerAdminConfig)
	mux.HandleFunc("/admin/user/", server.handlerAdminUserEdit)
	mux.HandleFunc("/admin/users", server.handlerAdminUsers)

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

func (s *Server) handlerFavicon(w http.ResponseWriter, r *http.Request) {
	if common.FileExists("data/favicon.ico") {
		http.ServeFile(w, r, "data/favicon.ico")
	} else {
		http.NotFound(w, r)
	}
}

func (s *Server) handlerStatic(w http.ResponseWriter, r *http.Request) {
	file := "static/" + filepath.Base(r.URL.Path)
	if s.debug {
		fmt.Printf("Attempting to serve file %q\n", file)
	}
	http.ServeFile(w, r, file)
}

func (s *Server) handlerPoster(w http.ResponseWriter, r *http.Request) {
	file := "posters/" + filepath.Base(r.URL.Path)
	if s.debug {
		fmt.Printf("Attempting to serve file %q\n", file)
	}
	http.ServeFile(w, r, file)
}

func (s *Server) handlerAddMovie(w http.ResponseWriter, r *http.Request) {
	data := dataAddMovie{
		dataPageBase: s.newPageBase("Add Movie", w, r),
	}

	err := r.ParseForm()
	if err != nil {
		fmt.Printf("Error parsing movie form: %v\n", err)
	}

	if r.Method == "POST" {
		errText := []string{}

		linktext := strings.ReplaceAll(r.PostFormValue("Links"), "\r", "")
		fmt.Printf("linktext: %q\n", linktext)
		data.ValLinks = linktext

		links := strings.Split(linktext, "\n")
		links, err = common.VerifyLinks(links)
		if err != nil {
			fmt.Printf("bad link: %v\n", err)
			data.ErrLinks = true
			errText = append(errText, "Invalid link(s) given.")
		}

		data.ValTitle = strings.TrimSpace(r.PostFormValue("MovieName"))
		if s.data.CheckMovieExists(r.PostFormValue("MovieName")) {
			data.ErrTitle = true
			fmt.Println("Movie exists")
			errText = append(errText, "Movie already exists")
		}

		if data.ValTitle == "" {
			errText = append(errText, "Missing movie title")
			data.ErrTitle = true
		}

		descr := strings.TrimSpace(r.PostFormValue("Description"))
		data.ValDescription = descr
		if len(descr) == 0 {
			data.ErrDescription = true
			errText = append(errText, "Missing description")
		}

		movie := &common.Movie{
			Name:        strings.TrimSpace(r.PostFormValue("MovieName")),
			Description: strings.TrimSpace(r.PostFormValue("Description")),
			Votes:       []*common.Vote{},
			Links:       links,
			Poster:      "data/unknown.jpg", // 165x250
		}

		var movieId int
		if !data.isError() {
			movieId, err = s.data.AddMovie(movie)
		}

		if err == nil && !data.isError() {
			http.Redirect(w, r, fmt.Sprintf("/movie/%d", movieId), http.StatusFound)
			return
		}

		//data.ErrorMessage = strings.Join(errText, "<br />")
		data.ErrorMessage = errText
		fmt.Printf("Movie not added. isError(): %t\nerr: %v\n", data.isError(), err)
	}

	if err := s.executeTemplate(w, "addmovie", data); err != nil {
		fmt.Printf("Error rendering template: %v\n", err)
	}
}

func (s *Server) doError(code int, message string, w http.ResponseWriter, r *http.Request) {
	fmt.Printf("%d for %q\n", code, r.URL.Path)
	dataErr := dataError{
		dataPageBase: s.newPageBase("Error", w, r),
		Message:      message,
		Code:         code,
	}

	w.WriteHeader(http.StatusNotFound)
	if err := s.executeTemplate(w, "error", dataErr); err != nil {
		fmt.Printf("Error rendering template: %v\n", err)
	}
}

func (s *Server) handlerRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		s.doError(http.StatusNotFound, fmt.Sprintf("%q not found", r.URL.Path), w, r)
		return
	}

	data := dataCycle{
		dataPageBase: s.newPageBase("Current Cycle", w, r),

		Cycle:  &common.Cycle{}, //s.data.GetCurrentCycle(),
		Movies: s.data.GetActiveMovies(),
	}

	if err := s.executeTemplate(w, "cyclevotes", data); err != nil {
		fmt.Printf("Error rendering template: %v\n", err)
		//http.Error(w, fmt.Sprintf("%v", err), http.StatusInternalServerError)
	}
}

func (s *Server) handlerVote(w http.ResponseWriter, r *http.Request) {
	user := s.getSessionUser(w, r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	var movieId int
	if _, err := fmt.Sscanf(r.URL.Path, "/vote/%d", &movieId); err != nil {
		s.doError(http.StatusBadRequest, "Invalid movie ID", w, r)
		fmt.Printf("invalid vote URL: %q\n", r.URL.Path)
		return
	}

	if _, err := s.data.GetMovie(movieId); err != nil {
		s.doError(http.StatusBadRequest, "Invalid movie ID", w, r)
		fmt.Printf("Movie with ID %d doesn't exist\n", movieId)
		return
	}

	if s.data.UserVotedForMovie(user.Id, movieId) {
		s.doError(http.StatusBadRequest, "You already voted for that movie!", w, r)
		return
	}

	if err := s.data.AddVote(user.Id, movieId); err != nil {
		s.doError(http.StatusBadRequest, "Something went wrong :c", w, r)
		fmt.Printf("Unable to cast vote: %v\n", err)
		return
	}

	ref := r.Header.Get("Referer")
	if ref == "" {
		http.Redirect(w, r, "/", http.StatusFound)
	}
	http.Redirect(w, r, ref, http.StatusFound)
}

func (s *Server) handlerMovie(w http.ResponseWriter, r *http.Request) {
	var movieId int
	var command string
	n, err := fmt.Sscanf(r.URL.String(), "/movie/%d/%s", &movieId, &command)
	if err != nil && n == 0 {
		dataError := dataMovieError{
			dataPageBase: s.newPageBase("Error", w, r),
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
			dataPageBase: s.newPageBase("Error", w, r),
			ErrorMessage: "Movie not found",
		}

		if err := s.executeTemplate(w, "movieError", dataError); err != nil {
			http.Error(w, fmt.Sprintf("%v", err), http.StatusInternalServerError)
			fmt.Println(err)
		}
		return
	}

	data := dataMovieInfo{
		dataPageBase: s.newPageBase(movie.Name, w, r),
		Movie:        movie,
	}

	if err := s.executeTemplate(w, "movieinfo", data); err != nil {
		http.Error(w, fmt.Sprintf("%v", err), http.StatusInternalServerError)
	}
}
