package logic

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/zorchenhimer/MoviePolls/common"
)

type dataAdminHome struct {
	dataPageBase

	Cycle *common.Cycle
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

// "deletes" a user.  The account will still exist along with the votes, but
// the name, password, email, and notification settings will all be removed.
func (s *Server) adminDeleteUser(w http.ResponseWriter, r *http.Request, user *common.User) {
	confirm := r.URL.Query().Get("confirm")
	if confirm == "yes" {
		s.l.Info("Deleting user %s", user)
		origName := user.Name
		user.Name = "[deleted]"
		for _, auth := range user.AuthMethods {
			s.data.DeleteAuthMethod(auth.Id)
		}
		user.AuthMethods = []*common.AuthMethod{}
		user.Email = ""
		user.NotifyCycleEnd = false
		user.NotifyVoteSelection = false
		user.Privilege = 0

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

type configValue struct {
	Key     string
	Default interface{}
	Value   interface{}
	Type    ConfigValueType
	Error   bool
}

type ConfigValueType int

const (
	ConfigInt ConfigValueType = iota
	ConfigString
	ConfigBool
	ConfigKey
)

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
