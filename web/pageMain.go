package web

import (
	"fmt"
	"net/http"

	//"strings"

	"github.com/zorchenhimer/MoviePolls/models"
)

func (s *webServer) handlerPageMain(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		s.doError(http.StatusNotFound, fmt.Sprintf("%q not found", r.URL.Path), w, r)
		return
	}

	movieList := []*models.Movie{}

	data := struct {
		dataPageBase
		Movies         []*models.Movie
		VotingEnabled  bool
		AvailableVotes int
		LastCycle      *models.Cycle
		Cycle          *models.Cycle
	}{
		dataPageBase: s.newPageBase("Current Cycle", w, r),
	}

	if r.Body != http.NoBody {
		err := r.ParseForm()
		if err != nil {
			s.l.Error(err.Error())
		}
		searchVal := r.FormValue("search")

		movieList, err = s.backend.SearchMovieTitles(searchVal)
		if err != nil {
			s.l.Error(err.Error())
		}
	} else {
		var err error = nil
		movieList, err = s.backend.GetActiveMovies()
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
		val, err := s.backend.GetAvailableVotes(data.User)
		if err != nil {
			s.doError(
				http.StatusBadRequest,
				fmt.Sprintf("Cannot get user votes :C"),
				w, r)
			s.l.Error("Unable to get votes for user %d: %v", data.User.Id, err)
			return
		}

		data.AvailableVotes = val
	}

	data.Movies = models.SortMoviesByVotes(movieList)
	data.VotingEnabled = s.backend.GetVotingEnabled()
	data.LastCycle = s.backend.GetPreviousCycle()

	cycle, err := s.backend.GetCurrentCycle()
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
