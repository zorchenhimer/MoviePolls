package logic

import (
	mpd "github.com/zorchenhimer/MoviePolls/data"
	mpm "github.com/zorchenhimer/MoviePolls/models"
)

type LogicData struct {
	Database     mpd.DataConnector
	Logger       *mpm.Logger
	PasswordSalt string
	Debug        bool
}
