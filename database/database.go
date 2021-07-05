package database

import (
	"errors"
	"fmt"
	"time"

	"github.com/zorchenhimer/MoviePolls/models"
)

type constructor func(string, *models.Logger) (Database, error)

var registeredDatabases map[string]constructor
var ErrNoValue = errors.New("No value for key")

func GetDatabase(backend, connectionString string, l *models.Logger) (Database, error) {
	dc, ok := registeredDatabases[backend]
	if !ok {
		return nil, fmt.Errorf("Backend %s is not available", backend)
	}

	return dc(connectionString, l)
}

func register(backend string, initFunc constructor) {
	if registeredDatabases == nil {
		registeredDatabases = map[string]constructor{}
	}

	registeredDatabases[backend] = initFunc
}

type Database interface {

	// ##################
	// ##### CREATE #####
	// ##################

	// TODO: remove AddCycle()
	AddCycle(plannedEnd *time.Time) (int, error)
	AddOldCycle(cycle *models.Cycle) (int, error)
	AddMovie(movie *models.Movie) (int, error)
	AddUser(user *models.User) (int, error)
	AddTag(tag *models.Tag) (int, error)
	AddAuthMethod(authMethod *models.AuthMethod) (int, error)
	AddLink(link *models.Link) (int, error)
	AddVote(userId, movieId int) error

	// ######################
	// ##### READ (get) #####
	// ######################

	GetCycle(id int) (*models.Cycle, error)
	GetCurrentCycle() (*models.Cycle, error) // Return nil when no cycle is active.
	GetMovie(id int) (*models.Movie, error)
	GetActiveMovies() ([]*models.Movie, error)
	GetUser(id int) (*models.User, error)
	GetUsers(start, count int) ([]*models.User, error)
	GetUserVotes(userId int) ([]*models.Movie, error)
	GetUserMovies(userId int) ([]*models.Movie, error)
	GetUsersWithAuth(auth models.AuthType, exclusive bool) ([]*models.User, error)
	//GetMovieVotes(userId int) []*Movie
	GetTag(id int) *models.Tag
	GetAuthMethod(id int) *models.AuthMethod
	GetLink(id int) *models.Link
	// Return a list of past cycles.  Start and end are an offset from
	// the current.  Ie, a start of 0 and an end of 5 will get the last
	// finished cycle and the four preceding it.  Currently active cycle will
	// not be returned.
	GetPastCycles(start, count int) ([]*models.Cycle, error)

	// Get all the movies that belong to the given Cycle
	GetMoviesFromCycle(id int) ([]*models.Movie, error)

	// #######################
	// ##### READ (find) #####
	// #######################

	FindTag(name string) (int, error)
	FindLink(url string) (int, error)

	// ##################
	// ##### UPDATE #####
	// ##################

	UpdateUser(user *models.User) error
	UpdateMovie(movie *models.Movie) error
	UpdateCycle(cycle *models.Cycle) error
	UpdateAuthMethod(authMethod *models.AuthMethod) error

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

	UserLocalLogin(name, hashedPw string) (*models.User, error)
	UserDiscordLogin(extid string) (*models.User, error)
	UserTwitchLogin(extid string) (*models.User, error)
	UserPatreonLogin(extid string) (*models.User, error)

	CheckOauthUsage(id string, authtype models.AuthType) bool

	SearchMovieTitles(query string) ([]*models.Movie, error)

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

type TestableDatabase interface {
	Database

	DeleteUser(userId int) error
	DeleteMovie(movieId int) error
	DeleteCycle(cycleId int) error

	Test_GetUserVotes(userId int) ([]*models.Vote, error)
}
