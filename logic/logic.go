package logic

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"mime/multipart"
	"strings"

	"github.com/zorchenhimer/MoviePolls/database"
	"github.com/zorchenhimer/MoviePolls/models"
)

type Logic interface {
	GetActiveMovies() ([]*models.Movie, error)
	SearchMovieTitles(query string) ([]*models.Movie, error)
	GetMovie(id int) *models.Movie
	AddMovie(fields map[string]*InputField, user *models.User) (int, map[string]*InputField)

	AddAuthMethodToUser(auth *models.AuthMethod, user *models.User) (*models.User, error)
	GetUserVotes(user *models.User) ([]*models.Movie, []*models.Movie, error)
	RemoveAuthMethodFromUser(auth *models.AuthMethod, user *models.User) (*models.User, error)

	GetKeys() (string, string, string, error)
	GetFormFillEnabled() (bool, error)

	GetAvailableVotes(user *models.User) (int, error)
	GetMaxUserVotes() int
	GetUnlimitedVotes() bool
	GetVotingEnabled() bool

	GetCurrentCycle() (*models.Cycle, error)
	GetPastCycles(start, count int) ([]*models.Cycle, error)
	GetPreviousCycle() *models.Cycle
}

type InputField struct {
	Value string
	Error error
}

type backend struct {
	data    database.Database
	urlKeys map[string]*models.UrlKey
	l       *models.Logger
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

	return back, nil
}

// AuthKey, EncryptKey, Salt
func (b *backend) GetKeys() (string, string, string, error) {
	authKey, err := b.data.GetCfgString("SessionAuth", "")
	if err != nil || authKey == "" {
		authKey = getCryptRandKey(64)
		err = b.data.SetCfgString("SessionAuth", authKey)
		if err != nil {
			return "", "", "", fmt.Errorf("Unable to set SessionAuth: %v", err)
		}
	}

	encryptKey, err := b.data.GetCfgString("SessionEncrypt", "")
	if err != nil || encryptKey == "" {
		encryptKey = getCryptRandKey(32)
		err = b.data.SetCfgString("SessionEncrypt", encryptKey)
		if err != nil {
			return "", "", "", fmt.Errorf("Unable to set SessionEncrypt: %v", err)
		}
	}

	passwordSalt, err := b.data.GetCfgString("PassSalt", "")
	if err != nil || passwordSalt == "" {
		passwordSalt = getCryptRandKey(32)
		err = b.data.SetCfgString("PassSalt", passwordSalt)
		if err != nil {
			return "", "", "", fmt.Errorf("Unable to set PassSalt: %v", err)
		}
	}

	return authKey, encryptKey, passwordSalt, nil
}

func getCryptRandKey(size int) string {
	out := ""
	large := big.NewInt(int64(1 << 60))
	large = large.Add(large, large)
	for len(out) < size {
		num, err := rand.Int(rand.Reader, large)
		if err != nil {
			panic("Error generating session key: " + err.Error())
		}
		out = fmt.Sprintf("%s%X", out, num)
	}

	if len(out) > size {
		out = out[:size]
	}
	return out
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
