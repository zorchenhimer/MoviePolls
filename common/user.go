package common

import (
	"fmt"
)

type PrivilegeLevel int

const (
	PRIV_USER PrivilegeLevel = iota
	PRIV_MOD
	PRIV_ADMIN
)

type User struct {
	Id    int
	Name  string
	Email string // nil if user didn't opt-in.

	NotifyCycleEnd      bool
	NotifyVoteSelection bool
	Privilege           PrivilegeLevel

	AuthMethods []*AuthMethod
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

func (u User) GetAuthMethod(method AuthType) (*AuthMethod, error) {
	for _, auth := range u.AuthMethods {
		if auth.Type == method {
			return auth, nil
		}
	}
	return nil, fmt.Errorf("No AuthMethod with type %s found for user %s.", method, u.Name)
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
