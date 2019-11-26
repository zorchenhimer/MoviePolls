package moviepoll

import (
	"fmt"
	"time"
	"strconv"
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

type configMap map[string]configValue

type cfgValType int
const (
	CVT_STRING cfgValType = iota
	CVT_INT
	CVT_BOOL
)

type configValue struct {
	Type cfgValType
	Value interface{}
}

func (v configValue) String() string {
	t := ""
	switch v.Type {
	case CVT_STRING:
		t = "string"
		break;
	case CVT_INT:
		t = "int"
		break;
	case CVT_BOOL:
		t = "bool"
		break;
	}

	return fmt.Sprintf("configValue{Type:%s Value:%v}", t, v.Value)
}

func (c configMap) GetString(key string) (string, error) {
	val, ok := c[key]
	if !ok {
		return "", fmt.Errorf("Setting with key %q does not exist", key)
	}

	switch val.Type {
	case CVT_STRING:
		return val.Value.(string), nil
	case CVT_INT:
		return fmt.Sprintf("%d", val.Value.(int)), nil
	case CVT_BOOL:
		return fmt.Sprintf("%t", val.Value.(bool)), nil
	default:
		return "", fmt.Errorf("Unknown type %d", val.Type)
	}
}

func (c configMap) GetInt(key string) (int, error) {
	val, ok := c[key]
	if !ok {
		return 0, fmt.Errorf("Setting with key %q does not exist", key)
	}

	switch val.Type {
	case CVT_STRING:
		ival, err := strconv.ParseInt(val.Value.(string), 10, 32)
		if err != nil {
			return 0, fmt.Errorf("Int parse error: %s", err)
		}

		return int(ival), nil
	case CVT_INT:
		return val.Value.(int), nil
	case CVT_BOOL:
		if val.Value.(bool) == true {
			return 1, nil
		}
		return 0, nil
	default:
		return 0, fmt.Errorf("Unknown type %d", val.Type)
	}
}

func (c configMap) GetBool(key string) (bool, error) {
	val, ok := c[key]
	if !ok {
		return false, fmt.Errorf("Setting with key %q does not exist", key)
	}

	switch val.Type {
	case CVT_STRING:
		bval, err := strconv.ParseBool(val.Value.(string))
		if err != nil {
			return false, fmt.Errorf("Bool parse error: %s", err)
		}
		return bval, nil
	case CVT_INT:
		if val.Value.(int) == 0 {
			return false, nil
		}
		return true, nil
	case CVT_BOOL:
		return val.Value.(bool), nil
	default:
		return false, fmt.Errorf("Unknown type %d", val.Type)
	}
}

func (c configMap) SetString(key, value string) {
	c[key] = configValue{CVT_STRING, value}
}

func (c configMap) SetInt(key string, value int) {
	c[key] = configValue{CVT_INT, value}
}

func (c configMap) SetBool(key string, value bool) {
	c[key] = configValue{CVT_BOOL, value}
}

