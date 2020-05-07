package moviepoll

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
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
			s.data.SetCfgInt("MaxUserVotes", int(maxVotes))
		}

		appReqStr := r.PostFormValue("EntriesRequireApproval")
		if appReqStr != "" {
			s.data.SetCfgInt("EntriesRequireApproval", int(maxVotes))
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
			s.data.SetCfgBool("VotingEnabled", false)
		} else {
			s.data.SetCfgBool("VotingEnabled", true)
		}

		tmdbToken := r.PostFormValue("TmdbToken")
		s.data.SetCfgString("TmdbToken", tmdbToken)

	}

	data.MaxUserVotes, err = s.data.GetCfgInt("MaxUserVotes", DefaultMaxUserVotes)
	if err != nil {
		fmt.Printf("Error getting configuration value for MaxUserVotes: %s\n", err)

		err = s.data.SetCfgInt("MaxUserVotes", data.MaxUserVotes)
		if err != nil {
			fmt.Printf("Error saving new configuration value for MaxUserVotes: %s\n", err)
		}
	}

	data.EntriesRequireApproval, err = s.data.GetCfgBool("EntriesRequireApproval", DefaultEntriesRequireApproval)
	if err != nil {
		fmt.Printf("Error getting configuration value for EntriesRequireApproval: %s\n", err)

		err = s.data.SetCfgBool("EntriesRequireApproval", data.EntriesRequireApproval)
		if err != nil {
			fmt.Printf("Error saving new configuration value for EntriesRequireApproval: %s\n", err)
		}
	}

	data.VotingEnabled, err = s.data.GetCfgBool("VotingEnabled", DefaultVotingEnabled)
	if err != nil {
		fmt.Printf("Error getting configuration value for VotingEnabled: %s\n", err)

		// try to resave new value
		err = s.data.SetCfgBool("VotingEnabled", data.VotingEnabled)
		if err != nil {
			fmt.Printf("Error saving new configuration value for VotingEnabled: %s\n", err)
		}
	}

	data.TmdbToken, err = s.data.GetCfgString("TmdbToken", DefaultTmdbToken)
	if err != nil {
		fmt.Printf("Error getting configuration value for TmdbToken: %s\n", err)

		// try to resave new value
		err = s.data.SetCfgString("TmdbToken", data.TmdbToken)
		if err != nil {
			fmt.Printf("Error saving new configuration value for TmdbToken: %s\n", err)
		}
	}

	if err := s.executeTemplate(w, "adminConfig", data); err != nil {
		fmt.Printf("Error rendering template: %v\n", err)
	}
}

func (s *Server) handlerAdminCycles(w http.ResponseWriter, r *http.Request) {
	if !s.checkAdminRights(w, r) {
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

		cycle := &common.Cycle{}

		start, err := time.Parse("2006-01-02", r.PostFormValue("startDate"))
		if err != nil {
			fmt.Println(err)
		}

		cycle.Start = start

		end, err := time.Parse("2006-01-02", r.PostFormValue("endDate"))
		if err != nil {
			fmt.Println(err)
		} else {
			cycle.End = &end
		}

		fmt.Printf("start: %s\nend: %s\n", start, end)

		_, err = s.data.AddOldCycle(cycle)
		if err != nil {
			fmt.Printf("Unable to add cycle: %v\n", err)
			s.doError(http.StatusInternalServerError, fmt.Sprintf("Unable to add cycle: %v", err), w, r)
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
	}{
		dataPageBase: s.newPageBase("Admin - Cycles", w, r),
		Cycle:        cycle,
	}

	if err := s.executeTemplate(w, "adminCycles", data); err != nil {
		fmt.Printf("Error rendering template: %v\n", err)
	}
}

func (s *Server) handlerAdminEndCycle(w http.ResponseWriter, r *http.Request) {
	if !s.checkAdminRights(w, r) {
		return
	}

	movies, err := s.data.GetActiveMovies()
	if err != nil {
		s.doError(http.StatusInternalServerError, fmt.Sprintf("Unable to get active movies: %v", err), w, r)
		return
	}

	ml := common.MovieList(movies)
	sort.Sort(sort.Reverse(ml))

	data := struct {
		dataPageBase
		Movies []*common.Movie
	}{
		dataPageBase: s.newPageBase("Admin - End Cycle", w, r),
		Movies:       ml,
	}

	if err := s.executeTemplate(w, "adminEndCycle", data); err != nil {
		fmt.Printf("Error rendering template: %v\n", err)
	}
}
