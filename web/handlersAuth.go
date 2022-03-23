package web

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	u "net/url"
	"regexp"
	"strings"
	"time"

	"github.com/zorchenhimer/MoviePolls/models"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/twitch"
)

// just some global variables
var twitchOAuthConfig = &oauth2.Config{}
var discordOAuthConfig = &oauth2.Config{}
var patreonOAuthConfig = &oauth2.Config{}

var openStates = []string{}

// Welp we need to do the endpoints ourself i guess ...
var discordEndpoint = oauth2.Endpoint{
	AuthURL:  "https://discord.com/api/oauth2/authorize",
	TokenURL: "https://discord.com/api/oauth2/token",
}

var patreonEndpoint = oauth2.Endpoint{
	AuthURL:  "https://www.patreon.com/oauth2/authorize",
	TokenURL: "https://www.patreon.com/api/oauth2/token",
}

// Initiate the OAuth configs, this includes loading the ConfigValues into "memory" to be used in the login methods
// Returns: Error if a config value could not be retrieved
func (s *webServer) initOauth() error {

	twitchOauthEnabled, err := s.backend.GetTwitchOauthEnabled()
	if err != nil {
		return err
	}

	discordOAuthEnabled, err := s.backend.GetDiscordOauthEnabled()
	if err != nil {
		return err
	}

	patreonOAuthEnabled, err := s.backend.GetPatreonOauthEnabled()
	if err != nil {
		return err
	}

	baseUrl, err := s.backend.GetHostAddress()
	if err != nil {
		return err
	}

	if twitchOauthEnabled || discordOAuthEnabled || patreonOAuthEnabled {
		if baseUrl == "" {
			return fmt.Errorf("Config Value for HostAddress cannot be empty to use OAuth")
		}
	}

	if twitchOauthEnabled {
		twitchClientID, err := s.backend.GetTwitchOauthClientID()
		if err != nil {
			return err
		}
		if twitchClientID == "" {
			return fmt.Errorf("Config Value for TwitchOauthClientID cannot be empty to use OAuth")
		}

		twitchClientSecret, err := s.backend.GetTwitchOauthClientSecret()
		if err != nil {
			return err
		}

		if twitchClientSecret == "" {
			return fmt.Errorf("Config Value for TwitchOauthClientSecret cannot be empty to use OAuth")
		}

		twitchOAuthConfig = &oauth2.Config{
			RedirectURL:  baseUrl + "/oauth/twitch/callback",
			ClientID:     twitchClientID,
			ClientSecret: twitchClientSecret,
			Scopes:       []string{"user:read:email"},
			Endpoint:     twitch.Endpoint, //this endpoint is predefined in the oauth2 package
		}
	}

	if discordOAuthEnabled {
		discordClientID, err := s.backend.GetDiscordOauthClientID()
		if err != nil {
			return err
		}

		if discordClientID == "" {
			return fmt.Errorf("Config Value for DiscordOauthClientID cannot be empty to use OAuth")
		}

		discordClientSecret, err := s.backend.GetDiscordOauthClientSecret()
		if err != nil {
			return err
		}

		if discordClientSecret == "" {
			return fmt.Errorf("Config Value for DiscordOauthClientSecret cannot be empty to use OAuth")
		}

		discordOAuthConfig = &oauth2.Config{
			RedirectURL:  baseUrl + "/oauth/discord/callback",
			ClientID:     discordClientID,
			ClientSecret: discordClientSecret,
			Scopes:       []string{"email", "identify"},
			Endpoint:     discordEndpoint,
		}
	}

	if patreonOAuthEnabled {
		patreonClientID, err := s.backend.GetPatreonOauthClientID()
		if err != nil {
			return err
		}

		if patreonClientID == "" {
			return fmt.Errorf("Config Value for PatreonOauthClientSecret cannot be empty to use OAuth")
		}

		patreonClientSecret, err := s.backend.GetPatreonOauthClientSecret()
		if err != nil {
			return err
		}

		if patreonClientSecret == "" {
			return fmt.Errorf("Config Value for PatreonOauthClientSecret cannot be empty to use OAuth")
		}

		patreonOAuthConfig = &oauth2.Config{
			RedirectURL:  baseUrl + "/oauth/patreon/callback",
			ClientID:     patreonClientID,
			ClientSecret: patreonClientSecret,
			Scopes:       []string{"identity", "identity[email]"},
			Endpoint:     patreonEndpoint,
		}
	}

	return nil
}

// Removes the AuthType LOCAL AuthMethod from the currently logged in user
func (s *webServer) handlerLocalAuthRemove(w http.ResponseWriter, r *http.Request) {
	s.l.Debug("local remove")

	user := s.getSessionUser(w, r)

	auth, err := user.GetAuthMethod(models.AUTH_LOCAL)

	if err != nil {
		s.l.Info("User %s does not have a password associated with him", user.Name)
		http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
		return
	}

	if len(user.AuthMethods) == 1 {
		s.l.Info("User %v only has the local Authmethod associated with him", user.Name)
		http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
		return
	}

	user, err = s.backend.RemoveAuthMethodFromUser(auth, user)

	if err != nil {
		s.l.Info("Could not remove password from user. %s", err.Error())
		http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
		return
	}

	err = s.backend.UpdateUser(user)
	if err != nil {
		s.l.Info("Could not update user %s", user.Name)
		http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
		return
	}

	// Logging the user out to ensure that he is logged in with an existing AuthMethod
	err = s.logout(w, r)
	if err != nil {
		s.l.Info("Could not logout user %s", user.Name)
		http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
		return
	}

	// Logging the user back in
	s.saveLoginUser(user, w, r)

	http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
	return
}

func (s *webServer) saveLoginUser(user *models.User, w http.ResponseWriter, r *http.Request) {
	if _, err := user.GetAuthMethod(models.AUTH_LOCAL); err == nil {
		err = s.login(user, models.AUTH_LOCAL, w, r)
		if err != nil {
			s.l.Info("Could not login user %s", user.Name)
		}
	} else if _, err := user.GetAuthMethod(models.AUTH_TWITCH); err == nil {
		err = s.login(user, models.AUTH_TWITCH, w, r)
		if err != nil {
			s.l.Info("Could not login user %s", user.Name)
		}
	} else if _, err := user.GetAuthMethod(models.AUTH_DISCORD); err == nil {
		err = s.login(user, models.AUTH_DISCORD, w, r)
		if err != nil {
			s.l.Info("Could not login user %s", user.Name)
		}
	} else if _, err := user.GetAuthMethod(models.AUTH_PATREON); err == nil {
		err = s.login(user, models.AUTH_PATREON, w, r)
		if err != nil {
			s.l.Info("Could not login user %s", user.Name)
		}
	}
}

func (s *webServer) handlerTwitchOAuth(w http.ResponseWriter, r *http.Request) {
	action := r.URL.Query().Get("action")

	switch action {
	case "login":
		// Generate a new state string for each login attempt and store it in the state list
		oauthStateString := "login_" + s.backend.GetCryptRandKey(32)
		openStates = append(openStates, oauthStateString)

		// Handle the Oauth redirect
		url := twitchOAuthConfig.AuthCodeURL(oauthStateString)
		http.Redirect(w, r, url, http.StatusTemporaryRedirect)

		s.l.Debug("twitch login")

	case "signup":
		// Generate a new state string for each login attempt and store it in the state list
		oauthStateString := "signup_" + s.backend.GetCryptRandKey(32)
		openStates = append(openStates, oauthStateString)

		// Handle the Oauth redirect
		url := twitchOAuthConfig.AuthCodeURL(oauthStateString)
		http.Redirect(w, r, url, http.StatusTemporaryRedirect)

		s.l.Debug("twitch sign up")

	case "add":
		// Generate a new state string for each login attempt and store it in the state list
		oauthStateString := "add_" + s.backend.GetCryptRandKey(32)
		openStates = append(openStates, oauthStateString)

		// Handle the Oauth redirect
		url := twitchOAuthConfig.AuthCodeURL(oauthStateString)
		http.Redirect(w, r, url, http.StatusTemporaryRedirect)

		s.l.Debug("twitch add")

	case "remove":
		user := s.getSessionUser(w, r)

		auth, err := user.GetAuthMethod(models.AUTH_TWITCH)

		if err != nil {
			s.l.Info("User %s does not have Twitch Oauth associated with him", user.Name)
			http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
			return
		}

		if len(user.AuthMethods) == 1 {
			s.l.Info("User %v only has Twitch Oauth associated with him", user.Name)
			http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
			return
		}

		user, err = s.backend.RemoveAuthMethodFromUser(auth, user)

		if err != nil {
			s.l.Info("Could not remove Twitch Oauth from user. %s", err.Error())
			http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
			return
		}

		err = s.backend.UpdateUser(user)
		if err != nil {
			s.l.Info("Could not update user %s", user.Name)
			http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
			return
		}

		// Log the user out to ensure he uses an existing AuthMethod
		err = s.logout(w, r)
		if err != nil {
			s.l.Info("Could not logout user %s", user.Name)
			http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
			return
		}

		// Find a new AuthMethod to log the user back in
		s.saveLoginUser(user, w, r)

		http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
		return
	}
}

// This function handles all Twitch Callbacks (add/signup/login)
func (s *webServer) handlerTwitchOAuthCallback(w http.ResponseWriter, r *http.Request) {
	state := r.FormValue("state")

	ok := false
	for _, expectedState := range openStates {
		if state == expectedState {
			ok = true
		}
	}
	if !ok {
		s.l.Info("Invalid/Unknown OAuth state string: '%s'", state)
		http.Redirect(w, r, "/user/login", http.StatusTemporaryRedirect)
		return
	}

	code := r.FormValue("code")
	token, err := twitchOAuthConfig.Exchange(oauth2.NoContext, code)
	if err != nil {
		s.l.Info("Code exchange failed: %s", err)
		http.Redirect(w, r, "/user/login", http.StatusTemporaryRedirect)
		return
	}

	// Request the User data from the API
	req, err := http.NewRequest("GET", "https://api.twitch.tv/helix/users", nil)
	req.Header.Add("Authorization", "Bearer "+token.AccessToken)
	req.Header.Add("Client-Id", twitchOAuthConfig.ClientID)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		s.l.Info("Could not retrieve Userdata from Twitch API: %s", err)
		http.Redirect(w, r, "/user/login", http.StatusTemporaryRedirect)
		return

	}

	if resp.StatusCode != 200 {
		s.l.Info("Status Code is not 200, its %v", resp.Status)
		http.Redirect(w, r, "/user/login", http.StatusTemporaryRedirect)
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		s.l.Info(err.Error())
		http.Redirect(w, r, "/user/login", http.StatusTemporaryRedirect)
		return
	}

	var data map[string][]map[string]interface{}

	if err := json.Unmarshal(body, &data); err != nil {
		s.l.Info(err.Error())
		http.Redirect(w, r, "/user/login", http.StatusTemporaryRedirect)
		return
	}

	if strings.HasPrefix(state, "signup_") {
		// Handle the sign up process
		auth := &models.AuthMethod{
			Type:         models.AUTH_TWITCH,
			ExtId:        data["data"][0]["id"].(string),
			AuthToken:    token.AccessToken,
			RefreshToken: token.RefreshToken,
			Date:         token.Expiry,
		}

		// check if Twitch Auth is already used
		if !s.backend.CheckOauthUsage(auth.ExtId, auth.Type) {
			// Create a new user
			newUser := &models.User{
				Name:                data["data"][0]["display_name"].(string),
				Email:               data["data"][0]["email"].(string),
				NotifyCycleEnd:      false,
				NotifyVoteSelection: false,
			}

			// add this new user to the database
			newUser.Id, err = s.backend.AddUser(newUser)

			if err != nil {
				s.l.Info(err.Error())
				http.Redirect(w, r, "/user/login", http.StatusTemporaryRedirect)
				return
			}

			// add the authmethod to the user
			newUser, err = s.backend.AddAuthMethodToUser(auth, newUser)

			if err != nil {
				s.l.Info(err.Error())
				http.Redirect(w, r, "/user/login", http.StatusTemporaryRedirect)
				return
			}

			// update the user in the DB with the user having the AuthMethod associated
			err = s.backend.UpdateUser(newUser)

			if err != nil {
				s.l.Info(err.Error())
				http.Redirect(w, r, "/user/login", http.StatusTemporaryRedirect)
				return
			}

			s.l.Debug("logging in %v", newUser.Name)

			err := s.login(newUser, models.AUTH_TWITCH, w, r)

			if err != nil {
				s.l.Info(err.Error())
				http.Redirect(w, r, "/user/login", http.StatusTemporaryRedirect)
				return
			}

			http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		} else {
			s.l.Debug("AuthMethod already used")
			http.Redirect(w, r, "/user/new", http.StatusTemporaryRedirect)
		}
	} else if strings.HasPrefix(state, "login_") {
		// Handle Twitch Login
		user, err := s.backend.UserTwitchLogin(data["data"][0]["id"].(string))
		if err != nil {
			s.l.Info(err.Error())
			http.Redirect(w, r, "/user/login", http.StatusTemporaryRedirect)
			return
		}
		s.l.Debug("logging in %v", user.Name)

		err = s.login(user, models.AUTH_TWITCH, w, r)

		if err != nil {
			s.l.Info(err.Error())
			http.Redirect(w, r, "/user/login", http.StatusTemporaryRedirect)
			return
		}

		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
	} else if strings.HasPrefix(state, "add_") {
		// Handle adding a Twitch AuthMethod to the logged in user

		// get the current user
		user := s.getSessionUser(w, r)

		auth := &models.AuthMethod{
			Type:         models.AUTH_TWITCH,
			ExtId:        data["data"][0]["id"].(string),
			AuthToken:    token.AccessToken,
			RefreshToken: token.RefreshToken,
			Date:         token.Expiry,
		}

		// check if this oauth is already used
		if !s.backend.CheckOauthUsage(auth.ExtId, auth.Type) {
			// check if the user already has an other Twitch OAuth connected
			_, err = user.GetAuthMethod(auth.Type)
			if err != nil {
				_, err = s.backend.AddAuthMethodToUser(auth, user)

				if err != nil {
					s.l.Info(err.Error())
					http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
					return
				}

				err = s.backend.UpdateUser(user)

				if err != nil {
					s.l.Info(err.Error())
					http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
					return
				}
			} else {
				s.l.Info("User %s already has %s Oauth associated", user.Name, auth.Type)
				http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
				return
			}
		} else {
			s.l.Info("The provided Oauth login is already used")

			s.callbackError = callbackError{
				user:    user.Id,
				message: "The provided Oauth login is already used",
			}
		}
		http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
		return
	}
	return
}

func (s *webServer) handlerDiscordOAuth(w http.ResponseWriter, r *http.Request) {

	action := r.URL.Query().Get("action")

	switch action {
	case "login":
		// Generate a new state string for each login attempt and store it in the state list
		oauthStateString := "login_" + s.backend.GetCryptRandKey(32)
		openStates = append(openStates, oauthStateString)

		// Handle the Oauth redirect
		url := discordOAuthConfig.AuthCodeURL(oauthStateString)
		http.Redirect(w, r, url, http.StatusTemporaryRedirect)

		s.l.Debug("discord login")

	case "signup":
		// Generate a new state string for each login attempt and store it in the state list
		oauthStateString := "signup_" + s.backend.GetCryptRandKey(32)
		openStates = append(openStates, oauthStateString)

		// Handle the Oauth redirect
		url := discordOAuthConfig.AuthCodeURL(oauthStateString)
		http.Redirect(w, r, url, http.StatusTemporaryRedirect)

		s.l.Debug("discord signup")

	case "add":
		// Generate a new state string for each login attempt and store it in the state list
		oauthStateString := "add_" + s.backend.GetCryptRandKey(32)
		openStates = append(openStates, oauthStateString)

		// Handle the Oauth redirect
		url := discordOAuthConfig.AuthCodeURL(oauthStateString)
		http.Redirect(w, r, url, http.StatusTemporaryRedirect)

		s.l.Debug("discord add")

	case "remove":
		user := s.getSessionUser(w, r)

		auth, err := user.GetAuthMethod(models.AUTH_DISCORD)

		if err != nil {
			s.l.Info("User %s does not have Discord Oauth associated with him", user.Name)
			http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
			return
		}

		if len(user.AuthMethods) == 1 {
			s.l.Info("User %v only has Discord Oauth associated with him", user.Name)
			http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
			return
		}

		user, err = s.backend.RemoveAuthMethodFromUser(auth, user)

		if err != nil {
			s.l.Info("Could not remove Discord Oauth from user. %s", err.Error())
			http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
			return
		}

		err = s.backend.UpdateUser(user)
		if err != nil {
			s.l.Info("Could not update user %s", user.Name)
			http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
			return
		}

		// Log the user out to ensure he is logged in with an existing AuthMethod
		err = s.logout(w, r)
		if err != nil {
			s.l.Info("Could not logout user %s", user.Name)
			http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
			return
		}

		// Try to log the user back in
		s.saveLoginUser(user, w, r)

		http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
		return
	}
}

// Handler for the Discord OAuth Callbacks (add/signup/login)
func (s *webServer) handlerDiscordOAuthCallback(w http.ResponseWriter, r *http.Request) {
	state := r.FormValue("state")

	ok := false
	for _, expectedState := range openStates {
		if state == expectedState {
			ok = true
		}
	}
	if !ok {
		s.l.Info("Invalid/Unknown OAuth state string: '%s'", state)
		http.Redirect(w, r, "/user/login", http.StatusTemporaryRedirect)
		return
	}

	code := r.FormValue("code")
	token, err := discordOAuthConfig.Exchange(oauth2.NoContext, code)
	if err != nil {
		s.l.Info("Code exchange failed: %s", err)
		http.Redirect(w, r, "/user/login", http.StatusTemporaryRedirect)
		return
	}

	// Request the User data from the API
	req, err := http.NewRequest("GET", "https://discord.com/api/users/@me", nil)
	req.Header.Add("Authorization", "Bearer "+token.AccessToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		s.l.Info("Could not retrieve Userdata from Discord API: %s", err)
		http.Redirect(w, r, "/user/login", http.StatusTemporaryRedirect)
		return
	}

	if resp.StatusCode != 200 {
		s.l.Info("Status Code is not 200, its %v", resp.Status)
		http.Redirect(w, r, "/user/login", http.StatusTemporaryRedirect)
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		s.l.Info(err.Error())
		http.Redirect(w, r, "/user/login", http.StatusTemporaryRedirect)
		return
	}

	var data map[string]interface{}

	if err := json.Unmarshal(body, &data); err != nil {
		s.l.Info(err.Error())
		http.Redirect(w, r, "/user/login", http.StatusTemporaryRedirect)
		return
	}

	if strings.HasPrefix(state, "signup_") {

		auth := &models.AuthMethod{
			Type:         models.AUTH_DISCORD,
			ExtId:        data["id"].(string),
			AuthToken:    token.AccessToken,
			RefreshToken: token.RefreshToken,
			Date:         token.Expiry,
		}

		if !s.backend.CheckOauthUsage(auth.ExtId, auth.Type) {
			newUser := &models.User{
				Name:                data["username"].(string),
				Email:               data["email"].(string),
				NotifyCycleEnd:      false,
				NotifyVoteSelection: false,
			}

			newUser.Id, err = s.backend.AddUser(newUser)

			if err != nil {
				s.l.Info(err.Error())
				http.Redirect(w, r, "/user/login", http.StatusTemporaryRedirect)
				return
			}

			newUser, err = s.backend.AddAuthMethodToUser(auth, newUser)

			if err != nil {
				s.l.Info(err.Error())
				http.Redirect(w, r, "/user/login", http.StatusTemporaryRedirect)
				return
			}

			err = s.backend.UpdateUser(newUser)

			if err != nil {
				s.l.Info(err.Error())
				http.Redirect(w, r, "/user/login", http.StatusTemporaryRedirect)
				return
			}

			s.l.Debug("logging in %v", newUser.Name)
			err := s.login(newUser, models.AUTH_DISCORD, w, r)

			if err != nil {
				s.l.Info(err.Error())
				http.Redirect(w, r, "/user/login", http.StatusTemporaryRedirect)
				return
			}

			http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		} else {
			s.l.Debug("AuthMethod already used")
			http.Redirect(w, r, "/user/new", http.StatusTemporaryRedirect)
		}
	} else if strings.HasPrefix(state, "login_") {
		user, err := s.backend.UserDiscordLogin(data["id"].(string))
		if err != nil {
			s.l.Info(err.Error())
			http.Redirect(w, r, "/user/login", http.StatusTemporaryRedirect)
			return
		}
		s.l.Debug("logging in %v", user.Name)
		err = s.login(user, models.AUTH_DISCORD, w, r)

		if err != nil {
			s.l.Info(err.Error())
			http.Redirect(w, r, "/user/login", http.StatusTemporaryRedirect)
			return
		}

		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
	} else if strings.HasPrefix(state, "add_") {
		user := s.getSessionUser(w, r)

		auth := &models.AuthMethod{
			Type:         models.AUTH_DISCORD,
			ExtId:        data["id"].(string),
			AuthToken:    token.AccessToken,
			RefreshToken: token.RefreshToken,
			Date:         token.Expiry,
		}

		if !s.backend.CheckOauthUsage(auth.ExtId, auth.Type) {
			_, err = user.GetAuthMethod(auth.Type)
			if err != nil {
				_, err = s.backend.AddAuthMethodToUser(auth, user)

				if err != nil {
					s.l.Info(err.Error())
					http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
					return
				}

				err = s.backend.UpdateUser(user)

				if err != nil {
					s.l.Info(err.Error())
					http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
					return
				}
			} else {
				s.l.Info("User %s already has %s Oauth associated", user.Name, auth.Type)
				http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
				return
			}
		} else {
			s.l.Info("The provided Oauth login is already used")

			s.callbackError = callbackError{
				user:    user.Id,
				message: "The provided Oauth login is already used",
			}
		}
		http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
		return
	}
	return
}

func (s *webServer) handlerPatreonOAuth(w http.ResponseWriter, r *http.Request) {
	action := r.URL.Query().Get("action")

	switch action {
	case "login":
		// Generate a new state string for each login attempt and store it in the state list
		oauthStateString := "login_" + s.backend.GetCryptRandKey(32)
		openStates = append(openStates, oauthStateString)

		// Handle the Oauth redirect
		url := patreonOAuthConfig.AuthCodeURL(oauthStateString)
		http.Redirect(w, r, url, http.StatusTemporaryRedirect)

		s.l.Debug("patreon login")

	case "signup":
		// Generate a new state string for each login attempt and store it in the state list
		oauthStateString := "signup_" + s.backend.GetCryptRandKey(32)
		openStates = append(openStates, oauthStateString)

		// Handle the Oauth redirect
		url := patreonOAuthConfig.AuthCodeURL(oauthStateString)
		http.Redirect(w, r, url, http.StatusTemporaryRedirect)

		s.l.Debug("patreon signup")

	case "add":
		// Generate a new state string for each login attempt and store it in the state list
		oauthStateString := "add_" + s.backend.GetCryptRandKey(32)
		openStates = append(openStates, oauthStateString)

		// Handle the Oauth redirect
		url := patreonOAuthConfig.AuthCodeURL(oauthStateString)
		http.Redirect(w, r, url, http.StatusTemporaryRedirect)

		s.l.Debug("patreon add")

	case "remove":
		user := s.getSessionUser(w, r)

		auth, err := user.GetAuthMethod(models.AUTH_PATREON)

		if err != nil {
			s.l.Info("User %s does not have Patreon Oauth associated with him", user.Name)
			http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
			return
		}

		if len(user.AuthMethods) == 1 {
			s.l.Info("User %v only has Patreon Oauth associated with him", user.Name)
			http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
			return
		}

		user, err = s.backend.RemoveAuthMethodFromUser(auth, user)

		if err != nil {
			s.l.Info("Could not remove Patreon Oauth from user. %s", err.Error())
			http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
			return
		}

		err = s.backend.UpdateUser(user)
		if err != nil {
			s.l.Info("Could not update user %s", user.Name)
			http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
			return
		}

		err = s.logout(w, r)
		if err != nil {
			s.l.Info("Could not logout user %s", user.Name)
			http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
			return
		}

		if _, err := user.GetAuthMethod(models.AUTH_TWITCH); err == nil {
			err = s.login(user, models.AUTH_TWITCH, w, r)
			if err != nil {
				s.l.Info("Could not login user %s", user.Name)
				http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
				return
			}
		} else if _, err := user.GetAuthMethod(models.AUTH_DISCORD); err == nil {
			err = s.login(user, models.AUTH_DISCORD, w, r)
			if err != nil {
				s.l.Info("Could not login user %s", user.Name)
				http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
				return
			}
		} else if _, err := user.GetAuthMethod(models.AUTH_LOCAL); err == nil {
			err = s.login(user, models.AUTH_LOCAL, w, r)
			if err != nil {
				s.l.Info("Could not login user %s", user.Name)
				http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
				return
			}
		}

		s.l.Debug("patreon remove")

		http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
		return
	}
}

func (s *webServer) handlerPatreonOAuthCallback(w http.ResponseWriter, r *http.Request) {
	state := r.FormValue("state")

	ok := false
	for _, expectedState := range openStates {
		if state == expectedState {
			ok = true
		}
	}
	if !ok {
		s.l.Info("Invalid/Unknown OAuth state string: '%s'", state)
		http.Redirect(w, r, "/user/login", http.StatusTemporaryRedirect)
		return
	}

	code := r.FormValue("code")
	token, err := patreonOAuthConfig.Exchange(oauth2.NoContext, code)
	if err != nil {
		s.l.Info("Code exchange failed: %s", err)
		http.Redirect(w, r, "/user/login", http.StatusTemporaryRedirect)
		return
	}

	// Request the User data from the API
	req, err := http.NewRequest("GET", "https://www.patreon.com/api/oauth2/v2/identity?fields"+u.QueryEscape("[user]")+"=email,first_name,full_name,last_name,vanity", nil)
	req.Header.Add("Authorization", "Bearer "+token.AccessToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		s.l.Info("Could not retrieve Userdata from Patreon API: %s", err)
		http.Redirect(w, r, "/user/login", http.StatusTemporaryRedirect)
		return

	}
	if resp.StatusCode != 200 {
		s.l.Info("Status Code is not 200, its %v", resp.Status)
		http.Redirect(w, r, "/user/login", http.StatusTemporaryRedirect)
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		s.l.Info(err.Error())
		http.Redirect(w, r, "/user/login", http.StatusTemporaryRedirect)
		return
	}

	var data map[string]interface{}

	if err := json.Unmarshal(body, &data); err != nil {
		s.l.Info(err.Error())
		http.Redirect(w, r, "/user/login", http.StatusTemporaryRedirect)
		return
	}

	data = data["data"].(map[string]interface{})

	if strings.HasPrefix(state, "signup_") {

		auth := &models.AuthMethod{
			Type:         models.AUTH_PATREON,
			ExtId:        data["id"].(string),
			AuthToken:    token.AccessToken,
			RefreshToken: token.RefreshToken,
			Date:         token.Expiry,
		}

		if !s.backend.CheckOauthUsage(auth.ExtId, auth.Type) {

			newUser := &models.User{
				Name:                data["attributes"].(map[string]interface{})["full_name"].(string),
				Email:               data["attributes"].(map[string]interface{})["email"].(string),
				NotifyCycleEnd:      false,
				NotifyVoteSelection: false,
			}

			newUser.Id, err = s.backend.AddUser(newUser)

			if err != nil {
				s.l.Info(err.Error())
				http.Redirect(w, r, "/user/login", http.StatusTemporaryRedirect)
				return
			}

			newUser, err = s.backend.AddAuthMethodToUser(auth, newUser)

			if err != nil {
				s.l.Info(err.Error())
				http.Redirect(w, r, "/user/login", http.StatusTemporaryRedirect)
				return
			}

			err = s.backend.UpdateUser(newUser)

			if err != nil {
				s.l.Info(err.Error())
				http.Redirect(w, r, "/user/login", http.StatusTemporaryRedirect)
				return
			}

			s.l.Debug("logging in %v", newUser.Name)

			err = s.login(newUser, models.AUTH_PATREON, w, r)

			if err != nil {
				s.l.Info(err.Error())
				http.Redirect(w, r, "/user/login", http.StatusTemporaryRedirect)
				return
			}
			http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		} else {
			s.l.Info("AuthMethod already used")
			http.Redirect(w, r, "/user/new", http.StatusTemporaryRedirect)
		}
	} else if strings.HasPrefix(state, "login_") {
		user, err := s.backend.UserPatreonLogin(data["id"].(string))
		if err != nil {
			s.l.Info(err.Error())
			http.Redirect(w, r, "/user/login", http.StatusTemporaryRedirect)
			return
		}
		s.l.Debug("logging in %v", user.Name)
		err = s.login(user, models.AUTH_PATREON, w, r)
		if err != nil {
			s.l.Info(err.Error())
			http.Redirect(w, r, "/user/login", http.StatusTemporaryRedirect)
			return
		}
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
	} else if strings.HasPrefix(state, "add_") {
		user := s.getSessionUser(w, r)

		auth := &models.AuthMethod{
			Type:         models.AUTH_PATREON,
			ExtId:        data["id"].(string),
			AuthToken:    token.AccessToken,
			RefreshToken: token.RefreshToken,
			Date:         token.Expiry,
		}

		if !s.backend.CheckOauthUsage(auth.ExtId, auth.Type) {
			_, err = user.GetAuthMethod(auth.Type)
			if err != nil {
				_, err = s.backend.AddAuthMethodToUser(auth, user)

				if err != nil {
					s.l.Info(err.Error())
					http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
					return
				}

				err = s.backend.UpdateUser(user)

				if err != nil {
					s.l.Info(err.Error())
					http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
					return
				}
			} else {
				s.l.Info("User %s already has %s Oauth associated", user.Name, auth.Type)
				http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
				return
			}
		} else {
			s.l.Info("The provided Oauth login is already used")

			s.callbackError = callbackError{
				user:    user.Id,
				message: "The provided Oauth login is already used",
			}
		}
		http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
		return
	}
	return
}

var re_auth = regexp.MustCompile(`^/auth/([^/#?]+)$`)

func (s *webServer) handlerAuth(w http.ResponseWriter, r *http.Request) {
	user := s.getSessionUser(w, r)

	s.l.Debug("[auth] RawQuery: %s", r.URL.RawQuery)
	s.l.Debug("[auth] Path: %s", r.URL.Path)

	matches := re_auth.FindStringSubmatch(r.URL.Path)
	var urlKey *models.UrlKey
	var ok bool
	if len(matches) != 2 {
		s.l.Debug("[auth] len != 2; matches: %v", matches)
		s.doError(http.StatusNotFound, fmt.Sprintf("%q not found", r.URL.Path), w, r)
		return
	}

	if urlKey, ok = s.backend.GetUrlKeys()[matches[1]]; !ok {
		s.l.Debug("[auth] map !ok; matches: %v", matches)
		mkeys := []string{}
		for key, _ := range s.backend.GetUrlKeys() {
			mkeys = append(mkeys, key)
		}
		s.l.Debug("[auth] map keys: %v", mkeys)
		s.doError(http.StatusNotFound, fmt.Sprintf("%q not found", r.URL.Path), w, r)
		return
	}

	var formError string
	var key string
	if r.Method == "POST" {
		err := r.ParseForm()
		if err != nil {
			s.l.Error("[auth] ParseForm(): %v", err)
			s.doError(http.StatusNotFound, "Something went wrong :C", w, r)
			return
		}

		key = strings.TrimSpace(r.PostFormValue("Key"))
		s.l.Debug("[auth] POST; key: %q", key)
	} else {
		s.l.Debug("[auth] RawQuery: %s", r.URL.RawQuery)
		key = r.URL.RawQuery
	}

	if key != "" && key != urlKey.Key {
		formError = "Invalid Key"
		goto renderPage
	}

	switch urlKey.Type {
	case models.UKT_AdminAuth:
		if user == nil {
			s.doError(http.StatusNotFound, fmt.Sprintf("%q not found", r.URL.Path), w, r)
			return
		}

		if key != "" {
			user.Privilege = 2
			err := s.backend.UpdateUser(user)
			if err != nil {
				s.doError(
					http.StatusInternalServerError,
					fmt.Sprintf("Unable to update user: %v", err),
					w, r)
				return
			}

			s.l.Info("%s has claimed Admin", user.Name)
			s.backend.DeleteUrlKey(key)
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

	case models.UKT_PasswordReset:
		s.l.Debug("Password top; key: %q", key)

		if key != "" {
			if r.Method == "POST" {
				s.l.Debug("Password POST")
				err := r.ParseForm()
				if err != nil {
					s.l.Error("[auth] ParseForm(): %v", err)
					s.doError(http.StatusNotFound, "Something went wrong :C", w, r)
					return
				}

				pass1 := r.PostFormValue("password1")
				pass2 := r.PostFormValue("password2")

				if pass1 != pass2 {
					s.l.Debug("Passwords do not match match")
					formError = "Passwords do not match!"
				} else if pass1 == "" {
					s.l.Debug("Passwords are blank")
					formError = "Password cannot be blank!"
				} else {
					s.l.Debug("Passwords match, saving it")
					user, err := s.backend.GetUser(urlKey.UserId)
					if err != nil {
						s.l.Error("[auth] GetUser(): %v", err)
						s.doError(http.StatusNotFound, "Something went wrong :C", w, r)
						return
					} else if user == nil {
						s.l.Error("User not found with ID %d", urlKey.UserId)
						s.doError(http.StatusNotFound, "Something went wrong :C", w, r)
						return
					}

					var localAuth *models.AuthMethod
					for _, auth := range user.AuthMethods {
						if auth.Type == models.AUTH_LOCAL {
							localAuth = auth
							break
						}
					}

					localAuth.Password = s.backend.HashPassword(pass1)
					localAuth.Date = time.Now()

					if err = s.backend.UpdateAuthMethod(localAuth); err != nil {
						s.l.Error("Unable to save AuthMethod with new password:", err)
						s.doError(http.StatusInternalServerError, "Unable to update password", w, r)
						return
					}

					if err = s.login(user, models.AUTH_LOCAL, w, r); err != nil {
						s.l.Error("Unable to login to session:", err)
						s.doError(http.StatusInternalServerError, "Something went wrong :C", w, r)
						return
					}

					s.l.Info("User %q has reset their password", user.Name)
					s.backend.DeleteUrlKey(key)
					http.Redirect(w, r, "/", http.StatusSeeOther)
					return
				}
			} // if POST

			s.l.Debug("Rendering password reset form")
			data := struct {
				dataPageBase
				UrlKey *models.UrlKey
				Error  string
			}{
				dataPageBase: s.newPageBase("Auth", w, r),
				UrlKey:       urlKey,
				Error:        formError,
			}

			if err := s.executeTemplate(w, "passwordReset", data); err != nil {
				s.l.Error("Error rendering template: %v", err)
			}
			return
		}
	}

renderPage:

	s.l.Debug("Rendering key form")
	data := struct {
		dataPageBase
		Url   string
		Error string
	}{
		dataPageBase: s.newPageBase("Auth", w, r),
		Url:          urlKey.Url,
		Error:        formError,
	}

	if err := s.executeTemplate(w, "auth", data); err != nil {
		s.l.Error("Error rendering template: %v", err)
	}
}
