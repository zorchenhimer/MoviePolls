package web

import (
	"fmt"
	"net/http"
)

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
				if movie.CycleWatched == nil && !movie.Removed {
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
