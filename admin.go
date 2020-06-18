package moviepoll

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/zorchenhimer/MoviePolls/common"
)

type dataAdminHome struct {
	dataPageBase

	Cycle *common.Cycle
}

type dataAdminUserEdit struct {
	dataPageBase

	User           *common.User
	CurrentVotes   []*common.Movie
	AvailableVotes int

	PassError   []string
	NotifyError []string
	UrlKey      *common.UrlKey
	Host        string
}

func (s *Server) checkAdminRights(w http.ResponseWriter, r *http.Request) bool {
	user := s.getSessionUser(w, r)

	ok := true
	if user == nil || user.Privilege < common.PRIV_MOD {
		ok = false
	}

	if !ok {
		if s.debug {
			s.doError(http.StatusUnauthorized, "You are not an admin.", w, r)
			return false
		}
		s.doError(http.StatusNotFound, fmt.Sprintf("%q not found", r.URL.Path), w, r)
		return false
	}

	return true
}

func (s *Server) handlerAdmin(w http.ResponseWriter, r *http.Request) {
	if !s.checkAdminRights(w, r) {
		return
	}

	cycle, err := s.data.GetCurrentCycle()
	if err != nil {
		s.doError(http.StatusInternalServerError, fmt.Sprintf("Unable to get current cycle: %v", err), w, r)
		return
	}

	data := dataAdminHome{
		dataPageBase: s.newPageBase("Admin", w, r),

		Cycle: cycle,
	}

	if err := s.executeTemplate(w, "adminHome", data); err != nil {
		s.l.Error("Error rendering template: %v", err)
	}
}

func (s *Server) handlerAdminUsers(w http.ResponseWriter, r *http.Request) {
	if !s.checkAdminRights(w, r) {
		return
	}

	ulist, err := s.data.GetUsers(0, 100)
	if err != nil {
		s.doError(
			http.StatusInternalServerError,
			fmt.Sprintf("Error getting users: %v", err),
			w, r)
		return
	}

	data := struct {
		dataPageBase

		Users []*common.User
	}{
		dataPageBase: s.newPageBase("Admin - Users", w, r),
		Users:        ulist,
	}

	if err := s.executeTemplate(w, "adminUsers", data); err != nil {
		s.l.Error("Error rendering template: %v", err)
	}
}

// "deletes" a user.  The account will still exist along with the votes, but
// the name, password, email, and notification settings will all be removed.
func (s *Server) adminDeleteUser(w http.ResponseWriter, r *http.Request, user *common.User) {
	confirm := r.URL.Query().Get("confirm")
	if confirm == "yes" {
		s.l.Info("Deleting user %s", user)
		origName := user.Name
		user.Name = "[deleted]"
		user.Password = ""
		user.PassDate = time.Now()
		user.Email = ""
		user.NotifyCycleEnd = false
		user.NotifyVoteSelection = false
		user.Privilege = 0
		user.RateLimitOverride = false

		err := s.data.UpdateUser(user)
		if err != nil {
			s.doError(
				http.StatusBadRequest,
				fmt.Sprintf("Unable to update user: %v", err),
				w, r)
			return
		}

		data := struct {
			dataPageBase

			Message  string
			Link     string
			LinkText string
		}{
			dataPageBase: s.newPageBase("Admin - Delete User", w, r),

			Message:  fmt.Sprintf("The user %q has been removed.", origName),
			Link:     "/admin/users",
			LinkText: "Ok",
		}

		if err := s.executeTemplate(w, "adminNotice", data); err != nil {
			s.l.Error("Error rendering template: %v", err)
		}
		return
	}

	s.l.Info("Confirm deleting user %s", user)

	data := struct {
		dataPageBase

		Message      string
		TrueMessage  string
		FalseMessage string
		TrueLink     string
		FalseLink    string
	}{
		dataPageBase: s.newPageBase("Admin - Delete User", w, r),
		Message:      fmt.Sprintf("Are you sure you want to remove the account of %q?  Its votes will stay intact, but everything else will be cleared.", user.Name),
		TrueMessage:  "Delete",
		FalseMessage: "Cancel",
		TrueLink:     fmt.Sprintf("/admin/user/%d?action=delete&confirm=yes", user.Id),
		FalseLink:    "/admin/users",
	}

	if err := s.executeTemplate(w, "adminConfirm", data); err != nil {
		s.l.Error("Error rendering template: %v", err)
	}
}

// Ban deletes a user and adds them to a ban list.  Users on this list can view
// the site but cannot create an account.
func (s *Server) adminBanUser(w http.ResponseWriter, r *http.Request, user *common.User) {
	s.doError(
		http.StatusBadRequest,
		"Ban user not implemented yet.",
		w, r)
}

// Purge removes the account entirely, including all of the account's votes.
// Should this add the user to the banlist?  Maybe add an option?
func (s *Server) adminPurgeUser(w http.ResponseWriter, r *http.Request, user *common.User) {
	confirm := r.URL.Query().Get("confirm")
	if confirm == "yes" {
		s.l.Info("Purging user %s", user)
		origName := user.Name
		err := s.data.PurgeUser(user.Id)
		if err != nil {
			s.doError(
				http.StatusBadRequest,
				fmt.Sprintf("Unable to purge user: %v", err),
				w, r)
			return
		}

		data := struct {
			dataPageBase

			Message  string
			Link     string
			LinkText string
		}{
			dataPageBase: s.newPageBase("Admin - Purge User", w, r),

			Message:  fmt.Sprintf("The user %q has been purged.", origName),
			Link:     "/admin/users",
			LinkText: "Ok",
		}

		if err := s.executeTemplate(w, "adminNotice", data); err != nil {
			s.l.Error("Error rendering template: %v", err)
		}
		return
	}

	s.l.Info("Confirm purging user %s", user)
	data := struct {
		dataPageBase

		Message      string
		TrueMessage  string
		FalseMessage string
		TrueLink     string
		FalseLink    string
	}{
		dataPageBase: s.newPageBase("Admin - Perge User", w, r),
		Message:      fmt.Sprintf("Are you sure you want to PURGE the account of %q?  Votes will be deleted.", user.Name),
		TrueMessage:  "PURGE",
		FalseMessage: "Cancel",
		TrueLink:     fmt.Sprintf("/admin/user/%d?action=purge&confirm=yes", user.Id),
		FalseLink:    "/admin/users",
	}

	if err := s.executeTemplate(w, "adminConfirm", data); err != nil {
		s.l.Error("Error rendering template: %v", err)
	}
}

func (s *Server) handlerAdminUserEdit(w http.ResponseWriter, r *http.Request) {
	if !s.checkAdminRights(w, r) {
		return
	}

	var uid int
	_, err := fmt.Sscanf(r.URL.Path, "/admin/user/%d", &uid)
	if err != nil {
		s.doError(
			http.StatusBadRequest,
			fmt.Sprintf("Unable to parse user ID: %v", err),
			w, r)
		return
	}

	user, err := s.data.GetUser(uid)
	if err != nil {
		s.doError(
			http.StatusBadRequest,
			fmt.Sprintf("Cannot get user: %v", err),
			w, r)
		return
	}

	action := r.URL.Query().Get("action")
	var urlKey *common.UrlKey
	switch action {
	//case "edit":
	//	// current function
	case "delete":
		s.adminDeleteUser(w, r, user)
		return
	case "ban":
		s.adminBanUser(w, r, user)
		return
	case "purge":
		s.adminPurgeUser(w, r, user)
		return
	case "password":
		urlKey, err = common.NewPasswordResetKey(user.Id)
		if err != nil {
			s.l.Error("Unable to generate UrlKey pair for user password reset: %v", err)
			s.doError(
				http.StatusBadRequest,
				fmt.Sprintf("Unable to generate UrlKey pair for user password reset: %v", err),
				w, r)
			return
		}

		s.l.Debug("Saving new urlKey with URL %s", urlKey.Url)
		s.urlKeys[urlKey.Url] = urlKey
	}

	totalVotes, err := s.data.GetCfgInt("MaxUserVotes", DefaultMaxUserVotes)
	if err != nil {
		s.l.Error("Error getting MaxUserVotes config setting: %v", err)
	}

	votes, err := s.data.GetUserVotes(uid)
	if err != nil {
		s.doError(
			http.StatusInternalServerError,
			fmt.Sprintf("Unable to get user votes: %v", err),
			w, r)
		return
	}

	host, err := s.data.GetCfgString(ConfigHostAddress, "http://<host>")
	if err != nil {
		s.doError(
			http.StatusInternalServerError,
			fmt.Sprintf("Unable to get server host address: %v", err),
			w, r)
		return
	}

	data := dataAdminUserEdit{
		dataPageBase: s.newPageBase("Admin - User Edit", w, r),

		User:           user,
		CurrentVotes:   votes,
		AvailableVotes: totalVotes - len(votes),
		UrlKey:         urlKey,
		Host:           host,
	}

	// FIXME: implement this
	if r.Method == "POST" {
	}

	if err := s.executeTemplate(w, "adminUserEdit", data); err != nil {
		s.l.Error("Error rendering template: %v", err)
	}
}

func (s *Server) handlerAdminConfig(w http.ResponseWriter, r *http.Request) {
	if !s.checkAdminRights(w, r) {
		return
	}

	data := struct {
		dataPageBase

		ErrorMessage []string

		MaxUserVotes           int
		EntriesRequireApproval bool
		VotingEnabled          bool
		TmdbToken              string
		MaxNameLength          int
		MinNameLength          int
		NoticeMessage          string
		HostAddress            string

		ErrMaxUserVotes  bool
		ErrMaxNameLength bool
		ErrMinNameLength bool
	}{
		dataPageBase: s.newPageBase("Admin - Config", w, r),
		ErrorMessage: []string{},
	}

	var err error

	if r.Method == "POST" {
		if err = r.ParseForm(); err != nil {
			s.l.Error("Unable to parse form: %v", err)
			s.doError(
				http.StatusInternalServerError,
				fmt.Sprintf("Unable to parse form: %v", err),
				w, r)
			return
		}

		maxVotesStr := r.PostFormValue("MaxUserVotes")
		maxVotes, err := strconv.ParseInt(maxVotesStr, 10, 32)
		if err != nil {
			data.ErrorMessage = append(
				data.ErrorMessage,
				fmt.Sprintf("MaxUserVotes invalid: %v", err))
			data.ErrMaxUserVotes = true
		} else {
			s.data.SetCfgInt(ConfigMaxUserVotes, int(maxVotes))
		}

		appReqStr := r.PostFormValue("EntriesRequireApproval")
		if appReqStr == "" {
			s.data.SetCfgBool(ConfigEntriesRequireApproval, false)
		} else {
			s.data.SetCfgBool(ConfigEntriesRequireApproval, true)
		}

		clearPass := r.PostFormValue("ClearPassSalt")
		if clearPass != "" {
			s.data.DeleteCfgKey("PassSalt")
		}

		// I'm pretty sure this breaks things
		clearCookies := r.PostFormValue("ClearCookies")
		if clearCookies != "" {
			s.data.DeleteCfgKey("SessionAuth")
			s.data.DeleteCfgKey("SessionEncrypt")
		}

		votingEnabled := r.PostFormValue("VotingEnabled")
		if votingEnabled == "" {
			s.data.SetCfgBool(ConfigVotingEnabled, false)
		} else {
			s.data.SetCfgBool(ConfigVotingEnabled, true)
		}

		tmdbToken := r.PostFormValue("TmdbToken")
		s.data.SetCfgString(ConfigTmdbToken, tmdbToken)

		maxNameLengthStr := r.PostFormValue("MaxNameLength")
		maxNameLength, err := strconv.ParseInt(maxNameLengthStr, 10, 32)
		if err != nil {
			data.ErrMaxNameLength = true
			data.ErrorMessage = append(
				data.ErrorMessage,
				fmt.Sprintf("MaxNameLength invalid: %v", err))

		} else if maxNameLength < 10 {
			data.ErrMaxNameLength = true
			data.ErrorMessage = append(
				data.ErrorMessage,
				"Max name length must be at least 10")

		} else {
			err = s.data.SetCfgInt(ConfigMaxNameLength, int(maxNameLength))
			if err != nil {
				s.l.Error("Unable to set %q: %v", ConfigMaxNameLength, err)
			}
		}

		minNameLengthStr := r.PostFormValue("MinNameLength")
		minNameLength, err := strconv.ParseInt(minNameLengthStr, 10, 32)
		if err != nil {
			data.ErrMinNameLength = true
			data.ErrorMessage = append(
				data.ErrorMessage,
				fmt.Sprintf("MinNameLength invalid: %v", err))

		} else if minNameLength < 4 {
			data.ErrMinNameLength = true
			data.ErrorMessage = append(
				data.ErrorMessage,
				"Min name length must be at least 4")

		} else {
			err = s.data.SetCfgInt(ConfigMinNameLength, int(minNameLength))
			if err != nil {
				s.l.Error("Unable to set %q: %v", ConfigMinNameLength, err)
			}
		}
		s.data.SetCfgString(ConfigNoticeBanner, strings.TrimSpace(r.PostFormValue("NoticeMessage")))

		hostAddress := strings.TrimSpace(r.PostFormValue("HostAddress"))
		err = s.data.SetCfgString(ConfigHostAddress, hostAddress)
		if err != nil {
			s.l.Error("Unable to set %q: %v", ConfigHostAddress, err)
		}
		s.l.Debug("hostAddress: %q", hostAddress)
	}

	// TODO: Show these errors in the UI.
	data.MaxUserVotes, err = s.data.GetCfgInt("MaxUserVotes", DefaultMaxUserVotes)
	if err != nil {
		s.l.Error("Error getting configuration value for MaxUserVotes: %s", err)

		err = s.data.SetCfgInt(ConfigMaxUserVotes, data.MaxUserVotes)
		if err != nil {
			s.l.Error("Error saving new configuration value for MaxUserVotes: %s", err)
		}
	}

	data.EntriesRequireApproval, err = s.data.GetCfgBool("EntriesRequireApproval", DefaultEntriesRequireApproval)
	if err != nil {
		s.l.Error("Error getting configuration value for EntriesRequireApproval: %s", err)

		err = s.data.SetCfgBool(ConfigEntriesRequireApproval, data.EntriesRequireApproval)
		if err != nil {
			s.l.Error("Error saving new configuration value for EntriesRequireApproval: %s", err)
		}
	}

	data.VotingEnabled, err = s.data.GetCfgBool("VotingEnabled", DefaultVotingEnabled)
	if err != nil {
		s.l.Error("Error getting configuration value for VotingEnabled: %s", err)

		// try to resave new value
		err = s.data.SetCfgBool(ConfigVotingEnabled, data.VotingEnabled)
		if err != nil {
			s.l.Error("Error saving new configuration value for VotingEnabled: %s", err)
		}
	}

	data.TmdbToken, err = s.data.GetCfgString("TmdbToken", DefaultTmdbToken)
	if err != nil {
		s.l.Error("Error getting configuration value for TmdbToken: %s", err)

		// try to resave new value
		err = s.data.SetCfgString(ConfigTmdbToken, data.TmdbToken)
		if err != nil {
			s.l.Error("Error saving new configuration value for TmdbToken: %s", err)
		}
	}

	data.MaxNameLength, err = s.data.GetCfgInt(ConfigMaxNameLength, DefaultMaxNameLength)
	if err != nil {
		s.l.Error("Error getting configuration value for %s: %v", ConfigMaxNameLength, err)

		err = s.data.SetCfgInt(ConfigMaxNameLength, DefaultMaxNameLength)
		if err != nil {
			s.l.Error("Unable to save configuration value for %s: %v", ConfigMaxNameLength, err)
		}
	}

	data.MinNameLength, err = s.data.GetCfgInt(ConfigMinNameLength, DefaultMinNameLength)
	if err != nil {
		s.l.Error("Error getting configuration value for %s: %v", ConfigMinNameLength, err)

		err = s.data.SetCfgInt(ConfigMinNameLength, DefaultMinNameLength)
		if err != nil {
			s.l.Error("Unable to save configuration value for %s: %v", ConfigMinNameLength, err)
		}
	}

	data.NoticeMessage, err = s.data.GetCfgString(ConfigNoticeBanner, "")
	if err != nil {
		s.l.Error("Error getting configuration value for %s: %v", ConfigNoticeBanner, err)

		err = s.data.SetCfgString(ConfigNoticeBanner, "")
		if err != nil {
			s.l.Error("Unable to save configuration value for %s: %v", ConfigNoticeBanner, err)
		}
	}

	data.HostAddress, err = s.data.GetCfgString(ConfigHostAddress, "")
	if err != nil {
		s.l.Error("Error getting configuration value for %s: %v", ConfigHostAddress, err)

		err = s.data.SetCfgString(ConfigHostAddress, "")
		if err != nil {
			s.l.Error("Unable to save configuration value for %s: %v", ConfigHostAddress, err)
		}
	}

	if err := s.executeTemplate(w, "adminConfig", data); err != nil {
		s.l.Error("Error rendering template: %v", err)
	}
}

func (s *Server) handlerAdminMovieEdit(w http.ResponseWriter, r *http.Request) {
	if !s.checkAdminRights(w, r) {
		return
	}

	var mid int
	_, err := fmt.Sscanf(r.URL.Path, "/admin/movie/%d", &mid)
	if err != nil {
		s.doError(
			http.StatusBadRequest,
			fmt.Sprintf("Unable to parse movie ID: %v", err),
			w, r)
		return
	}

	// TODO: this
	//action := r.URL.Query().Get("action")
	//switch action {
	//case "remove":
	//}

	if r.Method == "POST" {
		err = r.ParseMultipartForm(4096)
		if err != nil {
			s.l.Error("Unable to parse form: %v", err)
			s.doError(
				http.StatusInternalServerError,
				fmt.Sprintf("Unable to parse form: %v", err),
				w, r)
			return
		}

		movie, err := s.data.GetMovie(mid)
		if err != nil {
			s.doError(
				http.StatusBadRequest,
				fmt.Sprintf("Cannot get movie: %v", err),
				w, r)
			return
		}

		movie.Name = r.PostFormValue("MovieName")
		movie.Description = r.PostFormValue("MovieDescr")

		linktext := strings.ReplaceAll(r.FormValue("MovieLinks"), "\r", "")

		links := strings.Split(linktext, "\n")
		//links, err = common.VerifyLinks(links)
		//if err != nil {
		//	s.l.Error("bad link: %v", err)
		//	errText = append(errText, "Invalid link(s) given.")
		//} else {
		s.l.Debug("Links: %s", links)
		movie.Links = links
		//}

		posterFileName := strings.TrimSpace(r.FormValue("MovieName"))
		posterFile, _, _ := r.FormFile("PosterFile")
		if posterFile != nil {
			file, err := s.uploadFile(r, posterFileName)

			if err != nil {
				//data.ErrPoster = true
				//errText = append(errText, err.Error())
				s.l.Error("Unable to upload file: %v", err)
			} else {
				movie.Poster = file
			}
		}

		err = s.data.UpdateMovie(movie)
		if err != nil {
			s.l.Error("Unable to update movie: %v", err)
		}
	}

	movie, err := s.data.GetMovie(mid)
	if err != nil {
		s.doError(
			http.StatusBadRequest,
			fmt.Sprintf("Cannot get movie: %v", err),
			w, r)
		return
	}

	data := struct {
		dataPageBase
		Movie    *common.Movie
		LinkText string
	}{
		dataPageBase: s.newPageBase("Admin - Movies", w, r),
		Movie:        movie,
		LinkText:     strings.Join(movie.Links, "\n"),
	}

	if err := s.executeTemplate(w, "adminMovieEdit", data); err != nil {
		s.l.Error("Error rendering template: %v", err)
	}
}

func (s *Server) handlerAdminMovies(w http.ResponseWriter, r *http.Request) {
	if !s.checkAdminRights(w, r) {
		return
	}

	active, err := s.data.GetActiveMovies()
	if err != nil {
		s.doError(
			http.StatusInternalServerError,
			fmt.Sprintf("Unable to get active movies: %v", err),
			w, r)
		return
	}

	approval, err := s.data.GetCfgBool(ConfigEntriesRequireApproval, DefaultEntriesRequireApproval)
	if err != nil {
		s.doError(
			http.StatusInternalServerError,
			fmt.Sprintf("Unable to get entries require approval setting: %v", err),
			w, r)
		return
	}

	data := struct {
		dataPageBase
		Active  []*common.Movie
		Past    []*common.Movie
		Pending []*common.Movie

		RequireApproval bool
	}{
		dataPageBase: s.newPageBase("Admin - Movies", w, r),
		Active:       common.SortMoviesByName(active),
		//Pending:      common.SortMoviesByName(active),

		RequireApproval: approval,
	}

	if err := s.executeTemplate(w, "adminMovies", data); err != nil {
		s.l.Error("Error rendering template: %v", err)
	}
}

func (s *Server) handlerAdminCycles_Post(w http.ResponseWriter, r *http.Request) {
	if !s.checkAdminRights(w, r) {
		return
	}

	if r.Method != "POST" {
		http.Redirect(w, r, "/admin/cycles", http.StatusSeeOther)
		return
	}

	s.l.Debug("Cycle post")

	err := r.ParseForm()
	if err != nil {
		s.l.Error("Unable to parse form: %v", err)
		s.doError(http.StatusInternalServerError, fmt.Sprintf("Unable to parse form: %v", err), w, r)
		return
	}

	var plannedEnd *time.Time
	action := r.PostFormValue("actionType")
	s.l.Debug("Form action: %s", action)

	switch action {
	case "update":
		dateStr := strings.TrimSpace(r.PostFormValue("modEndDate"))
		if dateStr == "" {
			s.l.Debug("No date given in update")
			http.Redirect(w, r, "/admin/cycles", http.StatusSeeOther)
			return
		}

		cycle, err := s.data.GetCurrentCycle()
		if err != nil {
			s.l.Error(err.Error())
			s.doError(http.StatusInternalServerError, fmt.Sprintf("Unable to get current cycle: %v", err), w, r)
			return
		}

		end, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			s.l.Error(err.Error())
		} else {
			t := (&end).Round(time.Second)
			cycle.PlannedEnd = &t
		}

		err = s.data.UpdateCycle(cycle)
		if err != nil {
			s.l.Error(err.Error())
			s.doError(http.StatusInternalServerError, fmt.Sprintf("Unable to get current cycle: %v", err), w, r)
			return
		}

	case "create":
		end, err := time.Parse("2006-01-02", r.PostFormValue("endDate"))
		if err != nil {
			s.l.Error(err.Error())
		} else {
			t := (&end).Round(time.Second)
			plannedEnd = &t
		}

		_, err = s.data.AddCycle(plannedEnd)
		if err != nil {
			s.l.Error("Unable to add cycle: %v", err)
			s.doError(http.StatusInternalServerError, fmt.Sprintf("Unable to add cycle: %v", err), w, r)
			return
		}

		// Re-enable voting after successfully starting a new cycle
		err = s.data.SetCfgBool(ConfigVotingEnabled, true)
		if err != nil {
			s.doError(http.StatusInternalServerError, fmt.Sprintf("Unable to enable voting: %v", err), w, r)
			return
		}
	}

	http.Redirect(w, r, "/admin/cycles", http.StatusSeeOther)
}

func (s *Server) handlerAdminCycles(w http.ResponseWriter, r *http.Request) {
	if !s.checkAdminRights(w, r) {
		return
	}

	var action string
	if r.Method == "POST" {
		r.ParseForm()
		action = r.PostFormValue("action")
		s.l.Debug("POSTed values: %s", r.PostForm)
	}

	// URL parameters override POST
	if val := r.URL.Query().Get("action"); val != "" {
		action = val
	}

	s.l.Debug("action: %q", r.URL.Query().Get("action"))
	switch action {
	case "end":
		//adminEndCycle(w, r)
		s.cycleStage1(w, r)
		return

	case "cancel":
		s.l.Info("Canceling cycle end")
		err := s.data.SetCfgBool(ConfigVotingEnabled, true)
		if err != nil {
			s.doError(http.StatusInternalServerError, fmt.Sprintf("Unable to enable voting: %v", err), w, r)
			return
		}

		r.Method = "GET"
		http.Redirect(w, r, "/admin/cycles", http.StatusSeeOther)
		return

	case "select":
		s.cycleStage2(w, r)
		return
	}

	cycle, err := s.data.GetCurrentCycle()
	if err != nil {
		s.doError(http.StatusInternalServerError, fmt.Sprintf("Unable to get current cycle: %v", err), w, r)
		return
	}

	data := struct {
		dataPageBase
		Cycle *common.Cycle
		Past  []*common.Cycle
	}{
		dataPageBase: s.newPageBase("Admin - Cycles", w, r),

		Cycle: cycle,
		Past:  []*common.Cycle{},
	}

	pastCycles, err := s.data.GetPastCycles(0, 5)
	if err != nil {
		s.doError(http.StatusInternalServerError, fmt.Sprintf("Unable to get past cycles: %v", err), w, r)
		return
	}

	data.Past = pastCycles
	s.l.Debug("found %d past cycles: %s", len(pastCycles), pastCycles)

	s.l.Debug("Executing admin cycles template")
	if err := s.executeTemplate(w, "adminCycles", data); err != nil {
		s.l.Error("Error rendering template: %v", err)
	}
}

// display movies to select
func (s *Server) cycleStage1(w http.ResponseWriter, r *http.Request) {
	s.l.Debug("cycleStage1")
	err := s.data.SetCfgBool(ConfigVotingEnabled, false)
	if err != nil {
		s.doError(http.StatusInternalServerError, fmt.Sprintf("Unable to disable voting: %v", err), w, r)
		return
	}

	movies, err := s.data.GetActiveMovies()
	if err != nil {
		s.doError(http.StatusInternalServerError, fmt.Sprintf("Unable to get active movies: %v", err), w, r)
		return
	}

	//err = s.data.SetCfgString("CycleStage", "ended")
	//if err != nil {
	//	s.doError(http.StatusInternalServerError, fmt.Sprintf("Unable to set CycleStage: %v", err), w, r)
	//	return
	//}

	currentCycle, err := s.data.GetCurrentCycle()
	if err != nil || currentCycle == nil {
		s.doError(http.StatusInternalServerError, fmt.Sprintf("Unable to get current cycle: %v", err), w, r)
		return
	}

	err = s.data.SetCfgString("CycleEnding", fmt.Sprint(currentCycle.Id))
	if err != nil {
		s.doError(http.StatusInternalServerError, fmt.Sprintf("Unable to set ending cycle ID: %v", err), w, r)
		return
	}

	data := struct {
		dataPageBase

		Movies []*common.Movie
		Stage  int
	}{
		dataPageBase: s.newPageBase("Admin - End Cycle", w, r),

		Movies: common.SortMoviesByVotes(movies),
		Stage:  1,
	}

	if err := s.executeTemplate(w, "adminEndCycle", data); err != nil {
		s.l.Error("Error rendering template: %v", err)
	}
}

func (s *Server) cycleStage2(w http.ResponseWriter, r *http.Request) {
	s.l.Debug("cycleStage2")

	// No data received.  re-display list.
	if r.Method != "POST" {
		s.cycleStage1(w, r)
		return
	}

	//cycleId, err := s.data.GetCfgString("CycleEnding", "")
	//if err != nil {
	//	s.doError(http.StatusInternalServerError, fmt.Sprintf("Unable to get ending cycle ID: %v", err), w, r)
	//	return
	//}

	//var cId int
	//_, err = fmt.Sscanf(cycleId, "%d", &cId)
	//if err != nil {
	//	s.doError(http.StatusInternalServerError, fmt.Sprintf("invalid cycle id in CycleEnding key %q: %v", cycleId, err), w, r)
	//	return
	//}

	//cycle, err := s.data.GetCycle(cId)
	//if err != nil {
	//	s.doError(http.StatusInternalServerError, fmt.Sprintf("Unable to get cycle with ID %d: %v", cId, err), w, r)
	//	return
	//}

	var err error
	if err = r.ParseForm(); err != nil {
		s.doError(http.StatusInternalServerError, fmt.Sprintf("Parse form error: %v", err), w, r)
		return
	}
	//s.l.Debug("sumbit value: %s", r.PostForm.Get("submit"))

	cycle, err := s.data.GetCurrentCycle()
	if err != nil {
		s.doError(http.StatusInternalServerError, fmt.Sprintf("Unable to get current cycle: %v", err), w, r)
		return
	}

	movies := []*common.Movie{}

	// Get movie IDs from checkboxes
	for key, vals := range r.PostForm {
		//s.l.Debug("%s : (%d) [%s]", key, len(vals), strings.Join(vals, " "))
		if len(vals) > 0 && strings.HasPrefix(key, "cb_") && vals[0] != "" {
			s.l.Debug("scanning for ID")
			var id int
			_, err = fmt.Sscanf(key, "cb_%d", &id)
			if err != nil {
				s.l.Error("Error scanning cb_<id> from %q: %v", key, err)
				continue
			}

			s.l.Debug("selecting movie %s: %d", key, id)
			movie, err := s.data.GetMovie(id)
			if err != nil {
				s.l.Error("Unable to get movie with ID %d: %v", id, err)
				continue
			}

			movies = append(movies, movie)
		}
	}

	// Set movie as "watched" today
	watched := time.Now().Local().Round(time.Hour)
	for _, movie := range movies {
		s.l.Debug("> setting watched on %s", movie.Name)
		movie.CycleWatched = cycle
		err = s.data.UpdateMovie(movie)
		if err != nil {
			s.l.Error("Unable to update movie with ID %d: %v", movie.Id, err)
			continue
		}
	}

	cycle.Ended = &watched
	if err = s.data.UpdateCycle(cycle); err != nil {
		s.doError(http.StatusInternalServerError, fmt.Sprintf("Unable to update cycle: %v", err), w, r)
		return
	}

	// Clear status
	//err = s.data.SetCfgString("CycleStage", "")
	//if err != nil {
	//	s.doError(http.StatusInternalServerError, fmt.Sprintf("Unable to set CycleStage: %v", err), w, r)
	//	return
	//}

	// Redirect to admin page
	http.Redirect(w, r, "/admin/cycles", http.StatusSeeOther)
}
