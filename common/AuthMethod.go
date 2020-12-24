package common

import "time"

type AuthType string

const (
	DISCORD = "Discord"
	TWITCH  = "Twitch"
	PATREON = "Patreon"
	LOCAL   = "Local"
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
