package logic

import (
	"fmt"
	"mime/multipart"
	"strings"
	"time"

	"github.com/zorchenhimer/MoviePolls/database"
	"github.com/zorchenhimer/MoviePolls/models"
)

type Logic interface {
	// security
	GetKeys() (string, string, string, error)
	GetUrlKeys() map[string]*models.UrlKey
	SetUrlKey(key string, val *models.UrlKey)
	DeleteUrlKey(key string)
	GetCryptRandKey(size int) string
	HashPassword(password string) string

	// Movie stuff
	AddMovie(fields map[string]*InputField, user *models.User) (int, map[string]*InputField)
	GetMovie(id int) *models.Movie
	GetActiveMovies() ([]*models.Movie, error)
	SearchMovieTitles(query string) ([]*models.Movie, error)
	UpdateMovie(movie *models.Movie) error
	DeleteMovie(mid int) error

	// Link stuff
	AddLink(*models.Link) (int, error)

	// Cycle stuff
	AddCycle(*time.Time) (int, error)
	UpdateCycle(*models.Cycle) error
	EndCycle(cid int) error

	// User stuff
	AddUser(user *models.User) (int, error)
	UpdateUser(user *models.User) error
	GetUser(id int) (*models.User, error)
	GetUsers(low int, high int) ([]*models.User, error)
	GetUsersWithAuth(auth models.AuthType, exclusive bool) ([]*models.User, error)
	GetUserVotes(user *models.User) ([]*models.Movie, []*models.Movie, error)
	GetUserMovies(userId int) ([]*models.Movie, error)
	AddAuthMethodToUser(auth *models.AuthMethod, user *models.User) (*models.User, error)
	UpdateAuthMethod(auth *models.AuthMethod) error
	RemoveAuthMethodFromUser(auth *models.AuthMethod, user *models.User) (*models.User, error)
	UserTwitchLogin(extId string) (*models.User, error)
	UserDiscordLogin(extId string) (*models.User, error)
	UserPatreonLogin(extId string) (*models.User, error)
	UserLocalLogin(name string, passwd string) (*models.User, error)

	// Vote stuff
	AddVote(userid int, movieid int) error
	DeleteVote(userid int, movieid int) error
	UserVotedForMovie(userid int, movieid int) (bool, error)
	EnableVoting() error
	DisableVoting() error

	// Admin stuff
	CheckAdminRights(user *models.User) bool
	AdminDeleteUser(user *models.User) error
	AdminBanUser(user *models.User) error
	AdminPurgeUser(user *models.User) error

	// Settings
	GetConfigBanner() (string, error)

	GetFormFillEnabled() (bool, error)
	GetEntriesRequireApproval() (bool, error)

	GetAvailableVotes(user *models.User) (int, error)
	GetMaxUserVotes() (int, error)
	GetUnlimitedVotes() (bool, error)
	GetVotingEnabled() (bool, error)

	GetCurrentCycle() (*models.Cycle, error)
	GetMaxRemarksLength() (int, error)
	GetMinNameLength() (int, error)
	GetMaxNameLength() (int, error)
	GetPastCycles(start, count int) ([]*models.Cycle, error)
	GetPreviousCycle() *models.Cycle

	CheckOauthUsage(id string, authtype models.AuthType) bool
	GetTwitchOauthEnabled() (bool, error)
	GetTwitchOauthSignupEnabled() (bool, error)
	GetDiscordOauthEnabled() (bool, error)
	GetDiscordOauthSignupEnabled() (bool, error)
	GetPatreonOauthEnabled() (bool, error)
	GetPatreonOauthSignupEnabled() (bool, error)
	GetLocalSignupEnabled() (bool, error)
	GetHostAddress() (string, error)
	GetTwitchOauthClientID() (string, error)
	GetTwitchOauthClientSecret() (string, error)
	GetDiscordOauthClientID() (string, error)
	GetDiscordOauthClientSecret() (string, error)
	GetPatreonOauthClientID() (string, error)
	GetPatreonOauthClientSecret() (string, error)

	SetCfgInt(key string, value int) error
	SetCfgBool(key string, value bool) error
	SetCfgString(key string, value string) error
	GetCfgInt(key string, defVal int) (int, error)
	GetCfgBool(key string, defVal bool) (bool, error)
	GetCfgString(key string, defVal string) (string, error)
}

type InputField struct {
	Value string
	Error error
}

type backend struct {
	data         database.Database
	UrlKeys      map[string]*models.UrlKey
	authKey      string
	encryptKey   string
	passwordSalt string
	l            *models.Logger
}

func New(db database.Database, log *models.Logger) (Logic, error) {
	back := &backend{
		data:    db,
		UrlKeys: make(map[string]*models.UrlKey),
		l:       log,
	}

	back.setupConfig()
	err := back.LoadDefaultsIfNotSet()
	if err != nil {
		return nil, err
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

		back.UrlKeys[urlKey.Url] = urlKey

		host, err := back.GetHostAddress()
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

func (b *backend) GetUrlKeys() map[string]*models.UrlKey {
	return b.UrlKeys
}
func (b *backend) SetUrlKey(key string, val *models.UrlKey) {
	b.UrlKeys[key] = val
}
func (b *backend) DeleteUrlKey(key string) {
	delete(b.UrlKeys, key)
}
