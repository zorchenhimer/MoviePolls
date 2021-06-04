package data

import (
	"fmt"
	"time"

	mpm "github.com/zorchenhimer/MoviePolls/models"
)

type constructor func(string, *mpm.Logger) (DataConnector, error)

var registeredConnectors map[string]constructor

func GetDataConnector(backend, connectionString string, l *mpm.Logger) (DataConnector, error) {
	dc, ok := registeredConnectors[backend]
	if !ok {
		return nil, fmt.Errorf("Backend %s is not available", backend)
	}

	return dc(connectionString, l)
}

func register(backend string, initFunc constructor) {
	if registeredConnectors == nil {
		registeredConnectors = map[string]constructor{}
	}

	registeredConnectors[backend] = initFunc
}

type DataConnector interface {

	// ##################
	// ##### CREATE #####
	// ##################

	// TODO: remove AddCycle()
	AddCycle(plannedEnd *time.Time) (int, error)
	AddOldCycle(cycle *mpm.Cycle) (int, error)
	AddMovie(movie *mpm.Movie) (int, error)
	AddUser(user *mpm.User) (int, error)
	AddTag(tag *mpm.Tag) (int, error)
	AddAuthMethod(authMethod *mpm.AuthMethod) (int, error)
	AddLink(link *mpm.Link) (int, error)
	AddVote(userId, movieId int) error

	// ######################
	// ##### READ (get) #####
	// ######################

	GetCycle(id int) (*mpm.Cycle, error)
	GetCurrentCycle() (*mpm.Cycle, error) // Return nil when no cycle is active.
	GetMovie(id int) (*mpm.Movie, error)
	GetActiveMovies() ([]*mpm.Movie, error)
	GetUser(id int) (*mpm.User, error)
	GetUsers(start, count int) ([]*mpm.User, error)
	GetUserVotes(userId int) ([]*mpm.Movie, error)
	GetUserMovies(userId int) ([]*mpm.Movie, error)
	GetUsersWithAuth(auth mpm.AuthType, exclusive bool) ([]*mpm.User, error)
	//GetMovieVotes(userId int) []*Movie
	GetTag(id int) *mpm.Tag
	GetAuthMethod(id int) *mpm.AuthMethod
	GetLink(id int) *mpm.Link
	// Return a list of past cycles.  Start and end are an offset from
	// the current.  Ie, a start of 0 and an end of 5 will get the last
	// finished cycle and the four preceding it.  Currently active cycle will
	// not be returned.
	GetPastCycles(start, count int) ([]*mpm.Cycle, error)

	// Get all the movies that belong to the given Cycle
	GetMoviesFromCycle(id int) ([]*mpm.Movie, error)

	// #######################
	// ##### READ (find) #####
	// #######################

	FindTag(name string) (int, error)
	FindLink(url string) (int, error)

	// ##################
	// ##### UPDATE #####
	// ##################

	UpdateUser(user *mpm.User) error
	UpdateMovie(movie *mpm.Movie) error
	UpdateCycle(cycle *mpm.Cycle) error
	UpdateAuthMethod(authMethod *mpm.AuthMethod) error

	// ##################
	// ##### DELETE #####
	// ##################

	DeleteVote(userId, movieId int) error
	DeleteTag(tagId int)
	DeleteAuthMethod(authMethodId int)
	DeleteLink(linkId int)
	RemoveMovie(movieId int) error
	// Delete a user and their associated votes.  Should this include votes for
	// past cycles or just the current? (currently removes all)
	PurgeUser(userId int) error
	// Removes votes older than age
	DecayVotes(age int) error

	// ################
	// ##### MISC #####
	// ################

	UserLocalLogin(name, hashedPw string) (*mpm.User, error)
	UserDiscordLogin(extid string) (*mpm.User, error)
	UserTwitchLogin(extid string) (*mpm.User, error)
	UserPatreonLogin(extid string) (*mpm.User, error)

	CheckOauthUsage(id string, authtype mpm.AuthType) bool

	SearchMovieTitles(query string) ([]*mpm.Movie, error)

	CheckMovieExists(title string) (bool, error)
	CheckUserExists(name string) (bool, error)

	UserVotedForMovie(userId, movieId int) (bool, error)

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
	DeleteMovie(movieId int) error
	DeleteCycle(cycleId int) error

	Test_GetUserVotes(userId int) ([]*mpm.Vote, error)
}
