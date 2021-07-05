package web

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/zorchenhimer/MoviePolls/logic"
	"github.com/zorchenhimer/MoviePolls/models"
)

func (s *webServer) handlerAdminHome(w http.ResponseWriter, r *http.Request) {
	user := s.getSessionUser(w, r)
	if !s.backend.CheckAdminRights(user) {
		if s.debug {
			s.doError(http.StatusUnauthorized, "You are not an admin.", w, r)
		}
		s.doError(http.StatusNotFound, fmt.Sprintf("%q not found", r.URL.Path), w, r)
		return
	}

	cycle, err := s.backend.GetCurrentCycle()
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

func (s *webServer) handlerAdminUsers(w http.ResponseWriter, r *http.Request) {
	user := s.getSessionUser(w, r)
	if !s.backend.CheckAdminRights(user) {
		if s.debug {
			s.doError(http.StatusUnauthorized, "You are not an admin.", w, r)
		}
		s.doError(http.StatusNotFound, fmt.Sprintf("%q not found", r.URL.Path), w, r)
		return
	}

	ulist, err := s.backend.GetUsers(-1, 100)
	if err != nil {
		s.doError(
			http.StatusInternalServerError,
			fmt.Sprintf("Error getting users: %v", err),
			w, r)
		return
	}

	data := struct {
		dataPageBase

		Users []*models.User
	}{
		dataPageBase: s.newPageBase("Admin - Users", w, r),
		Users:        ulist,
	}

	if err := s.executeTemplate(w, "adminUsers", data); err != nil {
		s.l.Error("Error rendering template: %v", err)
	}
}

func (s *webServer) handlerAdminUserEdit(w http.ResponseWriter, r *http.Request) {
	user := s.getSessionUser(w, r)
	if !s.backend.CheckAdminRights(user) {
		if s.debug {
			s.doError(http.StatusUnauthorized, "You are not an admin.", w, r)
		}
		s.doError(http.StatusNotFound, fmt.Sprintf("%q not found", r.URL.Path), w, r)
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

	user, err = s.backend.GetUser(uid)
	if err != nil {
		s.doError(
			http.StatusBadRequest,
			fmt.Sprintf("Cannot get user: %v", err),
			w, r)
		return
	}

	action := r.URL.Query().Get("action")
	var urlKey *models.UrlKey
	switch action {
	//case "edit":
	//	// current function
	case "delete":

		confirm := r.URL.Query().Get("confirm")
		if confirm == "yes" {

			origName := user.Name
			s.backend.AdminDeleteUser(user)
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

		return
	case "ban":
		s.backend.AdminBanUser(user)
		return
	case "purge":
		confirm := r.URL.Query().Get("confirm")
		if confirm == "yes" {
			origName := user.Name
			err := s.backend.AdminPurgeUser(user)
			if err != nil {
				s.doError(
					http.StatusBadRequest,
					fmt.Sprintf("Could not purge user: %v", err),
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

		return
	case "password":
		urlKey, err = models.NewPasswordResetKey(user.Id)
		if err != nil {
			s.l.Error("Unable to generate UrlKey pair for user password reset: %v", err)
			s.doError(
				http.StatusBadRequest,
				fmt.Sprintf("Unable to generate UrlKey pair for user password reset: %v", err),
				w, r)
			return
		}

		s.l.Debug("Saving new urlKey with URL %s", urlKey.Url)
		s.backend.SetUrlKey(urlKey.Url, urlKey)
	}

	totalVotes, err := s.backend.GetMaxUserVotes()
	if err != nil {
		s.l.Error("Unable to get max user votes: %v", err)
	}

	activeVotes, _, err := s.backend.GetUserVotes(user)
	if err != nil {
		s.l.Error("Unable to get votes for user %d: %v", user.Id, err)
	}

	host, err := s.backend.GetHostAddress()
	if err != nil {
		s.doError(
			http.StatusInternalServerError,
			fmt.Sprintf("Unable to get server host address: %v", err),
			w, r)
		return
	}

	data := struct {
		dataPageBase

		User         *models.User
		CurrentVotes []*models.Movie
		//PastVotes      []*common.Movie
		AvailableVotes int

		PassError   []string
		NotifyError []string
		UrlKey      *models.UrlKey
		Host        string
	}{
		dataPageBase: s.newPageBase("Admin - User Edit", w, r),

		User:         user,
		CurrentVotes: activeVotes,
		//PastVotes:      watchedVotes,
		AvailableVotes: totalVotes - len(activeVotes),
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

func (s *webServer) handlerAdminConfig(w http.ResponseWriter, r *http.Request) {
	user := s.getSessionUser(w, r)
	if !s.backend.CheckAdminRights(user) {
		if s.debug {
			s.doError(http.StatusUnauthorized, "You are not an admin.", w, r)
		}
		s.doError(http.StatusNotFound, fmt.Sprintf("%q not found", r.URL.Path), w, r)
		return
	}

	data := struct {
		dataPageBase

		ErrorMessage []string
		Values       map[string]logic.ConfigValue
		Sections     []string

		TypeString     logic.ConfigValueType
		TypeStringPriv logic.ConfigValueType
		TypeBool       logic.ConfigValueType
		TypeInt        logic.ConfigValueType
	}{
		ErrorMessage: []string{},
		Values:       logic.ConfigValues,
		Sections:     logic.ConfigSections,

		TypeString:     logic.ConfigString,
		TypeStringPriv: logic.ConfigStringPriv,
		TypeBool:       logic.ConfigBool,
		TypeInt:        logic.ConfigInt,
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

		for key, val := range data.Values {
			str := r.PostFormValue(key)
			switch val.Type {
			case logic.ConfigString, logic.ConfigStringPriv:
				err = s.backend.SetCfgString(key, str)
				if err != nil {
					data.ErrorMessage = append(
						data.ErrorMessage,
						fmt.Sprintf("Unable to save string value for %s: %v", key, err))
				}

			case logic.ConfigInt:
				intVal, err := strconv.ParseInt(str, 10, 32)
				if err != nil {
					data.ErrorMessage = append(
						data.ErrorMessage,
						fmt.Sprintf("Value for %q is invalid: %v", key, err))
				} else {
					err = s.backend.SetCfgInt(key, int(intVal))
					if err != nil {
						data.ErrorMessage = append(
							data.ErrorMessage,
							fmt.Sprintf("Unable to save int value for %s: %v", key, err))
					}
				}

			case logic.ConfigBool:
				boolVal := false
				if str != "" {
					boolVal = true
				}
				err = s.backend.SetCfgBool(key, boolVal)
				if err != nil {
					data.ErrorMessage = append(
						data.ErrorMessage,
						fmt.Sprintf("Unable to save bool value for %s: %v", key, err))
				}

			default:
				data.ErrorMessage = append(
					data.ErrorMessage,
					fmt.Sprintf("Unknown config value type for %s: %v", key, val.Type))
			}
		}

		// Don't enable this stuff for now
		//if clearPassSalt := r.PostFormValue("ClearPassSalt"); clearPassSalt != "" {
		//	s.data.DeleteCfgKey("PassSalt")
		//}

		//if clearCookies := r.PostFormValue("ClearCookies"); clearCookies != "" {
		//	s.data.DeleteCfgKey("SessionAuth")
		//	s.data.DeleteCfgKey("SessionEncrypt")
		//}
	}

	newValues := map[string]logic.ConfigValue{}
	for key, val := range data.Values {
		var v interface{}

		switch val.Type {
		case logic.ConfigString, logic.ConfigStringPriv:
			v, err = s.backend.GetCfgString(key, val.Default.(string))
		case logic.ConfigBool:
			v, err = s.backend.GetCfgBool(key, val.Default.(bool))
		case logic.ConfigInt:
			v, err = s.backend.GetCfgInt(key, val.Default.(int))
		}

		if err != nil {
			data.ErrorMessage = append(
				data.ErrorMessage,
				fmt.Sprintf("Unable to get config value for %s: %v", key, err))
		} else {
			val.Value = v
		}

		newValues[key] = val
	}
	data.Values = newValues

	// Set this down here so the Notice Banner is updated
	data.dataPageBase = s.newPageBase("Admin - Config", w, r)

	// Reload Oauth
	err = s.initOauth()

	if err != nil {
		data.ErrorMessage = append(data.ErrorMessage, err.Error())
	}

	// getting ALL the booleans
	var localSignup, twitchSignup, patreonSignup, discordSignup, twitchOauth, patreonOauth, discordOauth bool

	for key, val := range data.Values {
		bval, ok := val.Value.(bool)
		if !ok && val.Type == logic.ConfigBool {
			data.ErrorMessage = append(data.ErrorMessage, "Could not parse field %s as boolean", key)
			break
		}
		switch key {
		case logic.ConfigLocalSignupEnabled:
			localSignup = bval
		case logic.ConfigTwitchOauthSignupEnabled:
			twitchSignup = bval
		case logic.ConfigTwitchOauthEnabled:
			twitchOauth = bval
		case logic.ConfigDiscordOauthSignupEnabled:
			discordSignup = bval
		case logic.ConfigDiscordOauthEnabled:
			discordOauth = bval
		case logic.ConfigPatreonOauthSignupEnabled:
			patreonSignup = bval
		case logic.ConfigPatreonOauthEnabled:
			patreonOauth = bval
		}
	}

	// Check that we have atleast ONE signup method enabled
	if !(localSignup || twitchSignup || discordSignup || patreonSignup) {
		data.ErrorMessage = append(data.ErrorMessage, "No Signup method is currently enabled, please ensure to enable atleast one method")
	}

	// Check that the corresponding oauth for the signup is enabled
	if twitchSignup && !twitchOauth {
		data.ErrorMessage = append(data.ErrorMessage, "To enable twitch signup you need to also enable twitch Oauth (and fill the token/secret)")
	}

	if discordSignup && !discordOauth {
		data.ErrorMessage = append(data.ErrorMessage, "To enable discord signup you need to also enable discord Oauth (and fill the token/secret)")
	}

	if patreonSignup && !patreonOauth {
		data.ErrorMessage = append(data.ErrorMessage, "To enable patreon signup you need to also enable patreon Oauth (and fill the token/secret)")
	}

	users, err := s.backend.GetUsersWithAuth(models.AUTH_TWITCH, true)
	if err, ok := err.(*models.ErrNoUsersFound); !ok || err == nil {
		if (len(users) != -1) && !twitchOauth {
			data.ErrorMessage = append(data.ErrorMessage, fmt.Sprintf("Disabling Twitch Oauth would cause %d users to be unable to login since they only have this auth method associated.", len(users)))
		}
	}

	users, err = s.backend.GetUsersWithAuth(models.AUTH_PATREON, true)
	if err, ok := err.(*models.ErrNoUsersFound); !ok || err == nil {
		if (len(users) != -1) && !patreonOauth {
			data.ErrorMessage = append(data.ErrorMessage, fmt.Sprintf("Disabling Patreon Oauth would cause %d users to be unable to login since they only have this auth method associated.", len(users)))
		}
	}

	users, err = s.backend.GetUsersWithAuth(models.AUTH_DISCORD, true)
	if err, ok := err.(*models.ErrNoUsersFound); !ok || err == nil {
		if (len(users) != -1) && !discordOauth {
			data.ErrorMessage = append(data.ErrorMessage, fmt.Sprintf("Disabling Discord Oauth would cause %d users to be unable to login since they only have this auth method associated.", len(users)))
		}
	}

	if err := s.executeTemplate(w, "adminConfig", data); err != nil {
		s.l.Error("Error rendering template: %v", err)
	}
}

func (s *webServer) handlerAdminMovieEdit(w http.ResponseWriter, r *http.Request) {
	user := s.getSessionUser(w, r)
	if !s.backend.CheckAdminRights(user) {
		if s.debug {
			s.doError(http.StatusUnauthorized, "You are not an admin.", w, r)
		}
		s.doError(http.StatusNotFound, fmt.Sprintf("%q not found", r.URL.Path), w, r)
		return
	}

	var mid int
	_, err := fmt.Sscanf(r.URL.Path, "/admin/movie/%d", &mid)
	if err != nil {
		s.l.Error("Unable to parse movie ID: %v", err)
		s.doError(
			http.StatusBadRequest,
			fmt.Sprintf("Unable to parse movie ID: %v", err),
			w, r)
		return
	}

	// TODO: Approve and Deny actions
	action := r.URL.Query().Get("action")
	switch action {
	case "remove":
		// TODO: Confirmation before removing
		err = s.backend.DeleteMovie(mid)
		if err != nil {
			s.l.Error("Unable to remove movie with ID %d: %v", mid, err)
			s.doError(
				http.StatusBadRequest,
				fmt.Sprintf("Unable to remove movie with ID %d: %v", mid, err),
				w, r)
			return
		}

		http.Redirect(w, r, "/admin/movies", http.StatusSeeOther)
		return
	}

	if r.Method == "POST" {
		err = r.ParseMultipartForm(4095)
		if err != nil {
			s.l.Error("Unable to parse form: %v", err)
			s.doError(
				http.StatusInternalServerError,
				fmt.Sprintf("Unable to parse form: %v", err),
				w, r)
			return
		}

		movie := s.backend.GetMovie(mid)

		movie.Name = r.PostFormValue("MovieName")
		movie.Description = r.PostFormValue("MovieDescr")

		linktext := strings.ReplaceAll(r.FormValue("MovieLinks"), "\r", "")

		links := strings.Split(linktext, "\n")
		// links is a slice of strings now
		s.l.Debug("Links: %s", links)

		linkstructs := []*models.Link{}

		// Convert links to structs
		for id, link := range links {

			ls, err := models.NewLink(link, id)

			if err != nil || ls == nil {
				s.l.Error("Cannot add link: %v", link)
				continue
			}

			id, err := s.backend.AddLink(ls)
			if err != nil {
				s.l.Error("Could not add link struct to db: %v", err.Error())
				continue
			}
			ls.Id = id

			linkstructs = append(linkstructs, ls)
		}
		movie.Links = linkstructs

		posterFileName := strings.TrimSpace(r.FormValue("MovieName"))
		posterFile, _, _ := r.FormFile("PosterFile")

		if posterFile != nil {
			file, err := s.uploadFile(r, posterFileName)

			if err != nil {
				//data.ErrPoster = true
				//errText = append(errText, err.Error())
				s.l.Error("Unable to upload file: %v", err)
			} else {
				movie.Poster = filepath.Base(file)
			}
		}

		err = s.backend.UpdateMovie(movie)
		if err != nil {
			s.l.Error("Unable to update movie: %v", err)
		}
	}

	movie := s.backend.GetMovie(mid)

	linktext := ""
	for _, link := range movie.Links {
		linktext = linktext + link.Url + "\n"
	}

	data := struct {
		dataPageBase
		Movie    *models.Movie
		LinkText string
	}{
		dataPageBase: s.newPageBase("Admin - Movies", w, r),
		Movie:        movie,
		LinkText:     linktext,
	}

	if err := s.executeTemplate(w, "adminMovieEdit", data); err != nil {
		s.l.Error("Error rendering template: %v", err)
	}
}

func (s *webServer) handlerAdminMovies(w http.ResponseWriter, r *http.Request) {
	user := s.getSessionUser(w, r)
	if !s.backend.CheckAdminRights(user) {
		if s.debug {
			s.doError(http.StatusUnauthorized, "You are not an admin.", w, r)
		}
		s.doError(http.StatusNotFound, fmt.Sprintf("%q not found", r.URL.Path), w, r)
		return
	}

	active, err := s.backend.GetActiveMovies()
	if err != nil {
		s.doError(
			http.StatusInternalServerError,
			fmt.Sprintf("Unable to get active movies: %v", err),
			w, r)
		return
	}

	approval, err := s.backend.GetEntriesRequireApproval()
	if err != nil {
		s.doError(
			http.StatusInternalServerError,
			fmt.Sprintf("Unable to get entries require approval setting: %v", err),
			w, r)
		return
	}

	data := struct {
		dataPageBase
		Active  []*models.Movie
		Past    []*models.Movie
		Pending []*models.Movie

		RequireApproval bool
	}{
		dataPageBase: s.newPageBase("Admin - Movies", w, r),
		Active:       models.SortMoviesByName(active),
		//Pending:      common.SortMoviesByName(active),

		RequireApproval: approval,
	}

	if err := s.executeTemplate(w, "adminMovies", data); err != nil {
		s.l.Error("Error rendering template: %v", err)
	}
}

func (s *webServer) handlerAdminCycles_Post(w http.ResponseWriter, r *http.Request) {
	user := s.getSessionUser(w, r)
	if !s.backend.CheckAdminRights(user) {
		if s.debug {
			s.doError(http.StatusUnauthorized, "You are not an admin.", w, r)
		}
		s.doError(http.StatusNotFound, fmt.Sprintf("%q not found", r.URL.Path), w, r)
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

		cycle, err := s.backend.GetCurrentCycle()
		if err != nil {
			s.l.Error(err.Error())
			s.doError(http.StatusInternalServerError, fmt.Sprintf("Unable to get current cycle: %v", err), w, r)
			return
		}

		end, err := time.Parse("2005-01-02", dateStr)
		if err != nil {
			s.l.Error(err.Error())
		} else {
			t := (&end).Round(time.Second)
			cycle.PlannedEnd = &t
		}

		err = s.backend.UpdateCycle(cycle)
		if err != nil {
			s.l.Error(err.Error())
			s.doError(http.StatusInternalServerError, fmt.Sprintf("Unable to get current cycle: %v", err), w, r)
			return
		}

	case "create":
		end, err := time.Parse("2005-01-02", r.PostFormValue("endDate"))
		if err != nil {
			s.l.Error(err.Error())
		} else {
			t := (&end).Round(time.Second)
			plannedEnd = &t
		}

		_, err = s.backend.AddCycle(plannedEnd)
		if err != nil {
			s.l.Error("Unable to add cycle: %v", err)
			s.doError(http.StatusInternalServerError, fmt.Sprintf("Unable to add cycle: %v", err), w, r)
			return
		}

		// Re-enable voting after successfully starting a new cycle
		err = s.backend.EnableVoting()
		if err != nil {
			s.doError(http.StatusInternalServerError, fmt.Sprintf("Unable to enable voting: %v", err), w, r)
			return
		}
	}

	http.Redirect(w, r, "/admin/cycles", http.StatusSeeOther)
}

func (s *webServer) handlerAdminCycles(w http.ResponseWriter, r *http.Request) {
	user := s.getSessionUser(w, r)
	if !s.backend.CheckAdminRights(user) {
		if s.debug {
			s.doError(http.StatusUnauthorized, "You are not an admin.", w, r)
		}
		s.doError(http.StatusNotFound, fmt.Sprintf("%q not found", r.URL.Path), w, r)
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
		err := s.backend.EnableVoting()
		if err != nil {
			s.doError(http.StatusInternalServerError, fmt.Sprintf("Unable to enable voting: %v", err), w, r)
			return
		}

		r.Method = "GET"
		http.Redirect(w, r, "/admin/cycles", http.StatusSeeOther)
		return

	case "select":
		s.cycleStage1(w, r)
		return
	}

	cycle, err := s.backend.GetCurrentCycle()
	if err != nil {
		s.doError(http.StatusInternalServerError, fmt.Sprintf("Unable to get current cycle: %v", err), w, r)
		return
	}

	data := struct {
		dataPageBase
		Cycle *models.Cycle
		Past  []*models.Cycle
	}{
		dataPageBase: s.newPageBase("Admin - Cycles", w, r),

		Cycle: cycle,
		Past:  []*models.Cycle{},
	}

	pastCycles, err := s.backend.GetPastCycles(0, 5)
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
func (s *webServer) cycleStage1(w http.ResponseWriter, r *http.Request) {
	s.l.Debug("cycleStage1")
	err := s.backend.DisableVoting()
	if err != nil {
		s.doError(http.StatusInternalServerError, fmt.Sprintf("Unable to disable voting: %v", err), w, r)
		return
	}

	movies, err := s.backend.GetActiveMovies()
	if err != nil {
		s.doError(http.StatusInternalServerError, fmt.Sprintf("Unable to get active movies: %v", err), w, r)
		return
	}

	//err = s.data.SetCfgString("CycleStage", "ended")
	//if err != nil {
	//	s.doError(http.StatusInternalServerError, fmt.Sprintf("Unable to set CycleStage: %v", err), w, r)
	//	return
	//}

	currentCycle, err := s.backend.GetCurrentCycle()
	if err != nil || currentCycle == nil {
		s.doError(http.StatusInternalServerError, fmt.Sprintf("Unable to get current cycle: %v", err), w, r)
		return
	}

	err = s.backend.EndCycle(currentCycle.Id)
	if err != nil {
		s.doError(http.StatusInternalServerError, fmt.Sprintf("Unable to set ending cycle ID: %v", err), w, r)
		return
	}

	data := struct {
		dataPageBase

		Movies []*models.Movie
		Stage  int
	}{
		dataPageBase: s.newPageBase("Admin - End Cycle", w, r),

		Movies: models.SortMoviesByVotes(movies),
		Stage:  1,
	}

	if err := s.executeTemplate(w, "adminEndCycle", data); err != nil {
		s.l.Error("Error rendering template: %v", err)
	}
}

func (s *webServer) cycleStage2(w http.ResponseWriter, r *http.Request) {
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

	cycle, err := s.backend.GetCurrentCycle()
	if err != nil {
		s.doError(http.StatusInternalServerError, fmt.Sprintf("Unable to get current cycle: %v", err), w, r)
		return
	}

	movies := []*models.Movie{}

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
			movie := s.backend.GetMovie(id)

			movies = append(movies, movie)
		}
	}

	// Set movie as "watched" today
	watched := time.Now().Local().Round(time.Hour)

	if val := r.PostFormValue("OverrideEndDate"); val != "" {
		newEnd, err := time.Parse("2006-01-02", r.PostFormValue("NewEndDate"))
		if err != nil {
			s.l.Error("Unable to parse new end date: %q: %v", r.PostFormValue("NewEndDate"), err)
		} else {
			watched = newEnd
		}
	}

	for _, movie := range movies {
		s.l.Debug("> setting watched on %s", movie.Name)
		movie.CycleWatched = cycle
		err = s.backend.UpdateMovie(movie)
		if err != nil {
			s.l.Error("Unable to update movie with ID %d: %v", movie.Id, err)
			continue
		}
	}

	cycle.Ended = &watched
	if err = s.backend.UpdateCycle(cycle); err != nil {
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
