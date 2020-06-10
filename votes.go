package moviepoll

import (
	"fmt"
	"net/http"
)

// Toggles votes
func (s *Server) handlerVote(w http.ResponseWriter, r *http.Request) {
	user := s.getSessionUser(w, r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	enabled, err := s.data.GetCfgBool("VotingEnabled", DefaultVotingEnabled)
	if err != nil {
		s.l.Error("Unable to get config value for VotingEnabled: %s\n", err)
	}

	// this should be false if an error was returned
	if !enabled {
		s.doError(
			http.StatusBadRequest,
			"Voting is not enabled",
			w, r)
		return
	}

	var movieId int
	if _, err := fmt.Sscanf(r.URL.Path, "/vote/%d", &movieId); err != nil {
		s.doError(http.StatusBadRequest, "Invalid movie ID", w, r)
		s.l.Info("invalid vote URL: %q\n", r.URL.Path)
		return
	}

	movie, err := s.data.GetMovie(movieId)
	if err != nil {
		s.doError(http.StatusBadRequest, "Invalid movie ID", w, r)
		s.l.Info("Movie with ID %d doesn't exist\n", movieId)
		return
	}
	if movie.CycleWatched != nil {
		s.doError(http.StatusBadRequest, "Movie already watched", w, r)
		s.l.Error("Attempted to vote on watched movie ID %d\n", movieId)
		return
	}

	userVoted, err := s.data.UserVotedForMovie(user.Id, movieId)
	if err != nil {
		s.doError(http.StatusBadRequest, "Something went wrong :c", w, r)
		s.l.Error("Cannot get user vote: %v", err)
		return
	}

	if userVoted {
		//s.doError(http.StatusBadRequest, "You already voted for that movie!", w, r)
		if err := s.data.DeleteVote(user.Id, movieId); err != nil {
			s.doError(http.StatusBadRequest, "Something went wrong :c", w, r)
			s.l.Error("Unable to remove vote: %v\n", err)
			return
		}
	} else {

		// TODO: implement this on the data layer
		votedMovies, err := s.data.GetUserVotes(user.Id)
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
			s.l.Error("Error getting MaxUserVotes config setting: %v\n", err)
			maxVotes = DefaultMaxUserVotes
		}

		if count >= maxVotes {
			s.doError(http.StatusBadRequest,
				"You don't have any more available votes!",
				w, r)
			return
		}

		if err := s.data.AddVote(user.Id, movieId); err != nil {
			s.doError(http.StatusBadRequest, "Something went wrong :c", w, r)
			s.l.Error("Unable to cast vote: %v\n", err)
			return
		}
	}

	ref := r.Header.Get("Referer")
	if ref == "" {
		http.Redirect(w, r, "/", http.StatusFound)
	}
	http.Redirect(w, r, ref, http.StatusFound)
}
