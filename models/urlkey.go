package models

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"time"
)

type UrlKeyType int

const (
	UKT_Unknown UrlKeyType = iota
	UKT_AdminAuth
	UKT_PasswordReset
)

type UrlKey struct {
	Url       string
	Key       string
	Type      UrlKeyType
	UserId    int // password resets
	Generated time.Time
}

func NewAdminAuth() (*UrlKey, error) {
	url, err := generatePass()
	if err != nil {
		return nil, fmt.Errorf("Error generating UrlKey token URL: %v", err)
	}

	key, err := generatePass()
	if err != nil {
		return nil, fmt.Errorf("Error generating UrlKey token key: %v", err)
	}

	return &UrlKey{
		Url:  url,
		Key:  key,
		Type: UKT_AdminAuth,
	}, nil
}

func NewPasswordResetKey(userId int) (*UrlKey, error) {
	url, err := generatePass()
	if err != nil {
		return nil, fmt.Errorf("Error generating UrlKey token URL: %v", err)
	}

	key, err := generatePass()
	if err != nil {
		return nil, fmt.Errorf("Error generating UrlKey token key: %v", err)
	}

	return &UrlKey{
		Url:    url,
		Key:    key,
		Type:   UKT_PasswordReset,
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
