package common

import (
	"time"
)

type DataConnector interface {
	GetCurrentCycle() *Cycle // Return nil when no cycle is active.
	GetMovie(id int) (*Movie, error)
	GetUser(id int) (*User, error)
	GetActiveMovies() []*Movie

	GetUserVotes(userId int) []*Movie

	//GetMovieVotes(userId int) []*Movie
	UserLogin(name, hashedPw string) (*User, error)

	// Return a list of past cycles.  Start and end are an offset from
	// the current.  Ie, a start of 1 and an end of 5 will get the last
	// finished cycle and the four preceding it.  The currently active cycle
	// would be at a start value of 0.
	GetPastCycles(start, end int) []*Cycle

	AddCycle(end *time.Time) (int, error)
	AddOldCycle(cycle *Cycle) (int, error)
	AddMovie(movie *Movie) (int, error)
	AddUser(user *User) (int, error)
	AddVote(userId, movieId int) error

	UpdateUser(user *User) error
	UpdateMovie(movie *Movie) error
	UpdateCycle(cycle *Cycle) error

	CheckMovieExists(title string) bool
	CheckUserExists(name string) bool

	UserVotedForMovie(userId, movieId int) bool

	// Admin stuff
	GetUsers(start, count int) ([]*User, error)

	// Configuration stuff
	GetCfgString(key string) (string, error)
	GetCfgInt(key string) (int, error)
	GetCfgBool(key string) (bool, error)

	SetCfgString(key, value string) error
	SetCfgInt(key string, value int) error
	SetCfgBool(key string, value bool) error

	DeleteCfgKey(key string) error
}
