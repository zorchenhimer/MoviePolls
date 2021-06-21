package logic

import (
	"fmt"
	"mime/multipart"
	"strings"

	"github.com/zorchenhimer/MoviePolls/database"
	"github.com/zorchenhimer/MoviePolls/models"
)

type Logic interface {
	// security
	GetKeys() (string, string, string, error)
	GetCryptRandKey(size int) string
	HashPassword(password string) string

	// Movie stuff
	AddMovie(fields map[string]*InputField, user *models.User) (int, map[string]*InputField)
	GetMovie(id int) *models.Movie
	GetActiveMovies() ([]*models.Movie, error)
	SearchMovieTitles(query string) ([]*models.Movie, error)

	// User stuff
	AddUser(user *models.User) (int, error)
	UpdateUser(user *models.User) error
	GetUserVotes(user *models.User) ([]*models.Movie, []*models.Movie, error)
	GetUserMovies(userId int) ([]*models.Movie, error)
	AddAuthMethodToUser(auth *models.AuthMethod, user *models.User) (*models.User, error)
	UpdateAuthMethod(auth *models.AuthMethod) error
	RemoveAuthMethodFromUser(auth *models.AuthMethod, user *models.User) (*models.User, error)
	UserTwitchLogin(extId string) (*models.User, error)
	UserDiscordLogin(extId string) (*models.User, error)
	UserPatreonLogin(extId string) (*models.User, error)

	// Settings
	GetFormFillEnabled() (bool, error)

	GetAvailableVotes(user *models.User) (int, error)
	GetMaxUserVotes() int
	GetUnlimitedVotes() bool
	GetVotingEnabled() bool

	GetCurrentCycle() (*models.Cycle, error)
	GetMaxRemarksLength() (int, error)
	GetPastCycles(start, count int) ([]*models.Cycle, error)
	GetPreviousCycle() *models.Cycle

	CheckOauthUsage(id string, authtype models.AuthType) bool
	GetTwitchOauthEnabled() (bool, error)
	GetDiscordOauthEnabled() (bool, error)
	GetPatreonOauthEnabled() (bool, error)
	GetHostAddress() (string, error)
	GetTwitchOauthClientID() (string, error)
	GetTwitchOauthClientSecret() (string, error)
	GetDiscordOauthClientID() (string, error)
	GetDiscordOauthClientSecret() (string, error)
	GetPatreonOauthClientID() (string, error)
	GetPatreonOauthClientSecret() (string, error)
}

type InputField struct {
	Value string
	Error error
}

type backend struct {
	data         database.Database
	urlKeys      map[string]*models.UrlKey
	authKey      string
	encryptKey   string
	passwordSalt string
	l            *models.Logger
}

func New(db database.Database, log *models.Logger) (Logic, error) {
	back := &backend{
		data:    db,
		urlKeys: make(map[string]*models.UrlKey),
		l:       log,
	}

	// check admin exists
	found := false
	start := 0
	count := 20

	for !found {
		users, err := db.GetUsers(start, 20)
		if err != nil {
			return nil, err
		}
		start += count

		if err != nil {
			return nil, fmt.Errorf("Error looking for admin: %v", err)
		}

		if len(users) == 0 {
			break
		}

		for _, u := range users {
			if u.IsAdmin() {
				found = true
				break
			}
		}
	}

	if !found {
		urlKey, err := models.NewAdminAuth()
		if err != nil {
			return nil, fmt.Errorf("Unable to get Url/Key pair for admin auth: %v", err)
		}

		back.urlKeys[urlKey.Url] = urlKey

		host, err := db.GetCfgString(ConfigHostAddress, "")
		if err != nil {
			return nil, fmt.Errorf("Unable to get host: %v", err)
		}

		if host == "" {
			host = "http://<host>"
		}
		host = strings.ToLower(host)

		if !strings.HasPrefix(host, "http") {
			host = "http://" + host
		}

		// Print directly to the console, not through the logger.
		fmt.Printf("Claim admin: %s/auth/%s Password: %s\n", host, urlKey.Url, urlKey.Key)
	}
	authKey, encryptKey, passwordSalt, err := back.GetKeys()
	if err != nil {
		return nil, err
	}
	back.authKey = authKey
	back.encryptKey = encryptKey
	back.passwordSalt = passwordSalt

	return back, nil
}

type inputForm struct {
	multipart.Form
}

func (f *inputForm) GetValue(key string) (string, error) {
	val, ok := f.Value[key]
	if !ok {
		return "", fmt.Errorf("[inputForm.GetValue] Key not found")
	}

	if len(val) == 0 {
		return "", fmt.Errorf("[inputForm.GetValue] Empty value")
	}

	return val[0], nil
}
