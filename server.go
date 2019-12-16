package moviepoll

import (
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"
	"strings"
	"time"

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

	cookies      *sessions.CookieStore
	passwordSalt string
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

	if options.Debug {
		fmt.Println("Debug mode turned on")
	}

	server := &Server{
		debug: options.Debug,
		data:  data,

		cookies: sessions.NewCookieStore([]byte(authKey), []byte(encryptKey)),
	}

	server.passwordSalt, err = cfg.GetString("PassSalt")
	if err != nil {
		server.passwordSalt = getCryptRandKey(32)
		cfg.SetString("PassSalt", server.passwordSalt)
	}

	mux := http.NewServeMux()
	mux.Handle("/api/", apiHandler{})
	mux.HandleFunc("/movie/", server.handlerMovie)
	mux.HandleFunc("/data/", server.handlerData)
	mux.HandleFunc("/login", server.handlerLogin)
	mux.HandleFunc("/add", server.handlerAddMovie)
	mux.HandleFunc("/account", server.handlerAccount)
	mux.HandleFunc("/account/new", server.handlerNewAccount)
	mux.HandleFunc("/vote/", server.handlerVote)
	mux.HandleFunc("/", server.handlerRoot)
	mux.HandleFunc("/favicon.ico", server.handlerFavicon)

	mux.HandleFunc("/admin/", server.handlerAdmin)
	mux.HandleFunc("/admin/config", server.handlerAdminConfig)
	mux.HandleFunc("/admin/users", server.handlerAdminUsers)
	mux.HandleFunc("/admin/user/", server.handlerAdminUserEdit)

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

func (s *Server) handlerFavicon(w http.ResponseWriter, r *http.Request) {
	if fileExists("data/favicon.ico") {
		http.ServeFile(w, r, "data/favicon.ico")
	} else {
		http.NotFound(w, r)
	}
}

func (s *Server) handlerData(w http.ResponseWriter, r *http.Request) {
	file := "data/" + filepath.Base(r.URL.Path)
	if s.debug {
		fmt.Printf("Attempting to serve file %q\n", file)
	}
	http.ServeFile(w, r, file)
}

func (s *Server) handlerLogin(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		fmt.Printf("Error parsing login form: %v\n", err)
	}

	user := s.getSessionUser(w, r)
	if user != nil {
		fmt.Println("Auth'd")
		if logout, ok := r.Form["logout"]; ok {
			fmt.Println("logout: ", logout)
			err = s.logout(w, r)
			if err != nil {
				fmt.Printf("Error logging out: %v", err)
			}
		} else {
			http.Redirect(w, r, "/account", http.StatusFound)
			return
		}
	}

	data := dataLoginForm{}
	doRedirect := false

	if r.Method == "POST" {
		// do login

		un := r.PostFormValue("Username")
		pw := r.PostFormValue("Password")
		user, err = s.data.UserLogin(un, s.hashPassword(pw))
		if err != nil {
			data.ErrorMessage = err.Error()
		} else {
			doRedirect = true
		}

	} else {
		fmt.Printf("> no post: %s\n", r.Method)
	}

	if user != nil {
		err = s.login(user, w, r)
		if err != nil {
			fmt.Printf("Unable to login: %v", err)
			s.doError(http.StatusInternalServerError, "Unable to login", w, r)
			return
		}
	}

	// Redirect to base page on successful login
	if doRedirect {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	data.dataPageBase = s.newPageBase("Login", w, r) // set this last to get correct login status

	if err := s.executeTemplate(w, "simplelogin", data); err != nil {
		fmt.Printf("Error rendering template: %v\n", err)
	}
}

func (s *Server) handlerNewAccount(w http.ResponseWriter, r *http.Request) {
	user := s.getSessionUser(w, r)
	if user != nil {
		http.Redirect(w, r, "/account", http.StatusFound)
		return
	}

	data := dataNewAccount{
		dataPageBase: s.newPageBase("Create Account", w, r),
	}

	doRedirect := false

	if r.Method == "POST" {
		err := r.ParseForm()
		if err != nil {
			fmt.Printf("Error parsing login form: %v\n", err)
			data.ErrorMessage = append(data.ErrorMessage, err.Error())
		}

		un := strings.TrimSpace(r.PostFormValue("Username"))
		// TODO: password requirements
		pw1 := r.PostFormValue("Password1")
		pw2 := r.PostFormValue("Password2")

		data.ValName = un

		if un == "" {
			data.ErrorMessage = append(data.ErrorMessage, "Username cannot be blank!")
			data.ErrName = true
		}

		if pw1 != pw2 {
			data.ErrorMessage = append(data.ErrorMessage, "Passwords do not match!")
			data.ErrPass = true

		} else if pw1 == "" {
			data.ErrorMessage = append(data.ErrorMessage, "Password cannot be blank!")
			data.ErrPass = true
		}

		notifyEnd := r.PostFormValue("NotifyEnd")
		notifySelected := r.PostFormValue("NotifySelected")
		email := r.PostFormValue("Email")

		data.ValEmail = email
		if notifyEnd != "" {
			data.ValNotifyEnd = true
		}

		if notifySelected != "" {
			data.ValNotifySelected = true
		}

		if (notifyEnd != "" || notifySelected != "") && email == "" {
			data.ErrEmail = true
			data.ErrorMessage = append(data.ErrorMessage, "Email required for notifications")
		}

		newUser := &User{
			Name:                un,
			Password:            s.hashPassword(pw1),
			Email:               email,
			NotifyCycleEnd:      data.ValNotifyEnd,
			NotifyVoteSelection: data.ValNotifySelected,
			PassDate:            time.Now(),
		}

		_, err = s.data.AddUser(newUser)
		if err != nil {
			data.ErrorMessage = append(data.ErrorMessage, err.Error())
		} else {
			err = s.login(newUser, w, r)
			if err != nil {
				fmt.Printf("Unable to login to session: %v\n", err)
				s.doError(http.StatusInternalServerError, "Login error", w, r)
				return
			}
			doRedirect = true
		}
	}

	if doRedirect {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	if err := s.executeTemplate(w, "newaccount", data); err != nil {
		fmt.Printf("Error rendering template: %v\n", err)
	}
}

func (s *Server) handlerAccount(w http.ResponseWriter, r *http.Request) {
	user := s.getSessionUser(w, r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	config, err := s.data.GetConfig()
	if err != nil {
		fmt.Println("Unable to get config:", err)
		s.doError(http.StatusInternalServerError, "Config error", w, r)
		return
	}

	totalVotes, err := config.GetInt("MaxUserVotes")
	if err != nil {
		fmt.Printf("Error getting MaxUserVotes config setting: %v\n", err)
		totalVotes = 5 // FIXME: define a default somewhere?
	}

	data := dataAccount{
		dataPageBase: s.newPageBase("Account", w, r),

		CurrentVotes: s.data.GetUserVotes(user.Id),
		TotalVotes:   totalVotes,
	}
	data.AvailableVotes = totalVotes - len(data.CurrentVotes)

	if r.Method == "POST" {
		err := r.ParseForm()
		if err != nil {
			fmt.Printf("ParseForm() error: %v\n", err)
			s.doError(http.StatusInternalServerError, "Form error", w, r)
			return
		}

		formVal := r.PostFormValue("Form")
		if formVal == "ChangePassword" {
			// Do password stuff
			currentPass := s.hashPassword(r.PostFormValue("PasswordCurrent"))
			newPass1_raw := r.PostFormValue("PasswordNew1")
			newPass2_raw := r.PostFormValue("PasswordNew2")

			if currentPass != user.Password {
				data.ErrCurrentPass = true
				data.PassError = append(data.PassError, "Invalid current password")
			}

			if newPass1_raw == "" {
				data.ErrNewPass = true
				data.PassError = append(data.PassError, "New password cannot be blank")
			}

			if newPass1_raw != newPass2_raw {
				data.ErrNewPass = true
				data.PassError = append(data.PassError, "Passwords do not match")
			}

			if !data.IsErrored() {
				// Change pass
				data.SuccessMessage = "Password successfully changed"
				user.Password = s.hashPassword(newPass1_raw)
				user.PassDate = time.Now()

				fmt.Printf("new PassDate: %s\n", user.PassDate)

				err = s.login(user, w, r)
				if err != nil {
					fmt.Println("Unable to login to session: %v", err)
					s.doError(http.StatusInternalServerError, "Unable to update password", w, r)
					return
				}

				if err = s.data.UpdateUser(user); err != nil {
					fmt.Println("Unable to save User with new password:", err)
					s.doError(http.StatusInternalServerError, "Unable to update password", w, r)
					return
				}
			}

		} else if formVal == "Notifications" {
			// Update notifications
		}
	}

	if err := s.executeTemplate(w, "account", data); err != nil {
		fmt.Printf("Error rendering template: %v\n", err)
	}
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

		Cycle:  &Cycle{}, //s.data.GetCurrentCycle(),
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
