package moviepoll

import (
	"fmt"
	"time"
)

type Cycle struct {
	Id int

	Start     time.Time
	End       time.Time
	EndingSet bool // has an end time been set? (ignore End value if false)
}

func (c Cycle) String() string {
	return fmt.Sprintf("Cycle{Id:%d}", c.Id)
}

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

	Poster string // TODO: make this procedural
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

type Vote struct {
	User  *User
	Movie *Movie
	// Decay based on cycles active.
	CycleAdded *Cycle
}

type PrivilegeLevel int

const (
	PRIV_USER PrivilegeLevel = iota
	PRIV_MOD
	PRIV_ADMIN
)

type User struct {
	Id         int
	Name       string
	Password   string
	OAuthToken string
	Email      string // nil if user didn't opt-in.

	NotifyCycleEnd      bool
	NotifyVoteSelection bool
	Privilege           PrivilegeLevel

	PassDate time.Time

	// Does this user ignore rate limit? (default true for mod/admin)
	RateLimitOverride bool
	LastMovieAdd      time.Time
}

func (u User) CheckPriv(lvl string) bool {
	switch lvl {
	case "ADMIN":
		return u.Privilege >= PRIV_ADMIN
	case "MOD":
		return u.Privilege >= PRIV_MOD
	}

	return false
}

func (u User) String() string {
	return fmt.Sprintf(
		"User{Id:%d Name:%q Email:%q NotifyCycleEnd:%t NotifyVoteSelection:%t Privilege:%d}",
		u.Id,
		u.Name,
		u.Email,
		u.NotifyCycleEnd,
		u.NotifyVoteSelection,
		u.Privilege,
	)
}
