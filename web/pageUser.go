package web

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/zorchenhimer/MoviePolls/models"
)

// /user/

func (s *webServer) handlerPageUser(w http.ResponseWriter, r *http.Request) {
	user := s.getSessionUser(w, r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	totalVotes, err := s.backend.GetMaxUserVotes()
	if err != nil {
		s.l.Error("Unable to get max votes: %v", err)
	}

	activeVotes, watchedVotes, err := s.backend.GetUserVotes(user)
	if err != nil {
		s.l.Error("Unable to get votes for user %d: %v", user.Id, err)
	}

	addedMovies, err := s.backend.GetUserMovies(user.Id)
	if err != nil {
		s.l.Error("Unable to get movies added by user %d: %v", user.Id, err)
	}

	unlimited, err := s.backend.GetUnlimitedVotes()

	if err != nil {
		s.l.Error("Unable to get UnlimitedVotes: %v", err)
	}

	data := struct {
		dataPageBase

		User *models.User

		TotalVotes     int
		AvailableVotes int
		UnlimitedVotes bool

		OAuthEnabled        bool
		TwitchOAuthEnabled  bool
		DiscordOAuthEnabled bool
		PatreonOAuthEnabled bool

		HasLocal   bool
		HasTwitch  bool
		HasDiscord bool
		HasPatreon bool

		CallbackError string

		ActiveVotes    []*models.Movie
		WatchedVotes   []*models.Movie
		AddedMovies    []*models.Movie
		SuccessMessage string

		PassError   []string
		NotifyError []string
		EmailError  []string

		ErrCurrentPass bool
		ErrNewPass     bool
		ErrEmail       bool
	}{
		dataPageBase: s.newPageBase("Account", w, r),

		User: user,

		TotalVotes:     totalVotes,
		AvailableVotes: totalVotes - len(activeVotes),
		UnlimitedVotes: unlimited,

		ActiveVotes:  activeVotes,
		WatchedVotes: watchedVotes,
		AddedMovies:  addedMovies,
	}

	if s.callbackError.message != "" {
		if user.Id == s.callbackError.user {
			data.CallbackError = s.callbackError.message

			s.callbackError.user = 0
			s.callbackError.message = ""
		}
	}

	twitchAuth, err := s.backend.GetTwitchOauthEnabled()
	if err != nil {
		s.doError(http.StatusInternalServerError, "Something went wrong :C", w, r)
		s.l.Error("Unable to get ConfigTwitchOauthEnabled config value: %v", err)
		return
	}
	data.TwitchOAuthEnabled = twitchAuth

	discordAuth, err := s.backend.GetDiscordOauthEnabled()
	if err != nil {
		s.doError(http.StatusInternalServerError, "Something went wrong :C", w, r)
		s.l.Error("Unable to get ConfigDiscordOauthEnabled config value: %v", err)
		return
	}
	data.DiscordOAuthEnabled = discordAuth

	patreonAuth, err := s.backend.GetPatreonOauthEnabled()
	if err != nil {
		s.doError(http.StatusInternalServerError, "Something went wrong :C", w, r)
		s.l.Error("Unable to get ConfigPatreonOauthEnabled config value: %v", err)
		return
	}
	data.PatreonOAuthEnabled = patreonAuth

	data.OAuthEnabled = twitchAuth || discordAuth || patreonAuth

	_, err = user.GetAuthMethod(models.AUTH_LOCAL)
	data.HasLocal = err == nil
	_, err = user.GetAuthMethod(models.AUTH_TWITCH)
	data.HasTwitch = err == nil
	_, err = user.GetAuthMethod(models.AUTH_DISCORD)
	data.HasDiscord = err == nil
	_, err = user.GetAuthMethod(models.AUTH_PATREON)
	data.HasPatreon = err == nil

	if r.Method == "POST" {
		err := r.ParseForm()
		if err != nil {
			s.l.Error("ParseForm() error: %v", err)
			s.doError(http.StatusInternalServerError, "Form error", w, r)
			return
		}

		formVal := r.PostFormValue("Form")
		if formVal == "ChangePassword" {
			// Do password stuff
			currentPass := s.backend.HashPassword(r.PostFormValue("PasswordCurrent"))
			newPass1_raw := r.PostFormValue("PasswordNew1")
			newPass2_raw := r.PostFormValue("PasswordNew2")

			localAuth, err := user.GetAuthMethod(models.AUTH_LOCAL)
			if err != nil {
				data.ErrCurrentPass = true
				data.PassError = append(data.PassError, "No Password detected.")
			} else {

				if currentPass != localAuth.Password {
					data.ErrCurrentPass = true
					data.PassError = append(data.PassError, "Invalid current password")
				}

				if newPass1_raw == "" {
					data.ErrNewPass = true
					data.PassError = append(data.PassError, "New password cannot be blank")
				}

				if newPass1_raw != newPass2_raw {
					data.ErrNewPass = true
					data.PassError = append(data.PassError, "Passwords do not match")
				}
				if !(data.ErrCurrentPass || data.ErrNewPass || data.ErrEmail) {
					// Change pass
					data.SuccessMessage = "Password successfully changed"
					localAuth.Password = s.backend.HashPassword(newPass1_raw)
					localAuth.Date = time.Now()

					if err = s.backend.UpdateAuthMethod(localAuth); err != nil {
						s.l.Error("Unable to save User with new password:", err)
						s.doError(http.StatusInternalServerError, "Unable to update password", w, r)
						return
					}

					s.l.Info("new Date_Local: %s", localAuth.Date)
					err = s.login(user, models.AUTH_LOCAL, w, r)
					if err != nil {
						s.l.Error("Unable to login to session:", err)
						s.doError(http.StatusInternalServerError, "Unable to update password", w, r)
						return
					}
				}
			}
		} else if formVal == "Notifications" {
			// Update notifications
		} else if formVal == "SetPassword" {
			pass1_raw := r.PostFormValue("Password1")
			pass2_raw := r.PostFormValue("Password2")

			_, err := user.GetAuthMethod(models.AUTH_LOCAL)
			if err == nil {
				data.ErrCurrentPass = true
				data.PassError = append(data.PassError, "Existing password detected. (how did you end up here anyways?)")
			} else {
				localAuth := &models.AuthMethod{
					Type: models.AUTH_LOCAL,
				}

				if pass1_raw == "" {
					data.ErrNewPass = true
					data.PassError = append(data.PassError, "New password cannot be blank")
				}

				if pass1_raw != pass2_raw {
					data.ErrNewPass = true
					data.PassError = append(data.PassError, "Passwords do not match")
				}
				if !(data.ErrCurrentPass || data.ErrNewPass || data.ErrEmail) {
					// Change pass
					data.SuccessMessage = "Password successfully set"
					localAuth.Password = s.backend.HashPassword(pass1_raw)
					localAuth.Date = time.Now()
					s.l.Info("new Date_Local: %s", localAuth.Date)

					user, err = s.backend.AddAuthMethodToUser(localAuth, user)

					if err != nil {
						s.l.Error("Unable to add AuthMethod %s to user %s", localAuth.Type, user.Name)
						s.doError(http.StatusInternalServerError, "Unable to link password to user", w, r)
					}

					s.backend.UpdateUser(user)

					if err != nil {
						s.l.Error("Unable to update user %s", user.Name)
						s.doError(http.StatusInternalServerError, "Unable to update user", w, r)
					}

					err = s.login(user, models.AUTH_LOCAL, w, r)
					if err != nil {
						s.l.Error("Unable to login to session:", err)
						s.doError(http.StatusInternalServerError, "Unable to update password", w, r)
					}

					http.Redirect(w, r, "/user", http.StatusFound)
				}
			}
		}
	}
	if err := s.executeTemplate(w, "account", data); err != nil {
		s.l.Error("Error rendering template: %v", err)
	}
}

// /user/login
func (s *webServer) handlerUserLogin(w http.ResponseWriter, r *http.Request) {

	err := r.ParseForm()
	if err != nil {
		s.l.Error("Error parsing login form: %v", err)
	}

	user := s.getSessionUser(w, r)
	if user != nil {
		http.Redirect(w, r, "/user", http.StatusFound)
		return
	}

	data := dataLoginForm{}
	doRedirect := false

	twitchAuth, err := s.backend.GetTwitchOauthEnabled()
	if err != nil {
		s.doError(http.StatusInternalServerError, "Something went wrong :C", w, r)
		s.l.Error("Unable to get ConfigTwitchOauthEnabled config value: %v", err)
		return
	}
	data.TwitchOAuth = twitchAuth

	discordAuth, err := s.backend.GetDiscordOauthEnabled()
	if err != nil {
		s.doError(http.StatusInternalServerError, "Something went wrong :C", w, r)
		s.l.Error("Unable to get ConfigDiscordOauthEnabled config value: %v", err)
		return
	}
	data.DiscordOAuth = discordAuth

	patreonAuth, err := s.backend.GetPatreonOauthEnabled()
	if err != nil {
		s.doError(http.StatusInternalServerError, "Something went wrong :C", w, r)
		s.l.Error("Unable to get ConfigPatreonOauthEnabled config value: %v", err)
		return
	}
	data.PatreonOAuth = patreonAuth

	data.OAuth = twitchAuth || discordAuth || patreonAuth

	if r.Method == "POST" {
		// do login

		un := r.PostFormValue("Username")
		pw := r.PostFormValue("Password")
		user, err = s.backend.UserLocalLogin(un, s.backend.HashPassword(pw))
		if err != nil {
			data.ErrorMessage = err.Error()
		} else {
			doRedirect = true
		}

	} else {
		s.l.Info("> no post: %s", r.Method)
	}

	if user != nil {
		err = s.login(user, models.AUTH_LOCAL, w, r)
		if err != nil {
			s.l.Error("Unable to login: %v", err)
			s.doError(http.StatusInternalServerError, "Unable to login", w, r)
			return
		}
	}

	// Redirect to base page on successful login
	if doRedirect {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	data.dataPageBase = s.newPageBase("Login", w, r) // set this last to get correct login status

	if err := s.executeTemplate(w, "simplelogin", data); err != nil {
		s.l.Error("Error rendering template: %v", err)
	}
}

// user/logout

func (s *webServer) handlerUserLogout(w http.ResponseWriter, r *http.Request) {
	err := s.logout(w, r)
	if err != nil {
		s.l.Error("Error logging out: %v", err)
	}

	http.Redirect(w, r, "/", http.StatusFound)
}

// /user/new

func (s *webServer) handlerUserNew(w http.ResponseWriter, r *http.Request) {
	user := s.getSessionUser(w, r)
	if user != nil {
		http.Redirect(w, r, "/account", http.StatusFound)
		return
	}

	data := struct {
		dataPageBase

		ErrorMessage []string
		ErrName      bool
		ErrPass      bool
		ErrEmail     bool

		OAuth         bool
		TwitchOAuth   bool
		TwitchSignup  bool
		DiscordOAuth  bool
		DiscordSignup bool
		PatreonOAuth  bool
		PatreonSignup bool
		LocalSignup   bool

		ValName           string
		ValEmail          string
		ValNotifyEnd      bool
		ValNotifySelected bool
	}{
		dataPageBase: s.newPageBase("Create Account", w, r),
	}

	doRedirect := false

	twitchAuth, err := s.backend.GetTwitchOauthEnabled()
	if err != nil {
		s.doError(http.StatusInternalServerError, "Something went wrong :C", w, r)
		s.l.Error("Unable to get ConfigTwitchOauthEnabled config value: %v", err)
		return
	}
	data.TwitchOAuth = twitchAuth

	twitchSignup, err := s.backend.GetTwitchOauthSignupEnabled()
	if err != nil {
		s.doError(http.StatusInternalServerError, "Something went wrong :C", w, r)
		s.l.Error("Unable to get ConfigTwitchOauthSignupEnabled config value: %v", err)
		return
	}
	data.TwitchSignup = twitchSignup

	discordAuth, err := s.backend.GetDiscordOauthEnabled()
	if err != nil {
		s.doError(http.StatusInternalServerError, "Something went wrong :C", w, r)
		s.l.Error("Unable to get ConfigDiscordOauthEnabled config value: %v", err)
		return
	}
	data.DiscordOAuth = discordAuth

	discordSignup, err := s.backend.GetDiscordOauthSignupEnabled()
	if err != nil {
		s.doError(http.StatusInternalServerError, "Something went wrong :C", w, r)
		s.l.Error("Unable to get ConfigDiscordOauthSignupEnabled config value: %v", err)
		return
	}
	data.DiscordSignup = discordSignup

	patreonAuth, err := s.backend.GetPatreonOauthEnabled()
	if err != nil {
		s.doError(http.StatusInternalServerError, "Something went wrong :C", w, r)
		s.l.Error("Unable to get ConfigPatreonOauthEnabled config value: %v", err)
		return
	}
	data.PatreonOAuth = patreonAuth

	patreonSignup, err := s.backend.GetPatreonOauthSignupEnabled()
	if err != nil {
		s.doError(http.StatusInternalServerError, "Something went wrong :C", w, r)
		s.l.Error("Unable to get ConfigPatreonOauthSignupEnabled config value: %v", err)
		return
	}
	data.PatreonSignup = patreonSignup

	localSignup, err := s.backend.GetLocalSignupEnabled()
	if err != nil {
		s.doError(http.StatusInternalServerError, "Something went wrong :C", w, r)
		s.l.Error("Unable to get ConfigLocalSignupEnabled config value: %v", err)
		return
	}
	data.LocalSignup = localSignup

	data.OAuth = twitchAuth || discordAuth || patreonAuth

	if r.Method == "POST" {
		err := r.ParseForm()
		if err != nil {
			s.l.Error("Error parsing login form: %v", err)
			data.ErrorMessage = append(data.ErrorMessage, err.Error())
		}

		un := strings.TrimSpace(r.PostFormValue("Username"))
		data.ValName = un

		// TODO: password requirements
		pw1 := r.PostFormValue("Password1")
		pw2 := r.PostFormValue("Password2")

		data.ValName = un

		if un == "" {
			data.ErrorMessage = append(data.ErrorMessage, "Username cannot be blank!")
			data.ErrName = true
		}

		maxlen, err := s.backend.GetMaxNameLength()
		if err != nil {
			s.doError(http.StatusInternalServerError, "Something went wrong :C", w, r)
			s.l.Error("Unable to get MaxNameLength config value: %v", err)
			return
		}

		minlen, err := s.backend.GetMinNameLength()
		if err != nil {
			s.doError(http.StatusInternalServerError, "Something went wrong :C", w, r)
			s.l.Error("Unable to get MinNameLength config value: %v", err)
			return
		}

		s.l.Debug("New user: %s (%d) maxlen: %d", un, len(un), maxlen)

		if len(un) > maxlen {
			data.ErrorMessage = append(data.ErrorMessage, fmt.Sprintf("Username cannot be longer than %d characters", maxlen))
			data.ErrName = true
		}

		if len(un) < minlen {
			data.ErrorMessage = append(data.ErrorMessage, fmt.Sprintf("Username cannot be shorter than %d characters", minlen))
			data.ErrName = true
		}

		if pw1 != pw2 {
			data.ErrorMessage = append(data.ErrorMessage, "Passwords do not match!")
			data.ErrPass = true

		} else if pw1 == "" {
			data.ErrorMessage = append(data.ErrorMessage, "Password cannot be blank!")
			data.ErrPass = true
		}

		notifyEnd := r.PostFormValue("NotifyEnd")
		notifySelected := r.PostFormValue("NotifySelected")
		email := r.PostFormValue("Email")

		data.ValEmail = email
		if notifyEnd != "" {
			data.ValNotifyEnd = true
		}

		if notifySelected != "" {
			data.ValNotifySelected = true
		}

		if (notifyEnd != "" || notifySelected != "") && email == "" {
			data.ErrEmail = true
			data.ErrorMessage = append(data.ErrorMessage, "Email required for notifications")
		}

		auth := &models.AuthMethod{
			Type:     models.AUTH_LOCAL,
			Password: s.backend.HashPassword(pw1),
			Date:     time.Now(),
		}

		if err != nil {
			s.l.Error(err.Error())
			data.ErrorMessage = append(data.ErrorMessage, "Could not create new User, message the server admin")
		}

		if len(data.ErrorMessage) == 0 {
			newUser := &models.User{
				Name:                un,
				Email:               email,
				NotifyCycleEnd:      data.ValNotifyEnd,
				NotifyVoteSelection: data.ValNotifySelected,
			}

			newUser, err = s.backend.AddAuthMethodToUser(auth, newUser)

			newUser.Id, err = s.backend.AddUser(newUser)
			if err != nil {
				data.ErrorMessage = append(data.ErrorMessage, err.Error())
			} else {
				err = s.login(newUser, models.AUTH_LOCAL, w, r)
				if err != nil {
					s.l.Error("Unable to login to session: %v", err)
					s.doError(http.StatusInternalServerError, "Login error", w, r)
					return
				}
				doRedirect = true
			}
		}
	}

	if doRedirect {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	if err := s.executeTemplate(w, "newaccount", data); err != nil {
		s.l.Error("Error rendering template: %v", err)
	}
}
