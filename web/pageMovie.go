package web

import (
	"fmt"
	"net/http"

	"github.com/zorchenhimer/MoviePolls/models"
)

func (s *webServer) handlerPageMovie(w http.ResponseWriter, r *http.Request) {
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

	movie := s.backend.GetMovie(movieId)
	if movie == nil {
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

	data := struct {
		dataPageBase
		Movie          *models.Movie
		VotingEnabled  bool
		AvailableVotes int
	}{
		dataPageBase: s.newPageBase(movie.Name, w, r),
		Movie:        movie,
	}

	enabled, err := s.backend.GetVotingEnabled()
	if err != nil {
		s.doError(
			http.StatusBadRequest,
			fmt.Sprintf("Cannot get VotingEnabled"),
			w, r)
		s.l.Error("Unable to get VotingEnabled: %v", err)
	}
	data.VotingEnabled = enabled
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

	if err := s.executeTemplate(w, "movieinfo", data); err != nil {
		http.Error(w, fmt.Sprintf("%v", err), http.StatusInternalServerError)
	}
}
