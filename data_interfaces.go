package moviepoll

import (
	"time"
)

type DataConnector interface {
	GetCurrentCycle() *Cycle // should this always return a cycle?
	GetMovie(id int) (*Movie, error)
	GetUser(id int) (*User, error)

	// Return a list of past cycles.  Start and end are an offset from
	// the current.  Ie, a start of 1 and an end of 5 will get the last
	// finished cycle and the four preceding it.  The currently active cycle
	// would be at a start value of 0.
	GetPastCycles(start, end int) []*Cycle

	AddMovie(movie *Movie) error
	AddUser(user *User) error
	AddCycle(end *time.Time) error

	GetConfig() (*Configurator, error)
	SaveConfig(config *Configurator) error
}

type Configurator interface {
	GetString(key string) (string, error)
	GetInt(key string) (int, error)
	GetBool(key string) (bool, error)

	SetString(key, value string) error
	SetInt(key string, value int) error
	SetBool(key string, value bool) error
}
