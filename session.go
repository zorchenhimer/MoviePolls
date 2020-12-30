package moviepoll

import (
	"crypto/sha256"
	"fmt"
	"net/http"

	"github.com/gorilla/sessions"
	"github.com/zorchenhimer/MoviePolls/common"
)

func (s *Server) logout(w http.ResponseWriter, r *http.Request) error {
	session, err := s.cookies.Get(r, SessionName)
	if err != nil {
		return fmt.Errorf("Unable to get session from store: %v", err)
	}

	return delSession(session, w, r)
}

func (s *Server) login(user *common.User, authType common.AuthType, w http.ResponseWriter, r *http.Request) error {
	session, err := s.cookies.Get(r, SessionName)
	if err != nil {
		return fmt.Errorf("Unable to get session from store: %v", err)
	}

	auth, err := user.GetAuthMethod(authType)
	if err != nil {
		return err
	}

	session.Values["UserId"] = user.Id

	switch authType {
	case common.AUTH_LOCAL:

		gobbed, err := auth.PassDate.GobEncode()
		if err != nil {
			return fmt.Errorf("Unable to gob PassDate")
		}

		session.Values["PassDate"] = fmt.Sprintf("%X", sha256.Sum256([]byte(gobbed)))
	case common.AUTH_TWITCH:
		gobbed, err := auth.RefreshDate.GobEncode()
		if err != nil {
			return fmt.Errorf("Unable to gob RefreshDate")
		}

		session.Values["RefreshDate_Twitch"] = fmt.Sprintf("%X", sha256.Sum256([]byte(gobbed)))
	case common.AUTH_PATREON:
		gobbed, err := auth.RefreshDate.GobEncode()
		if err != nil {
			return fmt.Errorf("Unable to gob RefreshDate")
		}

		session.Values["RefreshDate_Patreon"] = fmt.Sprintf("%X", sha256.Sum256([]byte(gobbed)))
	case common.AUTH_DISCORD:
		gobbed, err := auth.RefreshDate.GobEncode()
		if err != nil {
			return fmt.Errorf("Unable to gob RefreshDate")
		}

		session.Values["RefreshDate_Discord"] = fmt.Sprintf("%X", sha256.Sum256([]byte(gobbed)))
	default:
		return fmt.Errorf("Login without a valid auth method")
	}

	return session.Save(r, w)
}

func delSession(session *sessions.Session, w http.ResponseWriter, r *http.Request) error {
	delete(session.Values, "UserId")
	delete(session.Values, "PassDate")
	delete(session.Values, "RefreshDate_Discord")
	delete(session.Values, "RefreshDate_Twitch")
	delete(session.Values, "RefreshDate_Patreon")

	return session.Save(r, w)
}

func (s *Server) getSessionUser(w http.ResponseWriter, r *http.Request) *common.User {
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

	user, err := s.data.GetUser(userId)
	if err != nil {
		s.l.Error("Unable to get user with ID %d: %v", userId, err)
		err = delSession(session, w, r)
		if err != nil {
			s.l.Error("Unable to delete cookie: %v", err)
		}
		return nil
	}

	// I am sorry - CptPie
	passDate, _ := session.Values["PassDate"].(string)
	refreshTwitch, _ := session.Values["RefreshDate_Twitch"].(string)
	refreshDiscord, _ := session.Values["RefreshDate_Discord"].(string)
	refreshPatreon, _ := session.Values["RefreshDate_Patreon"].(string)

	if passDate != "" {
		localAuth, err := user.GetAuthMethod(common.AUTH_LOCAL)

		if err != nil {
			s.l.Error(err.Error())
			return nil
		}

		gobbed, err := localAuth.PassDate.GobEncode()

		if err != nil || fmt.Sprintf("%X", sha256.Sum256([]byte(gobbed))) != passDate {
			s.l.Info("User's PassDate did not match stored value")
			err = delSession(session, w, r)
			if err != nil {
				s.l.Error("Unable to delete cookie: %v", err)
			}
			return nil
		}
	} else if refreshTwitch != "" {
		twitchAuth, err := user.GetAuthMethod(common.AUTH_TWITCH)

		if err != nil {
			s.l.Error(err.Error())
			return nil
		}

		gobbed, err := twitchAuth.RefreshDate.GobEncode()

		if err != nil || fmt.Sprintf("%X", sha256.Sum256([]byte(gobbed))) != refreshTwitch {
			s.l.Info("User's RefreshDate did not match stored value")
			err = delSession(session, w, r)
			if err != nil {
				s.l.Error("Unable to delete cookie: %v", err)
			}
			return nil
		}
	} else if refreshDiscord != "" {
		discordAuth, err := user.GetAuthMethod(common.AUTH_DISCORD)

		if err != nil {
			s.l.Error(err.Error())
			return nil
		}

		gobbed, err := discordAuth.RefreshDate.GobEncode()

		if err != nil || fmt.Sprintf("%X", sha256.Sum256([]byte(gobbed))) != refreshDiscord {
			s.l.Info("User's RefreshDate did not match stored value")
			err = delSession(session, w, r)
			if err != nil {
				s.l.Error("Unable to delete cookie: %v", err)
			}
			return nil
		}
	} else if refreshPatreon != "" {
		patreonAuth, err := user.GetAuthMethod(common.AUTH_PATREON)

		if err != nil {
			s.l.Error(err.Error())
			return nil
		}

		gobbed, err := patreonAuth.RefreshDate.GobEncode()

		if err != nil || fmt.Sprintf("%X", sha256.Sum256([]byte(gobbed))) != refreshPatreon {
			s.l.Info("User's RefreshDate did not match stored value")
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
