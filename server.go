package moviepoll

import (
	"crypto/rand"
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"
	//"time"
	"math/big"
	"strings"

	"github.com/gorilla/sessions"
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
	data      DataConnector

	// TODO: do this better (connect it to a proper account)
	adminUser string
	adminPass string
	cookies   *sessions.CookieStore
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

	cfg, err := data.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("Unable to get config: %v", err)
	}

	un, err := cfg.GetString("AdminUsername")
	if err != nil {
		return nil, fmt.Errorf("Missing admin username in config!")
	}

	pw, err := cfg.GetString("AdminPassword")
	if err != nil {
		return nil, fmt.Errorf("Missing admin password in config!")
	}

	authKey, err := cfg.GetString("SessionAuth")
	if err != nil {
		authKey = getCryptRandKey(64)
		cfg.SetString("SessionAuth", authKey)
	}

	encryptKey, err := cfg.GetString("SessionEncrypt")
	if err != nil {
		encryptKey = getCryptRandKey(32)
		cfg.SetString("SessionEncrypt", encryptKey)
	}

	server := &Server{
		debug: options.Debug,
		data:  data,

		adminUser: un,
		adminPass: pw,

		cookies: sessions.NewCookieStore([]byte(authKey), []byte(encryptKey)),
	}

	mux := http.NewServeMux()
	mux.Handle("/api/", apiHandler{})
	mux.HandleFunc("/movie/", server.handler_Movie)
	mux.HandleFunc("/data/", server.handler_Data)
	mux.HandleFunc("/login", server.handler_Login)
	mux.HandleFunc("/add", server.handler_AddMovie)
	mux.HandleFunc("/", server.handler_Root)
	mux.HandleFunc("/favicon.ico", server.handler_Favicon)

	hs.Handler = mux
	server.s = hs

	err = server.registerTemplates()
	if err != nil {
		return nil, err
	}

	server.data.SaveConfig(cfg)

	return server, nil
}

func (server *Server) Run() error {
	return server.s.ListenAndServe()
}

func (s *Server) handler_Favicon(w http.ResponseWriter, r *http.Request) {
	if fileExists("data/favicon.ico") {
		http.ServeFile(w, r, "data/favicon.ico")
	} else {
		http.NotFound(w, r)
	}
}

func (s *Server) handler_Data(w http.ResponseWriter, r *http.Request) {
	file := "data/" + filepath.Base(r.URL.Path)
	fmt.Printf("Attempting to serve file %q\n", file)
	http.ServeFile(w, r, file)
}

func (s *Server) handler_Login(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		fmt.Printf("Error parsing login form: %v\n", err)
	}

	isAuthed := s.getSessionBool("authed", r)

	if isAuthed {
		fmt.Println("Auth'd")
		if logout, ok := r.Form["logout"]; ok {
			fmt.Println("logout: ", logout)
			isAuthed = false
		}
	}

	data := dataLoginForm{}
	doRedirect := false

	if r.Method == "POST" {
		// do login

		un := r.PostFormValue("Username")
		pw := r.PostFormValue("Password")
		if un == s.adminUser && pw == s.adminPass {
			// do login (eg, session stuff)
			//data.ErrorMessage = "Login successfull!"
			fmt.Println("Successful login")
			isAuthed = true
			doRedirect = true
		} else {
			data.ErrorMessage = "Missing or invalid login credentials"
			fmt.Printf("Invalid login with: %q/%q\n", un, pw)
		}
	} else {
		fmt.Printf("> no post: %s\n", r.Method)
	}

	data.Authed = isAuthed
	s.setSessionValue("authed", isAuthed, w, r)

	// Redirect to base page on successful login
	if doRedirect {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	data.dataPageBase = s.newPageBase("Login", r) // set this last to get correct login status

	if err := s.executeTemplate(w, "simplelogin", data); err != nil {
		fmt.Printf("Error rendering template: %v\n", err)
	}
}

func (s *Server) handler_AddMovie(w http.ResponseWriter, r *http.Request) {
	fmt.Println(s.data.GetConnectionString())
	data := dataAddMovie{
		dataPageBase: s.newPageBase("Add Movie", r),
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
		links, err = verifyLinks(links)
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

		movie := &Movie{
			Name:        strings.TrimSpace(r.PostFormValue("MovieName")),
			Description: strings.TrimSpace(r.PostFormValue("Description")),
			Votes:       []*Vote{},
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

// TODO: 404 when URL isn't "/"
func (s *Server) handler_Root(w http.ResponseWriter, r *http.Request) {
	data := dataCycleOther{
		dataPageBase: s.newPageBase("Current Cycle", r),

		Cycle:  &Cycle{}, //s.data.GetCurrentCycle(),
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
			dataPageBase: s.newPageBase("Error", r),
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
			dataPageBase: s.newPageBase("Error", r),
			ErrorMessage: "Movie not found",
		}

		if err := s.executeTemplate(w, "movieError", dataError); err != nil {
			http.Error(w, fmt.Sprintf("%v", err), http.StatusInternalServerError)
			fmt.Println(err)
		}
		return
	}

	data := dataMovieInfo{
		dataPageBase: s.newPageBase(movie.Name, r),
		Movie:        movie,
	}

	if err := s.executeTemplate(w, "movieinfo", data); err != nil {
		http.Error(w, fmt.Sprintf("%v", err), http.StatusInternalServerError)
	}
}

func getCryptRandKey(size int) string {
	out := ""
	large := big.NewInt(int64(1 << 60))
	large = large.Add(large, large)
	for len(out) < size {
		num, err := rand.Int(rand.Reader, large)
		if err != nil {
			panic("Error generating session key: " + err.Error())
		}
		out = fmt.Sprintf("%s%X", out, num)
	}

	if len(out) > size {
		out = out[:size]
	}
	return out
}

func (s *Server) getSessionBool(key string, r *http.Request) bool {
	session, err := s.cookies.Get(r, SessionName)
	if err != nil {
		fmt.Printf("Unable to get session from store: %v\n", err)
		return false
	}

	val := session.Values[key]
	var boolVal bool
	var ok bool
	if boolVal, ok = val.(bool); !ok {
		boolVal = false
	}

	return boolVal
}

func (s *Server) setSessionValue(key string, val interface{}, w http.ResponseWriter, r *http.Request) {
	session, err := s.cookies.Get(r, SessionName)
	if err != nil {
		fmt.Printf("Unable to get session from store: %v\n", err)
	}
	session.Values[key] = val

	err = session.Save(r, w)
	if err != nil {
		fmt.Printf("Unable to save cookie: %v\n", err)
	}
}
