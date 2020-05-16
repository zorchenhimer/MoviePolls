package common

import (
	"fmt"
	//"time"
)

type Movie struct {
	Id          int
	Name        string
	Links       []string
	Description string

	CycleAdded   *Cycle
	CycleWatched *Cycle

	Removed  bool // Removed by a mod or admin
	Approved bool // Approved by a mod or admin (if required by config)

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

	return fmt.Sprintf("Movie{Id:%d Name:%q Links:%s Description:%q CycleAdded:%s CycleWatched:%s Votes:%s}",
		m.Id,
		m.Name,
		m.Links,
		m.Description,
		m.CycleAdded,
		m.CycleWatched,
		votes,
	)
}

type MovieList []*Movie

func (ml MovieList) Len() int           { return len(ml) }
func (ml MovieList) Less(i, j int) bool { return len(ml[i].Votes) < len(ml[j].Votes) }
func (ml MovieList) Swap(i, j int)      { ml[i], ml[j] = ml[j], ml[i] }
