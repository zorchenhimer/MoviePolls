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
	Type         AuthType
	Password     string
	PassDate     time.Time
	AuthToken    string
	RefreshToken string
	RefreshDate  time.Time
}
