package moviepoll

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/zorchenhimer/MoviePolls/common"
)

// Returns current active votes and votes for watched movies
func (s *Server) getUserVotes(user *common.User) ([]*common.Movie, []*common.Movie, error) {
	voted, err := s.data.GetUserVotes(user.Id)
	if err != nil {
		return nil, nil, fmt.Errorf("Unable to get all user votes for ID %d: %v", user.Id, err)
	}

	current := []*common.Movie{}
	watched := []*common.Movie{}

	for _, movie := range voted {
		if movie.Removed == true {
			continue
		}

		if movie.CycleWatched == nil {
			current = append(current, movie)
		} else {
			watched = append(watched, movie)
		}
	}

	return current, watched, nil
}

func (s *Server) AddAuthMethodToUser(auth *common.AuthMethod, user *common.User) (*common.User, error) {

	if user.AuthMethods == nil {
		user.AuthMethods = []*common.AuthMethod{}
	}

	// Check if the user already has this authtype associated with him
	if _, err := user.GetAuthMethod(auth.Type); err != nil {

		id, err := s.data.AddAuthMethod(auth)

		if err != nil {
			return nil, fmt.Errorf("Could not create new AuthMethod %s for user %s", auth.Type, user.Name)
		}

		auth.Id = id

		user.AuthMethods = append(user.AuthMethods, auth)

		return user, err
	} else {
		return nil, fmt.Errorf("AuthMethod %s is already associated with the user %s", auth.Type, user.Name)
	}
}

func (s *Server) RemoveAuthMethodFromUser(auth *common.AuthMethod, user *common.User) (*common.User, error) {

	if user.AuthMethods == nil {
		user.AuthMethods = []*common.AuthMethod{}
	}

	// Check if the user already has this authtype associated with him
	_, err := user.GetAuthMethod(auth.Type)
	if err != nil {
		return nil, fmt.Errorf("AuthMethod %s is not associated with the user %s", auth.Type, user.Name)
	}
	s.data.DeleteAuthMethod(auth.Id)

	// thanks golang for not having a delete method for slices ...
	oldauths := user.AuthMethods
	newAuths := []*common.AuthMethod{}
	for _, a := range oldauths {
		if a != auth {
			newAuths = append(newAuths, a)
		}
	}

	user.AuthMethods = newAuths

	return user, err
}
