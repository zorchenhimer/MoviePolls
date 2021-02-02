package common

import "time"

type AuthType string

const (
	AUTH_DISCORD = "Discord"
	AUTH_TWITCH  = "Twitch"
	AUTH_PATREON = "Patreon"
	AUTH_LOCAL   = "Local"
)

type AuthMethod struct {
	Id           int
	ExtId        string
	Type         AuthType
	Password     string
	AuthToken    string
	RefreshToken string
	Date         time.Time
}
