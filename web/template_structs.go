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

type dataAddMovie struct {
	dataPageBase
	ErrorMessage []string

	// Offending input
	ErrTitle       bool
	ErrDescription bool
	ErrLinks       bool
	ErrRemarks     bool
	ErrPoster      bool
	ErrAutofill    bool

	// Values for input if error
	ValTitle       string
	ValDescription string
	ValLinks       string
	ValRemarks     string
	//ValPoster      bool

	AutofillEnabled bool
	FormfillEnabled bool

	MaxRemarksLength int
}

func (d dataAddMovie) isError() bool {
	return d.ErrTitle || d.ErrDescription || d.ErrLinks || d.ErrPoster || d.ErrAutofill || d.ErrRemarks
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
