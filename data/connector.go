package data

import (
	"fmt"
	//"time"

	"github.com/zorchenhimer/MoviePolls/common"
)

type constructor func(string) (DataConnector, error)

var registeredConnectors map[string]constructor

func GetDataConnector(backend, connectionString string) (DataConnector, error) {
	dc, ok := registeredConnectors[backend]
	if !ok {
		return nil, fmt.Errorf("Backend %s is not available", backend)
	}

	return dc(connectionString)
}

func register(backend string, initFunc constructor) {
	if registeredConnectors == nil {
		registeredConnectors = map[string]constructor{}
	}

	registeredConnectors[backend] = initFunc
}

type DataConnector interface {
	GetCurrentCycle() (*common.Cycle, error) // Return nil when no cycle is active.
	GetMovie(id int) (*common.Movie, error)
	GetUser(id int) (*common.User, error)
	GetActiveMovies() ([]*common.Movie, error)

	GetUserVotes(userId int) ([]*common.Movie, error)

	//GetMovieVotes(userId int) []*Movie
	UserLogin(name, hashedPw string) (*common.User, error)

	// Return a list of past cycles.  Start and end are an offset from
	// the current.  Ie, a start of 1 and an end of 5 will get the last
	// finished cycle and the four preceding it.  The currently active cycle
	// would be at a start value of 0.
	GetPastCycles(start, end int) ([]*common.Cycle, error)

	// TODO: remove AddCycle()
	//AddCycle(end *time.Time) (int, error)
	AddOldCycle(cycle *common.Cycle) (int, error)
	AddMovie(movie *common.Movie) (int, error)
	AddUser(user *common.User) (int, error)

	AddVote(userId, movieId int) error
	DeleteVote(userId, movieId int) error

	UpdateUser(user *common.User) error
	UpdateMovie(movie *common.Movie) error
	UpdateCycle(cycle *common.Cycle) error

	CheckMovieExists(title string) (bool, error)
	CheckUserExists(name string) (bool, error)

	UserVotedForMovie(userId, movieId int) (bool, error)

	// Admin stuff
	GetUsers(start, count int) ([]*common.User, error)

	// Configuration stuff
	// The default value must be passed in.  If no key is found, the default
	// value *is not* saved here.
	GetCfgString(key, value string) (string, error)
	GetCfgInt(key string, value int) (int, error)
	GetCfgBool(key string, value bool) (bool, error)

	SetCfgString(key, value string) error
	SetCfgInt(key string, value int) error
	SetCfgBool(key string, value bool) error

	DeleteCfgKey(key string) error
}

type TestableDataConnector interface {
	DataConnector

	DeleteUser(userId int) error
	DeleteMovie(userId int) error
	DeleteCycle(userId int) error
}
