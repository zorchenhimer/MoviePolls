package logic

import (
	"crypto/rand"
	"crypto/sha512"
	"fmt"
	"math/big"

	"github.com/zorchenhimer/MoviePolls/models"
)

func (b *backend) GetCryptRandKey(size int) string {
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

// AuthKey, EncryptKey, Salt
func (b *backend) GetKeys() (string, string, string, error) {
	authKey, err := b.data.GetCfgString("SessionAuth", "")
	if err != nil || authKey == "" {
		authKey = b.GetCryptRandKey(64)
		err = b.data.SetCfgString("SessionAuth", authKey)
		if err != nil {
			return "", "", "", fmt.Errorf("Unable to set SessionAuth: %v", err)
		}
	}

	encryptKey, err := b.data.GetCfgString("SessionEncrypt", "")
	if err != nil || encryptKey == "" {
		encryptKey = b.GetCryptRandKey(32)
		err = b.data.SetCfgString("SessionEncrypt", encryptKey)
		if err != nil {
			return "", "", "", fmt.Errorf("Unable to set SessionEncrypt: %v", err)
		}
	}

	passwordSalt, err := b.data.GetCfgString("PassSalt", "")
	if err != nil || passwordSalt == "" {
		passwordSalt = b.GetCryptRandKey(32)
		err = b.data.SetCfgString("PassSalt", passwordSalt)
		if err != nil {
			return "", "", "", fmt.Errorf("Unable to set PassSalt: %v", err)
		}
	}

	return authKey, encryptKey, passwordSalt, nil
}

func (b *backend) HashPassword(pass string) string {
	return fmt.Sprintf("%x", sha512.Sum512([]byte(b.passwordSalt+pass)))
}

func NewAdminAuth() (*models.UrlKey, error) {
	url, err := generatePass()
	if err != nil {
		return nil, fmt.Errorf("Error generating UrlKey token URL: %v", err)
	}

	key, err := generatePass()
	if err != nil {
		return nil, fmt.Errorf("Error generating UrlKey token key: %v", err)
	}

	return &models.UrlKey{
		Url:  url,
		Key:  key,
		Type: models.UKT_AdminAuth,
	}, nil
}

func (b *backend) NewPasswordResetKey(userId int) (*models.UrlKey, error) {
	url, err := generatePass()
	if err != nil {
		return nil, fmt.Errorf("Error generating UrlKey token URL: %v", err)
	}

	key, err := generatePass()
	if err != nil {
		return nil, fmt.Errorf("Error generating UrlKey token key: %v", err)
	}

	return &models.UrlKey{
		Url:    url,
		Key:    key,
		Type:   models.UKT_PasswordReset,
		UserId: userId,
	}, nil
}

// TODO: do something better with this
func generatePass() (string, error) {
	out := ""
	for len(out) < 20 {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(15)))
		if err != nil {
			return "", err
		}

		out = fmt.Sprintf("%s%X", out, num)
	}
	return out, nil
}
