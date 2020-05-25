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

type dataAdminUsers struct {
	dataPageBase

	Users []*common.User
}

type dataAdminUserEdit struct {
	dataPageBase

	User           *common.User
	CurrentVotes   []*common.Movie
	AvailableVotes int

	PassError   []string
	NotifyError []string
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
		fmt.Printf("Error rendering template: %v\n", err)
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

	data := dataAdminUsers{
		dataPageBase: s.newPageBase("Admin - Users", w, r),
		Users:        ulist,
	}

	if err := s.executeTemplate(w, "adminUsers", data); err != nil {
		fmt.Printf("Error rendering template: %v\n", err)
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

	totalVotes, err := s.data.GetCfgInt("MaxUserVotes", DefaultMaxUserVotes)
	if err != nil {
		fmt.Printf("Error getting MaxUserVotes config setting: %v\n", err)
	}

	votes, err := s.data.GetUserVotes(uid)
	if err != nil {
		s.doError(
			http.StatusInternalServerError,
			fmt.Sprintf("Unable to get user votes: %v", err),
			w, r)
		return
	}

	data := dataAdminUserEdit{
		dataPageBase: s.newPageBase("Admin - User Edit", w, r),

		User:         user,
		CurrentVotes: votes,
	}
	data.AvailableVotes = totalVotes - len(data.CurrentVotes)

	if r.Method == "POST" {
		// do a thing
	}

	if err := s.executeTemplate(w, "adminUserEdit", data); err != nil {
		fmt.Printf("Error rendering template: %v\n", err)
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

		ErrMaxUserVotes bool
	}{
		dataPageBase: s.newPageBase("Admin - Config", w, r),
		ErrorMessage: []string{},
	}

	var err error

	if r.Method == "POST" {
		if err = r.ParseForm(); err != nil {
			fmt.Printf("Unable to parse form: %v\n", err)
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

	}

	data.MaxUserVotes, err = s.data.GetCfgInt("MaxUserVotes", DefaultMaxUserVotes)
	if err != nil {
		fmt.Printf("Error getting configuration value for MaxUserVotes: %s\n", err)

		err = s.data.SetCfgInt(ConfigMaxUserVotes, data.MaxUserVotes)
		if err != nil {
			fmt.Printf("Error saving new configuration value for MaxUserVotes: %s\n", err)
		}
	}

	data.EntriesRequireApproval, err = s.data.GetCfgBool("EntriesRequireApproval", DefaultEntriesRequireApproval)
	if err != nil {
		fmt.Printf("Error getting configuration value for EntriesRequireApproval: %s\n", err)

		err = s.data.SetCfgBool(ConfigEntriesRequireApproval, data.EntriesRequireApproval)
		if err != nil {
			fmt.Printf("Error saving new configuration value for EntriesRequireApproval: %s\n", err)
		}
	}

	data.VotingEnabled, err = s.data.GetCfgBool("VotingEnabled", DefaultVotingEnabled)
	if err != nil {
		fmt.Printf("Error getting configuration value for VotingEnabled: %s\n", err)

		// try to resave new value
		err = s.data.SetCfgBool(ConfigVotingEnabled, data.VotingEnabled)
		if err != nil {
			fmt.Printf("Error saving new configuration value for VotingEnabled: %s\n", err)
		}
	}

	data.TmdbToken, err = s.data.GetCfgString("TmdbToken", DefaultTmdbToken)
	if err != nil {
		fmt.Printf("Error getting configuration value for TmdbToken: %s\n", err)

		// try to resave new value
		err = s.data.SetCfgString(ConfigTmdbToken, data.TmdbToken)
		if err != nil {
			fmt.Printf("Error saving new configuration value for TmdbToken: %s\n", err)
		}
	}

	if err := s.executeTemplate(w, "adminConfig", data); err != nil {
		fmt.Printf("Error rendering template: %v\n", err)
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
		fmt.Printf("Error rendering template: %v\n", err)
	}
}

func (s *Server) handlerAdminCycles(w http.ResponseWriter, r *http.Request) {
	if !s.checkAdminRights(w, r) {
		return
	}

	var action string
	if r.Method == "POST" {
		action = r.PostFormValue("action")
	}

	// URL parameters override POST
	if val := r.URL.Query().Get("action"); val != "" {
		action = val
	}

	fmt.Printf("action: %q\n", r.URL.Query().Get("action"))
	switch action {
	case "end":
		//adminEndCycle(w, r)
		s.cycleStage1(w, r)
		return

	case "cancel":
		fmt.Println("Canceling cycle end")
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

	var err error

	if r.Method == "POST" {
		fmt.Println("Cycle post")
		if err = r.ParseForm(); err != nil {
			fmt.Printf("Unable to parse form: %v\n", err)
			s.doError(http.StatusInternalServerError, fmt.Sprintf("Unable to parse form: %v", err), w, r)
			return
		}

		var plannedEnd *time.Time
		end, err := time.Parse("2006-01-02", r.PostFormValue("endDate"))
		if err != nil {
			fmt.Println(err)
		} else {
			t := (&end).Round(time.Second)
			plannedEnd = &t
		}

		_, err = s.data.AddCycle(plannedEnd)
		if err != nil {
			fmt.Printf("Unable to add cycle: %v\n", err)
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
	fmt.Printf("found %d past cycles: %s\n", len(pastCycles), pastCycles)

	fmt.Println("Executing admin cycles template")
	if err := s.executeTemplate(w, "adminCycles", data); err != nil {
		fmt.Printf("Error rendering template: %v\n", err)
	}
}

// display movies to select
func (s *Server) cycleStage1(w http.ResponseWriter, r *http.Request) {
	fmt.Println("cycleStage1")
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
		fmt.Printf("Error rendering template: %v\n", err)
	}
}

func (s *Server) cycleStage2(w http.ResponseWriter, r *http.Request) {
	fmt.Println("cycleStage2")

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
	//fmt.Printf("sumbit value: %s\n", r.PostForm.Get("submit"))

	cycle, err := s.data.GetCurrentCycle()
	if err != nil {
		s.doError(http.StatusInternalServerError, fmt.Sprintf("Unable to get current cycle: %v", err), w, r)
		return
	}

	movies := []*common.Movie{}

	// Get movie IDs from checkboxes
	for key, vals := range r.PostForm {
		//fmt.Printf("%s : (%d) [%s]\n", key, len(vals), strings.Join(vals, " "))
		if len(vals) > 0 && strings.HasPrefix(key, "cb_") && vals[0] != "" {
			fmt.Println("scanning for ID")
			var id int
			_, err = fmt.Sscanf(key, "cb_%d", &id)
			if err != nil {
				fmt.Printf("Error scanning cb_<id> from %q: %v\n", key, err)
				continue
			}

			fmt.Printf("selecting movie %s: %d\n", key, id)
			movie, err := s.data.GetMovie(id)
			if err != nil {
				fmt.Printf("Unable to get movie with ID %d: %v", id, err)
				continue
			}

			movies = append(movies, movie)
		}
	}

	// Set movie as "watched" today
	watched := time.Now().Local().Round(time.Hour)
	for _, movie := range movies {
		fmt.Printf("> setting watched on %s\n", movie.Name)
		movie.CycleWatched = cycle
		err = s.data.UpdateMovie(movie)
		if err != nil {
			fmt.Printf("Unable to update movie with ID %d: %v", movie.Id, err)
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

	//if err := s.executeTemplate(w, "adminEndCycle", data); err != nil {
	//	fmt.Printf("Error rendering template: %v\n", err)
	//}

	// Redirect to admin page
	http.Redirect(w, r, "/admin/cycles", http.StatusSeeOther)
}
