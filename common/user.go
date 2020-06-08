package common

import (
	"fmt"
	"time"
)

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

func (u User) IsAdmin() bool {
	return u.Privilege >= PRIV_ADMIN
}

func (u User) IsMod() bool {
	return u.Privilege >= PRIV_ADMIN
}

func (u User) String() string {
	return fmt.Sprintf(
		"User{Id:%d Name:%q Email:%q NotifyCycleEnd:%t NotifyVoteSelection:%t Privilege:%d PassDate:%s}",
		u.Id,
		u.Name,
		u.Email,
		u.NotifyCycleEnd,
		u.NotifyVoteSelection,
		u.Privilege,
		u.PassDate,
	)
}
