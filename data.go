package moviepoll

import (
	"fmt"
	"time"
)

type Cycle struct {
	Id int

	Start time.Time
	End   time.Time
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
	CycleAdded   *Cycle

	Removed  bool // Removed by a mod or admin
	Approved bool // Approved by a mod or admin (if required by config)
	Watched *time.Time

	Votes []*Vote
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
	User *User
	Movie *Movie
	// Decay based on cycles active.
	CycleAdded *Cycle
}

type Configuration struct {
	boolSettings   map[string]bool
	intSettings    map[string]int
	stringSettings map[string]string
}

func (c *Configuration) GetString(key string) (string, error) {
	if val, ok := c.stringSettings[key]; ok {
		return val, nil
	}
	return "", fmt.Errorf("Invalid string value key %q", key)
}

func (c *Configuration) GetInt(key string) (int, error) {
	if val, ok := c.intSettings[key]; ok {
		return val, nil
	}
	return 0, fmt.Errorf("Invalid int value key %q", key)
}

func (c *Configuration) GetBool(key string) (bool, error) {
	if val, ok := c.boolSettings[key]; ok {
		return val, nil
	}
	return false, fmt.Errorf("Invalid bool value key %q", key)
}

func (c *Configuration) SetString(key, value string) error {
	c.stringSettings[key] = value
	return nil
}

func (c *Configuration) SetInt(key string, value int) error {
	c.intSettings[key] = value
	return nil
}

func (c *Configuration) SetBool(key string, value bool) error {
	c.boolSettings[key] = value
	return nil
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
	OAuthToken string
	Email      string // nil if user didn't opt-in.

	NotifyCycleEnd      bool
	NotifyVoteSelection bool
	Privilege           PrivilegeLevel
}

type Choice struct {
	Id int
	//MovieID int
	Movie *Movie
	//CycleID int
	Cycle *Cycle
}
