package models

import (
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
