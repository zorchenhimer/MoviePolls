package web

import (
	"crypto/sha256"
	"fmt"
	"net/http"

	"github.com/gorilla/sessions"
	"github.com/zorchenhimer/MoviePolls/models"
)

func (s *webServer) logout(w http.ResponseWriter, r *http.Request) error {
	session, err := s.cookies.Get(r, SessionName)
	if err != nil {
		return fmt.Errorf("Unable to get session from store: %v", err)
	}

	return delSession(session, w, r)
}

func (s *webServer) login(user *models.User, authType models.AuthType, w http.ResponseWriter, r *http.Request) error {
	session, err := s.cookies.Get(r, SessionName)
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

	session.Values["Date_"+string(auth.Type)] = fmt.Sprintf("%X", sha256.Sum256(gobbed))

	return session.Save(r, w)
}

func delSession(session *sessions.Session, w http.ResponseWriter, r *http.Request) error {
	delete(session.Values, "UserId")
	delete(session.Values, "Date_Local")
	delete(session.Values, "Date_Discord")
	delete(session.Values, "Date_Twitch")
	delete(session.Values, "Date_Patreon")

	return session.Save(r, w)
}

func (s *webServer) getSessionUser(w http.ResponseWriter, r *http.Request) *models.User {
	session, err := s.cookies.Get(r, SessionName)
	if err != nil {
		s.l.Error("Unable to get session from store: %v", err)
		err = delSession(session, w, r)
		if err != nil {
			s.l.Error("Unable to delete cookie: %v", err)
		}
		return nil
	}

	val := session.Values["UserId"]
	var userId int
	var ok bool

	if userId, ok = val.(int); !ok {
		err = delSession(session, w, r)
		if err != nil {
			s.l.Error("Unable to delete cookie: %v", err)
		}
		return nil
	}

	user, err := s.backend.GetUser(userId)
	if err != nil {
		s.l.Error("Unable to get user with ID %d: %v", userId, err)
		err = delSession(session, w, r)
		if err != nil {
			s.l.Error("Unable to delete cookie: %v", err)
		}
		return nil
	}

	// I am sorry - CptPie
	passDate, _ := session.Values["Date_Local"].(string)
	refreshTwitch, _ := session.Values["Date_Twitch"].(string)
	refreshDiscord, _ := session.Values["Date_Discord"].(string)
	refreshPatreon, _ := session.Values["Date_Patreon"].(string)

	if passDate != "" {
		localAuth, err := user.GetAuthMethod(models.AUTH_LOCAL)

		if err != nil {
			s.l.Error(err.Error())
			return nil
		}

		gobbed, err := localAuth.Date.GobEncode()

		if err != nil || fmt.Sprintf("%X", sha256.Sum256(gobbed)) != passDate {
			s.l.Info("User's Date_Local did not match stored value")
			err = delSession(session, w, r)
			if err != nil {
				s.l.Error("Unable to delete cookie: %v", err)
			}
			return nil
		}
	} else if refreshTwitch != "" {
		twitchAuth, err := user.GetAuthMethod(models.AUTH_TWITCH)

		if err != nil {
			s.l.Error(err.Error())
			return nil
		}

		gobbed, err := twitchAuth.Date.GobEncode()

		if err != nil || fmt.Sprintf("%X", sha256.Sum256(gobbed)) != refreshTwitch {
			s.l.Info("User's Date_Twitch did not match stored value")
			err = delSession(session, w, r)
			if err != nil {
				s.l.Error("Unable to delete cookie: %v", err)
			}
			return nil
		}
	} else if refreshDiscord != "" {
		discordAuth, err := user.GetAuthMethod(models.AUTH_DISCORD)

		if err != nil {
			s.l.Error(err.Error())
			return nil
		}

		gobbed, err := discordAuth.Date.GobEncode()

		if err != nil || fmt.Sprintf("%X", sha256.Sum256(gobbed)) != refreshDiscord {
			s.l.Info("User's Date_Discord did not match stored value")
			err = delSession(session, w, r)
			if err != nil {
				s.l.Error("Unable to delete cookie: %v", err)
			}
			return nil
		}
	} else if refreshPatreon != "" {
		patreonAuth, err := user.GetAuthMethod(models.AUTH_PATREON)

		if err != nil {
			s.l.Error(err.Error())
			return nil
		}

		gobbed, err := patreonAuth.Date.GobEncode()

		if err != nil || fmt.Sprintf("%X", sha256.Sum256(gobbed)) != refreshPatreon {
			s.l.Info("User's Date_Patreon did not match stored value")
			err = delSession(session, w, r)
			if err != nil {
				s.l.Error("Unable to delete cookie: %v", err)
			}
			return nil
		}
	} else {
		//WTF MAN
		s.l.Error("No valid login method detected")
		return nil
	}

	return user
}
