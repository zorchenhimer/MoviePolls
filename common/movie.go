package common

import (
	"fmt"
	"time"
)

type Movie struct {
	Id          int
	Name        string
	Links       []string
	Description string

	//CycleAddedId int
	CycleAdded *Cycle

	Removed  bool // Removed by a mod or admin
	Approved bool // Approved by a mod or admin (if required by config)
	Watched  *time.Time

	Votes []*Vote

	Poster  string // TODO: make this procedural
	AddedBy *User
}

func (m Movie) UserVoted(userId int) bool {
	for _, v := range m.Votes {
		if v.User.Id == userId {
			return true
		}
	}
	return false
}

func (m Movie) String() string {
	votes := []string{}
	for _, v := range m.Votes {
		votes = append(votes, v.User.Name)
	}

	return fmt.Sprintf("Movie{Id:%d Name:%q Links:%s Description:%q CycleAdded:%s Votes:%s}",
		m.Id,
		m.Name,
		m.Links,
		m.Description,
		m.CycleAdded,
		votes,
	)
}
