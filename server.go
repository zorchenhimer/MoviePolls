package moviepoll

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gorilla/sessions"
	"github.com/zorchenhimer/MoviePolls/common"
	mpd "github.com/zorchenhimer/MoviePolls/data"
)

const SessionName string = "moviepoll-session"

// defaults
const (
	DefaultMaxUserVotes           int    = 5
	DefaultEntriesRequireApproval bool   = false
	DefaultAutofillEnabled        bool   = false
	DefaultFormfillEnabled        bool   = true
	DefaultVotingEnabled          bool   = false
	DefaultJikanEnabled           bool   = false
	DefaultJikanBannedTypes       string = "TV,music"
	DefaultJikanMaxEpisodes       int    = 1
	DefaultTmdbEnabled            bool   = false
	DefaultTmdbToken              string = ""
	DefaultMaxNameLength          int    = 100
	DefaultMinNameLength          int    = 4

	DefaultMaxTitleLength       int = 100
	DefaultMaxDescriptionLength int = 1000
	DefaultMaxLinkLength        int = 500 // length of all links combined
	DefaultMaxRemarksLength     int = 200
)

// configuration keys
const (
	ConfigVotingEnabled          string = "VotingEnabled"
	ConfigMaxUserVotes           string = "MaxUserVotes"
	ConfigEntriesRequireApproval string = "EntriesRequireApproval"
	ConfigAutofillEnabled        string = "AutofillEnabled"
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

	ConfigMaxTitleLength       string = "MaxTitleLength"
	ConfigMaxDescriptionLength string = "MaxDescriptionLength"
	ConfigMaxLinkLength        string = "MaxLinkLength"
	ConfigMaxRemarksLength     string = "MaxRemarksLength"
)

type Options struct {
	Listen   string // eg, "127.0.0.1:8080" or ":8080" (defaults to 0.0.0.0:8080)
	Debug    bool   // debug logging to console
	LogLevel common.LogLevel
	LogFile  string
}

type Server struct {
	templates map[string]*template.Template
	s         *http.Server
	debug     bool // turns on debug things (eg, reloading templates on each page request)
	data      mpd.DataConnector

	cookies      *sessions.CookieStore
	passwordSalt string

	l *common.Logger

	urlKeys map[string]*common.UrlKey
}

func NewServer(options Options) (*Server, error) {
	if options.Listen == "" {
		options.Listen = ":8090"
	}

	l, err := common.NewLogger(options.LogLevel, options.LogFile)
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

	if options.Debug {
		l.Info("Debug mode turned on")
	}

	server := &Server{
		debug: options.Debug,
		data:  data,

		cookies: sessions.NewCookieStore([]byte(authKey), []byte(encryptKey)),
		l:       l,
		urlKeys: make(map[string]*common.UrlKey),
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
		urlKey, err := common.NewAdminAuth()
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

	mux := http.NewServeMux()
	mux.Handle("/api/", apiHandler{})
	mux.HandleFunc("/movie/", server.handlerMovie)
	mux.HandleFunc("/static/", server.handlerStatic)
	mux.HandleFunc("/posters/", server.handlerPoster)
	mux.HandleFunc("/add", server.handlerAddMovie)

	// list of past cycles
	mux.HandleFunc("/history", server.handlerHistory)

	mux.HandleFunc("/user", server.handlerUser)
	mux.HandleFunc("/user/login", server.handlerUserLogin)
	mux.HandleFunc("/user/logout", server.handlerUserLogout)
	mux.HandleFunc("/user/new", server.handlerUserNew)

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

func (s *Server) AddUser(user *common.User) error {
	user.Password = s.hashPassword(user.Password)
	_, err := s.data.AddUser(user)
	return err
}

func (s *Server) handlerFavicon(w http.ResponseWriter, r *http.Request) {
	if common.FileExists("data/favicon.ico") {
		http.ServeFile(w, r, "data/favicon.ico")
	} else {
		http.NotFound(w, r)
	}
}

func (s *Server) handlerStatic(w http.ResponseWriter, r *http.Request) {
	file := strings.TrimLeft(filepath.Clean("/"+r.URL.Path), "/\\")
	if s.debug {
		s.l.Info("Attempting to serve file %q", file)
	}
	http.ServeFile(w, r, file)
}

func (s *Server) handlerPoster(w http.ResponseWriter, r *http.Request) {
	file := strings.TrimLeft(filepath.Clean("/"+r.URL.Path), "/\\")
	if s.debug {
		s.l.Info("Attempting to serve file %q", file)
	}
	http.ServeFile(w, r, file)
}

func (s *Server) handlerAddMovie(w http.ResponseWriter, r *http.Request) {

	// Get the user which adds a movie
	user := s.getSessionUser(w, r)
	if user == nil {
		http.Redirect(w, r, "/user/login", http.StatusSeeOther)
		return
	}

	// Get the current cycle to see if we can add a movie
	currentCycle, err := s.data.GetCurrentCycle()
	if err != nil {
		s.doError(
			http.StatusInternalServerError,
			"Something went wrong :C",
			w, r)

		s.l.Error("Unable to get current cycle: %v", err)
		return
	}

	if currentCycle == nil {
		s.doError(
			http.StatusInternalServerError,
			"No cycle active!",
			w, r)
		return
	}

	autofillEnabled, err := s.data.GetCfgBool(ConfigAutofillEnabled, DefaultAutofillEnabled)

	if err != nil {
		s.doError(
			http.StatusInternalServerError,
			"Something went wrong :C",
			w, r)

		s.l.Error("Unable to get config value %s: %v", ConfigAutofillEnabled, err)
		return
	}

	formfillEnabled, err := s.data.GetCfgBool(ConfigFormfillEnabled, DefaultFormfillEnabled)
	if err != nil {
		s.doError(
			http.StatusInternalServerError,
			"Something went wrong :C",
			w, r)

		s.l.Error("Unable to get config value %s: %v", ConfigFormfillEnabled, err)
		return
	}

	data := dataAddMovie{
		dataPageBase:    s.newPageBase("Add Movie", w, r),
		AutofillEnabled: autofillEnabled,
		FormfillEnabled: formfillEnabled,
	}

	if r.Method == "POST" {
		err = r.ParseMultipartForm(4096)
		if err != nil {
			s.l.Error("Error parsing movie form: %v", err)
		}

		// errText := []string{}

		movie := &common.Movie{}

		if autofillEnabled {
			if r.FormValue("AutofillBox") == "on" {
				// do autofill
				s.l.Debug("autofill")
				results, links := s.handleAutofill(&data, w, r)

				if results == nil || links == nil {
					data.ErrorMessage = append(data.ErrorMessage, "Could not autofill all fields")
					data.ErrAutofill = true
					return
				} else {
					// Fill all the fields in the movie struct
					movie.Name = results[0]
					movie.Description = results[1]
					movie.Poster = filepath.Base(results[2])
					movie.Remarks = results[3]
					movie.Links = links
					movie.AddedBy = user

					// Prepare a int for the id
					var movieId int

					movieId, err = s.data.AddMovie(movie)
					if err != nil {
						s.l.Error("Movie could not be added. Error: %v", err)
					} else {
						http.Redirect(w, r, fmt.Sprintf("/movie/%d", movieId), http.StatusFound)
					}
				}

			}
		} else if formfillEnabled {
			s.l.Debug("formfill")
			// do formfill
			// results := s.handleFormfill(&data, w, r)

		} else {
			s.l.Debug("neither")
			// neither autofill nor formfill are enabled
		}
	}
	//	 // Get all links from the corresponding input field
	//	 linktext := strings.ReplaceAll(r.FormValue("Links"), "\r", "")
	//	 data.ValLinks = linktext

	//	 maxLinkLength, err := s.data.GetCfgInt(ConfigMaxLinkLength, DefaultMaxLinkLength)
	//	 if err != nil {
	//	 	s.l.Error("Unable to get %q: %v", ConfigMaxTitleLength, err)
	//	 	s.doError(
	//	 		http.StatusInternalServerError,
	//	 		"something went wrong :C",
	//	 		w, r)
	//	 	return
	//	 }

	// if len(linktext) > maxLinkLength {
	// 	s.l.Debug("Links too long: %d", len(linktext))
	// 	data.ErrTitle = true
	// 	errText = append(errText, "Links too long!")
	// }
	//		links := strings.Split(linktext, "\n")
	//		links, err = common.VerifyLinks(links)
	//		if err != nil {
	//			s.l.Error("bad link: %v", err)
	//			data.ErrLinks = true
	//			errText = append(errText, "Invalid link(s) given.")
	//		}

	//		remarks := strings.ReplaceAll(r.FormValue("Remarks"), "\r", "")
	//		data.ValRemarks = remarks

	//		maxRemarksLength, err := s.data.GetCfgInt(ConfigMaxRemarksLength, DefaultMaxRemarksLength)
	//		if err != nil {
	//			s.l.Error("Unable to get %q: %v", ConfigMaxRemarksLength, err)
	//			s.doError(
	//				http.StatusInternalServerError,
	//				"something went wrong :C",
	//				w, r)
	//			return
	//		}

	//		if len(remarks) > maxRemarksLength {
	//			s.l.Debug("Remarks too long: %d", len(remarks))
	//			data.ErrRemarks = true
	//			errText = append(errText, "Remarks too long!")
	//		}

	//		// New Movie, just filling the Poster field for the "unknown.jpg" default
	//		movie := &common.Movie{
	//			Poster: "unknown.jpg", // 165x250
	//		}

	//		// Here we check the AutofillBox since the other fields are ignored
	//		if r.FormValue("AutofillBox") == "on" {
	//			s.l.Debug("Autofill checked")
	//			results, errors, rerenderSite := s.handleAutofill(links, w, r)

	//			if len(errors) > 0 {
	//				errText = append(errText, errors...)
	//				data.ErrAutofill = true

	//				if rerenderSite {
	//					data.ErrorMessage = errText
	//					if err := s.executeTemplate(w, "addmovie", data); err != nil {
	//						s.l.Error("Error rendering template: %v", err)
	//					}
	//					return
	//				}
	//			} else {
	//				movie.Name = results[0]
	//				movie.Description = results[1]
	//				movie.Poster = filepath.Base(results[2])
	//				movie.Links = links
	//				movie.Remarks = remarks
	//			}
	//		} else {
	//			s.l.Debug("Autofill not checked")

	//			// Here we continue with the other input checks
	//			maxTitleLength, err := s.data.GetCfgInt(ConfigMaxTitleLength, DefaultMaxTitleLength)
	//			if err != nil {
	//				s.l.Error("Unable to get %q: %v", ConfigMaxTitleLength, err)
	//				s.doError(
	//					http.StatusInternalServerError,
	//					"something went wrong :C",
	//					w, r)
	//				return
	//			}

	//			data.ValTitle = strings.TrimSpace(r.FormValue("MovieName"))
	//			if len(data.ValTitle) > maxTitleLength {
	//				s.l.Debug("Title too long: %d", len(data.ValTitle))
	//				data.ErrTitle = true
	//				errText = append(errText, "Title too long!")
	//			} else if len(data.ValTitle) == 0 {
	//				s.l.Debug("Title too short: %d", len(common.CleanMovieName(data.ValTitle)))
	//				data.ErrTitle = true
	//				errText = append(errText, "Title too short!")
	//			}

	//			movieExists, err := s.data.CheckMovieExists(r.FormValue("MovieName"))
	//			if err != nil {
	//				s.doError(
	//					http.StatusInternalServerError,
	//					fmt.Sprintf("Unable to check if movie exists: %v", err),
	//					w, r)
	//				return
	//			}

	//			if movieExists {
	//				data.ErrTitle = true
	//				s.l.Debug("Movie exists")
	//				errText = append(errText, "Movie already exists")
	//			}

	//			if data.ValTitle == "" && !(r.FormValue("AutofillBox") == "on") {
	//				errText = append(errText, "Missing movie title")
	//				data.ErrTitle = true
	//			}

	//			descr := strings.TrimSpace(r.FormValue("Description"))
	//			data.ValDescription = descr

	//			maxDescriptionLength, err := s.data.GetCfgInt(ConfigMaxDescriptionLength, DefaultMaxDescriptionLength)
	//			if err != nil {
	//				s.l.Error("Unable to get %q: %v", ConfigMaxTitleLength, err)
	//				s.doError(
	//					http.StatusInternalServerError,
	//					"something went wrong :C",
	//					w, r)
	//				return
	//			}

	//			if len(data.ValDescription) > maxDescriptionLength {
	//				s.l.Debug("Description too long: %d", len(data.ValDescription))
	//				data.ErrDescription = true
	//				errText = append(errText, "Description too long!")
	//			}

	//			if len(descr) == 0 && !(r.FormValue("AutofillBox") == "on") {
	//				data.ErrDescription = true
	//				errText = append(errText, "Missing description")
	//			}

	//			movie.Name = strings.TrimSpace(r.FormValue("MovieName"))
	//			movie.Description = strings.TrimSpace(r.FormValue("Description"))
	//			movie.Links = links
	//			movie.Remarks = remarks

	//			posterFileName := strings.TrimSpace(r.FormValue("MovieName"))

	//			posterFile, _, _ := r.FormFile("PosterFile")

	//			if posterFile != nil {
	//				file, err := s.uploadFile(r, posterFileName)

	//				if err != nil {
	//					data.ErrPoster = true
	//					errText = append(errText, err.Error())
	//				} else {
	//					movie.Poster = filepath.Base(file)
	//				}
	//			}
	//		}

	//		var movieId int

	//		if !data.isError() {
	//			movie.AddedBy = user
	//			movieId, err = s.data.AddMovie(movie)
	//		}

	//		if err == nil && !data.isError() {
	//			http.Redirect(w, r, fmt.Sprintf("/movie/%d", movieId), http.StatusFound)
	//			return
	//		}

	//		//data.ErrorMessage = strings.Join(errText, "<br />")
	//		data.ErrorMessage = errText
	//		s.l.Error("Movie not added. isError(): %t\nerr: %v", data.isError(), err)
	//	}

	if err := s.executeTemplate(w, "addmovie", data); err != nil {
		s.l.Error("Error rendering template: %v", err)
	}
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

func (s *Server) handlerRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		s.doError(http.StatusNotFound, fmt.Sprintf("%q not found", r.URL.Path), w, r)
		return
	}

	movieList := []*common.Movie{}

	data := struct {
		dataPageBase
		Movies         []*common.Movie
		VotingEnabled  bool
		AvailableVotes int
		LastCycle      *common.Cycle
		Cycle          *common.Cycle
	}{
		dataPageBase: s.newPageBase("Current Cycle", w, r),
	}

	if r.Body != http.NoBody {
		err := r.ParseForm()
		if err != nil {
			s.l.Error(err.Error())
		}
		searchVal := r.FormValue("search")

		movieList, err = s.data.SearchMovieTitles(searchVal)
		if err != nil {
			s.l.Error(err.Error())
		}
	} else {
		var err error = nil
		movieList, err = s.data.GetActiveMovies()
		if err != nil {
			s.l.Error(err.Error())
			s.doError(
				http.StatusBadRequest,
				fmt.Sprintf("Cannot get active movies. Please contact the server admin."),
				w, r)
			return
		}
	}

	if data.User != nil {
		votedMovies, err := s.data.GetUserVotes(data.User.Id)
		if err != nil {
			s.doError(
				http.StatusBadRequest,
				fmt.Sprintf("Cannot get user votes: %v", err),
				w, r)
			return
		}

		count := 0
		for _, movie := range votedMovies {
			// Only count active movies
			if movie.CycleWatched == nil && movie.Removed == false {
				count++
			}
		}

		maxVotes, err := s.data.GetCfgInt("MaxUserVotes", DefaultMaxUserVotes)
		if err != nil {
			s.l.Error("Error getting MaxUserVotes config setting: %v", err)
			maxVotes = DefaultMaxUserVotes
		}
		data.AvailableVotes = maxVotes - count
	}

	data.Movies = common.SortMoviesByVotes(movieList)
	data.VotingEnabled, _ = s.data.GetCfgBool("VotingEnabled", DefaultVotingEnabled)

	cycles, err := s.data.GetPastCycles(0, 1)
	if err != nil {
		s.l.Error("Error getting PastCycle: %v", err)
	}
	if cycles != nil {
		if len(cycles) != 0 {
			data.LastCycle = cycles[0]
		}
	}

	cycle, err := s.data.GetCurrentCycle()
	if err != nil {
		s.l.Error("Error getting Current Cycle: %v", err)
	}
	if cycle != nil {
		data.Cycle = cycle
	}

	if err := s.executeTemplate(w, "cyclevotes", data); err != nil {
		s.l.Error("Error rendering template: %v", err)
	}
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
			s.l.Error(err.Error())
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
			s.l.Error("movie not found: " + err.Error())
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

// outsourced autofill logic
func (s *Server) handleAutofill(data *dataAddMovie, w http.ResponseWriter, r *http.Request) (results []string, links []string) {

	// Get all needed values from the form

	// Get all links from the corresponding input field
	linktext := strings.ReplaceAll(r.FormValue("Links"), "\r", "")
	data.ValLinks = linktext

	// Get the remarks from the corresponding input field
	remarkstext := strings.ReplaceAll(r.FormValue("Remarks"), "\r", "")
	data.ValRemarks = remarkstext

	// Check link maxlength
	maxLinkLength, err := s.data.GetCfgInt(ConfigMaxLinkLength, DefaultMaxLinkLength)
	if err != nil {
		s.l.Error("Unable to get %q: %v", ConfigMaxLinkLength, err)
		s.doError(
			http.StatusInternalServerError,
			"something went wrong :C",
			w, r)
		return
	}

	if len(linktext) > maxLinkLength {
		s.l.Debug("Links too long: %d", len(linktext))
		data.ErrorMessage = append(data.ErrorMessage, fmt.Sprintf("Links too long! Max Length: %d characters", maxLinkLength))
		data.ErrLinks = true
	}

	// Check for valid links
	links = strings.Split(linktext, "\n")
	links, err = common.VerifyLinks(links)
	if err != nil {
		s.l.Error(err.Error())
		data.ErrorMessage = append(data.ErrorMessage, "Invalid link(s) given.")
		data.ErrLinks = true
	}

	var sourcelink string
	// make sure we have a link to look at
	if len(links) == 0 {
		s.l.Error("no links given")
		data.ErrorMessage = append(data.ErrorMessage, "No link found.")
		data.ErrLinks = true
	} else {
		sourcelink = links[0]
	}

	// Check Remarks max length
	maxRemarksLength, err := s.data.GetCfgInt(ConfigMaxRemarksLength, DefaultMaxRemarksLength)
	if err != nil {
		s.l.Error("Unable to get %q: %v", ConfigMaxRemarksLength, err)
		s.doError(
			http.StatusInternalServerError,
			"something went wrong :C",
			w, r)
		return
	}

	if len(remarkstext) > maxRemarksLength {
		s.l.Debug("Remarks too long: %d", len(remarkstext))
		data.ErrorMessage = append(data.ErrorMessage, fmt.Sprintf("Remarks too long! Max Length: %d characters", maxRemarksLength))
		data.ErrRemarks = true
	}

	// Exit early if any errors got reported
	if data.isError() {
		return nil, nil
	}

	if strings.Contains(sourcelink, "myanimelist") {
		s.l.Debug("MAL link")

		results, err = s.handleJikan(data, w, r, sourcelink)

		if err != nil {
			return nil, nil
		}

		var title string

		if len(results) != 3 {
			s.l.Error("Jikan API results have an unexpected length, expected 3 got %v", len(results))
			data.ErrorMessage = append(data.ErrorMessage, "API autofill did not return enough data, contact the server administrator")
			return nil, nil
		} else {
			title = results[0]
		}

		exists, err := s.data.CheckMovieExists(title)
		if err != nil {
			s.l.Error(err.Error())
			s.doError(
				http.StatusInternalServerError,
				"something went wrong :C",
				w, r)
			return nil, nil
		}

		if exists {
			s.l.Debug("Movie already exists")
			data.ErrorMessage = append(data.ErrorMessage, "Movie already exists in database")
			data.ErrAutofill = true
			return nil, nil
		}

		results = append(results, remarkstext)
		return results, links

	} else if strings.Contains(sourcelink, "imdb") {
		s.l.Debug("IMDB link")

		results, err = s.handleTmdb(data, w, r, sourcelink)

		if err != nil {
			return nil, nil
		}

		var title string

		if len(results) != 3 {
			s.l.Error("Tmdb API results have an unexpected length, expected 3 got %v", len(results))
			data.ErrorMessage = append(data.ErrorMessage, "API autofill did not return enough data, did you input a link to a series?")
			return nil, nil
		} else {
			title = results[0]
		}

		exists, err := s.data.CheckMovieExists(title)
		if err != nil {
			s.l.Error(err.Error())
			s.doError(
				http.StatusInternalServerError,
				"something went wrong :C",
				w, r)
			return nil, nil

		}

		if exists {
			s.l.Debug("Movie already exists")
			data.ErrorMessage = append(data.ErrorMessage, "Movie already exists in database")
			data.ErrAutofill = true
			return nil, nil
		}

		results = append(results, remarkstext)
		return results, links

	} else {
		s.l.Debug("no link")
		data.ErrorMessage = append(data.ErrorMessage, "To use autofill an imdb or myanimelist link as first link is required")
		data.ErrLinks = true
		return nil, nil
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

// List of past cycles
func (s *Server) handlerHistory(w http.ResponseWriter, r *http.Request) {
	past, err := s.data.GetPastCycles(0, 100)
	if err != nil {
		s.doError(
			http.StatusInternalServerError,
			fmt.Sprintf("something went wrong :C"),
			w, r)
		s.l.Error("Unable to get past cycles: ", err)
		return
	}

	data := struct {
		dataPageBase
		Cycles []*common.Cycle
	}{
		dataPageBase: s.newPageBase("Cycle History", w, r),
		Cycles:       past,
	}

	if err := s.executeTemplate(w, "history", data); err != nil {
		s.l.Error("Error rendering template: %v", err)
	}
}

func (s *Server) handleJikan(data *dataAddMovie, w http.ResponseWriter, r *http.Request, sourcelink string) ([]string, error) {

	jikanEnabled, err := s.data.GetCfgBool("JikanEnabled", DefaultJikanEnabled)
	if err != nil {
		s.l.Error("Error while retriving config value 'JikanEnabled':\n %v", err)
		s.doError(
			http.StatusInternalServerError,
			"something went wrong :C",
			w, r)
		return nil, err
	}

	s.l.Debug("jikanEnabled: %v", jikanEnabled)

	if !jikanEnabled {
		s.l.Debug("Aborting Jikan autofill since it is not enabled")
		data.ErrorMessage = append(data.ErrorMessage, "Jikan API usage was not enabled by the site administrator")
		return nil, fmt.Errorf("Jikan not enabled")
	} else {
		// Get Data from MAL (jikan api)
		rgx := regexp.MustCompile(`[htp]{4}s?:\/\/[^\/]*\/anime\/([0-9]*)`)
		match := rgx.FindStringSubmatch(sourcelink)
		var id string
		if len(match) < 2 {
			s.l.Debug("Regex match didn't find the anime id in %v", sourcelink)
			data.ErrorMessage = append(data.ErrorMessage, "Could not retrive anime id from provided link, did you input a manga link?")
			data.ErrLinks = true
			return nil, fmt.Errorf("Could not retrive anime id from link")
		} else {
			id = match[1]
		}

		bannedTypesString, err := s.data.GetCfgString(ConfigJikanBannedTypes, DefaultJikanBannedTypes)

		if err != nil {
			s.l.Error("Error while retriving config value 'JikanBannedTypes':\n %v", err)
			s.doError(
				http.StatusInternalServerError,
				"something went wrong :C",
				w, r)
			return nil, err
		}

		bannedTypes := strings.Split(bannedTypesString, ",")

		maxEpisodes, err := s.data.GetCfgInt(ConfigJikanMaxEpisodes, DefaultJikanMaxEpisodes)

		if err != nil {
			s.l.Error("Error while retriving config value 'JikanMaxEpisodes':\n %v", err)
			s.doError(
				http.StatusInternalServerError,
				"something went wrong :C",
				w, r)
			return nil, err
		}

		sourceAPI := jikan{id: id, l: s.l, excludedTypes: bannedTypes, maxEpisodes: maxEpisodes}

		// Request data from API
		results, err := getMovieData(&sourceAPI)

		if err != nil {
			s.l.Error("Error while accessing Jikan API: %v", err)
			data.ErrorMessage = append(data.ErrorMessage, err.Error())
			return nil, err
		}

		return results, nil
	}
}

func (s *Server) handleTmdb(data *dataAddMovie, w http.ResponseWriter, r *http.Request, sourcelink string) ([]string, error) {

	tmdbEnabled, err := s.data.GetCfgBool("TmdbEnabled", DefaultTmdbEnabled)
	if err != nil {
		s.l.Error("Error while retriving config value 'TmdbEnabled':\n %v", err)
		data.ErrorMessage = append(data.ErrorMessage, "Something went wrong :C")
		return nil, err
	}

	if !tmdbEnabled {
		s.l.Debug("Aborting Tmdb autofill since it is not enabled")
		data.ErrorMessage = append(data.ErrorMessage, "Tmdb API usage was not enabled by the site administrator")
		return nil, fmt.Errorf("Tmdb not enabled")
	} else {
		// Retrieve token from database
		token, err := s.data.GetCfgString("TmdbToken", "")
		if err != nil || token == "" {

			s.l.Debug("Aborting Tmdb autofill since no token was found")
			data.ErrorMessage = append(data.ErrorMessage, "TmdbToken is either empty or not set in the admin config")
			return nil, fmt.Errorf("TmdbToken is either empty or not set in the admin config")
		}
		// get the movie id
		rgx := regexp.MustCompile(`[htp]{4}s?:\/\/[^\/]*\/title\/(tt[0-9]*)`)
		match := rgx.FindStringSubmatch(sourcelink)
		var id string
		if len(match) < 2 {
			s.l.Debug("Regex match didn't find the movie id in %v", sourcelink)
			data.ErrorMessage = append(data.ErrorMessage, "Could not retrive movie id from provided link")
			data.ErrLinks = true
			return nil, fmt.Errorf("Could not retrive movie id from link")
		} else {
			id = match[1]
		}

		sourceAPI := tmdb{id: id, token: token, l: s.l}

		// Request data from API
		results, err := getMovieData(&sourceAPI)

		if err != nil {
			s.l.Error("Error while accessing Tmdb API: %v", err)
			data.ErrorMessage = append(data.ErrorMessage, err.Error())
			return nil, err
		}

		return results, nil
	}
}
