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
	votingEnabled, err := s.backend.GetVotingEnabled()
	if err != nil {
		s.l.Error("Error getting VotingEnabled: %v", err)
	}
	data.VotingEnabled = votingEnabled
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

// This is here since i didnt find a better place ...
func (s *webServer) handlerVote(w http.ResponseWriter, r *http.Request) {
	user := s.getSessionUser(w, r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	enabled, err := s.backend.GetVotingEnabled()

	if !enabled || err != nil {
		s.doError(
			http.StatusBadRequest,
			"Voting is not enabled",
			w, r)
		return
	}

	var movieId int
	if _, err := fmt.Sscanf(r.URL.Path, "/vote/%d", &movieId); err != nil {
		s.doError(http.StatusBadRequest, "Invalid movie ID", w, r)
		s.l.Info("invalid vote URL: %q", r.URL.Path)
		return
	}

	movie := s.backend.GetMovie(movieId)

	if movie.CycleWatched != nil {
		s.doError(http.StatusBadRequest, "Movie already watched", w, r)
		s.l.Error("Attempted to vote on watched movie ID %d", movieId)
		return
	}

	userVoted, err := s.backend.UserVotedForMovie(user.Id, movieId)
	if err != nil {
		s.doError(http.StatusBadRequest, "Something went wrong :c", w, r)
		s.l.Error("Cannot get user vote: %v", err)
		return
	}

	if userVoted {
		//s.doError(http.StatusBadRequest, "You already voted for that movie!", w, r)
		if err := s.backend.DeleteVote(user.Id, movieId); err != nil {
			s.doError(http.StatusBadRequest, "Something went wrong :c", w, r)
			s.l.Error("Unable to remove vote: %v", err)
			return
		}
	} else {

		unlimited, err := s.backend.GetUnlimitedVotes()

		if err != nil {
			s.doError(
				http.StatusBadRequest,
				fmt.Sprintf("Cannot get UnlimitedVotes: %v", err),
				w, r)
			return
		}

		if !unlimited {
			// TODO: implement this on the data layer
			votedMovies, _, err := s.backend.GetUserVotes(user)
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

			maxVotes, err := s.backend.GetMaxUserVotes()

			if err != nil {
				s.doError(http.StatusBadRequest,
					fmt.Sprintf("Cannot get Max user votes %v", err),
					w, r)
				return
			}

			if count >= maxVotes {
				s.doError(http.StatusBadRequest,
					"You don't have any more available votes!",
					w, r)
				return
			}
		}

		if err := s.backend.AddVote(user.Id, movieId); err != nil {
			s.doError(http.StatusBadRequest, "Something went wrong :c", w, r)
			s.l.Error("Unable to cast vote: %v", err)
			return
		}
	}

	ref := r.Header.Get("Referer")
	if ref == "" {
		http.Redirect(w, r, "/", http.StatusFound)
	}
	http.Redirect(w, r, ref, http.StatusFound)
}
