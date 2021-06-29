package logic

import (
	"errors"
	"fmt"

	"github.com/zorchenhimer/MoviePolls/database"
	"github.com/zorchenhimer/MoviePolls/models"
)

func (b *backend) AddVote(userid int, movieid int) error {
	return b.data.AddVote(userid, movieid)
}

func (b *backend) DeleteVote(userid int, movieid int) error {
	return b.data.DeleteVote(userid, movieid)
}

func (b *backend) UserVotedForMovie(userid int, movieid int) (bool, error) {
	return b.data.UserVotedForMovie(userid, movieid)
}

func (b *backend) GetMaxUserVotes() (int, error) {
	key := ConfigMaxUserVotes
	config, ok := ConfigValues[key]
	if !ok {
		return 0, fmt.Errorf("Could not find ConfigValue named %s", key)
	}
	val, err := b.data.GetCfgInt(key, config.Default.(int))
	if errors.Is(err, database.ErrNoValue) {
		err = b.data.SetCfgInt(key, config.Default.(int))
		if err != nil {
			b.l.Error("Unable to set default value for %s: %v", key, err)
		}
		return val, nil
	}

	return val, err
}

func (b *backend) GetVotingEnabled() (bool, error) {
	key := ConfigVotingEnabled
	config, ok := ConfigValues[key]
	if !ok {
		return false, fmt.Errorf("Could not find ConfigValue named %s", key)
	}
	val, err := b.data.GetCfgBool(key, config.Default.(bool))
	if errors.Is(err, database.ErrNoValue) {
		err = b.data.SetCfgBool(key, config.Default.(bool))
		if err != nil {
			b.l.Error("Unable to set default value for %s: %v", key, err)
		}
		return val, nil
	}

	return val, err
}

func (b *backend) GetUnlimitedVotes() (bool, error) {
	key := ConfigVotingEnabled
	config, ok := ConfigValues[key]
	if !ok {
		return false, fmt.Errorf("Could not find ConfigValue named %s", key)
	}
	val, err := b.data.GetCfgBool(key, config.Default.(bool))
	if errors.Is(err, database.ErrNoValue) {
		err = b.data.SetCfgBool(key, config.Default.(bool))
		if err != nil {
			b.l.Error("Unable to set default value for %s: %v", key, err)
		}
		return val, nil
	}

	return val, err
}

func (b *backend) GetAvailableVotes(user *models.User) (int, error) {
	unlimited, err := b.GetUnlimitedVotes()

	if err != nil {
		return 0, err
	}

	if unlimited {
		// Should this always return 1, or some higher value?
		return 1, nil
	}

	maxVotes, err := b.GetMaxUserVotes()

	if err != nil {
		return 0, err
	}

	active, _, err := b.GetUserVotes(user)
	if err != nil {
		return 0, err
	}
	return maxVotes - len(active), nil
}

func (b *backend) EnableVoting() error {
	return b.SetCfgBool(ConfigVotingEnabled, true)
}

func (b *backend) DisableVoting() error {
	return b.SetCfgBool(ConfigVotingEnabled, false)
}
