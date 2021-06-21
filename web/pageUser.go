package web

import (
	"net/http"
	"time"

	"github.com/zorchenhimer/MoviePolls/models"
)

func (s *webServer) handlerPageUser(w http.ResponseWriter, r *http.Request) {
	user := s.getSessionUser(w, r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	totalVotes := s.backend.GetMaxUserVotes()

	activeVotes, watchedVotes, err := s.backend.GetUserVotes(user)
	if err != nil {
		s.l.Error("Unable to get votes for user %d: %v", user.Id, err)
	}

	addedMovies, err := s.backend.GetUserMovies(user.Id)
	if err != nil {
		s.l.Error("Unable to get movies added by user %d: %v", user.Id, err)
	}

	unlimited := s.backend.GetUnlimitedVotes()

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
