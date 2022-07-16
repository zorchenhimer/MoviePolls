package web

import (
	"github.com/zorchenhimer/MoviePolls/models"
)

type dataPageBase struct {
	PageTitle string
	Notice    string

	User         *models.User
	CurrentCycle *models.Cycle
}

type dataMovieError struct {
	dataPageBase
	ErrorMessage string
}

type dataLoginForm struct {
	dataPageBase
	ErrorMessage string
	Authed       bool
	OAuth        bool
	TwitchOAuth  bool
	DiscordOAuth bool
	PatreonOAuth bool
}

type dataError struct {
	dataPageBase

	Message string
	Code    int
}

type dataAdminHome struct {
	dataPageBase

	Cycle *models.Cycle
}
