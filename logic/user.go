package logic

import (
	"fmt"

	"github.com/zorchenhimer/MoviePolls/models"
)

func (b *backend) GetUserVotes(user *models.User) ([]*models.Movie, []*models.Movie, error) {
	voted, err := b.data.GetUserVotes(user.Id)
	if err != nil {
		return nil, nil, fmt.Errorf("Unable to get all user votes for ID %d: %v", user.Id, err)
	}

	current := []*models.Movie{}
	watched := []*models.Movie{}

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

func (b *backend) AddAuthMethodToUser(auth *models.AuthMethod, user *models.User) (*models.User, error) {

	if user.AuthMethods == nil {
		user.AuthMethods = []*models.AuthMethod{}
	}

	// Check if the user already has this authtype associated with him
	if _, err := user.GetAuthMethod(auth.Type); err != nil {

		id, err := b.data.AddAuthMethod(auth)

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

func (b *backend) RemoveAuthMethodFromUser(auth *models.AuthMethod, user *models.User) (*models.User, error) {

	if user.AuthMethods == nil {
		user.AuthMethods = []*models.AuthMethod{}
	}

	// Check if the user already has this authtype associated with him
	_, err := user.GetAuthMethod(auth.Type)
	if err != nil {
		return nil, fmt.Errorf("AuthMethod %s is not associated with the user %s", auth.Type, user.Name)
	}
	b.data.DeleteAuthMethod(auth.Id)

	// thanks golang for not having a delete method for slices ...
	oldauths := user.AuthMethods
	newAuths := []*models.AuthMethod{}
	for _, a := range oldauths {
		if a != auth {
			newAuths = append(newAuths, a)
		}
	}

	user.AuthMethods = newAuths

	return user, err
}
