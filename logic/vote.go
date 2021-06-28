package logic

import "github.com/zorchenhimer/MoviePolls/models"

func (b *backend) AddVote(userid int, movieid int) error {
	return b.data.AddVote(userid, movieid)
}

func (b *backend) DeleteVote(userid int, movieid int) error {
	return b.data.DeleteVote(userid, movieid)
}

func (b *backend) UserVotedForMovie(userid int, movieid int) (bool, error) {
	return b.data.UserVotedForMovie(userid, movieid)
}

func (b *backend) GetMaxUserVotes() int {
	val, err := b.data.GetCfgInt(ConfigMaxUserVotes, DefaultMaxUserVotes)
	if err != nil {
		b.l.Error("Error getting MaxUserVotes config setting: %v", err)

		err = b.data.SetCfgInt(ConfigMaxUserVotes, DefaultMaxUserVotes)
		if err != nil {
			b.l.Error("Error setting default for MaxUserVotes setting: %v", err)
		}

		return DefaultMaxUserVotes
	}

	return val
}

func (b *backend) GetVotingEnabled() bool {
	val, err := b.data.GetCfgBool(ConfigVotingEnabled, DefaultVotingEnabled)
	if err != nil {
		b.l.Error("Error getting VotingEnabled config setting: %v", err)

		err = b.data.SetCfgBool(ConfigVotingEnabled, DefaultVotingEnabled)
		if err != nil {
			b.l.Error("Error setting default for VotingEnabled setting: %v", err)
		}

		return DefaultVotingEnabled
	}

	return val
}

func (b *backend) GetUnlimitedVotes() bool {
	val, err := b.data.GetCfgBool(ConfigUnlimitedVotes, DefaultUnlimitedVotes)
	if err != nil {
		b.l.Error("Error getting UnlimitedVotes config setting: %v", err)

		err = b.data.SetCfgBool(ConfigUnlimitedVotes, DefaultUnlimitedVotes)
		if err != nil {
			b.l.Error("Error setting default for UnlimitedVotes setting: %v", err)
		}

		return DefaultUnlimitedVotes
	}
	return val
}

func (b *backend) GetAvailableVotes(user *models.User) (int, error) {
	if b.GetUnlimitedVotes() {
		// Should this always return 1, or some higher value?
		return 1, nil
	}

	maxVotes := b.GetMaxUserVotes()

	active, _, err := b.GetUserVotes(user)
	if err != nil {
		return 0, err
	}
	return maxVotes - len(active), nil
}
