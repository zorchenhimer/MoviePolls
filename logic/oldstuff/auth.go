package logic

import (
	"crypto/sha256"
	"fmt"
)

func (l *LogicData) logout() error {
	session, err := l.cookies.Get(r, SessionName)
	if err != nil {
		return fmt.Errorf("Unable to get session from store: %v", err)
	}

	return delSession(session, w, r)
}

func (l *LogicData) login(user *mpm.User, authType mpm.AuthType) error {
	session, err := l.cookies.Get(r, SessionName)
	if err != nil {
		return fmt.Errorf("Unable to get session from store: %v", err)
	}

	auth, err := user.GetAuthMethod(authType)
	if err != nil {
		return err
	}

	session.Values["UserId"] = user.Id

	gobbed, err := auth.Date.GobEncode()
	if err != nil {
		return fmt.Errorf("Unable to gob Date")
	}

	session.Values["Date_"+string(auth.Type)] = fmt.Sprintf("%X", sha256.Sum256([]byte(gobbed)))

	return session.Save(r, w)
}
