package moviepoll

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	u "net/url"
	"strings"

	"github.com/zorchenhimer/MoviePolls/common"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/twitch"
)

// just some global variables
var twitchOAuthConfig = &oauth2.Config{}
var discordOAuthConfig = &oauth2.Config{}
var patreonOAuthConfig = &oauth2.Config{}

// var oauthStateString string
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
func (s *Server) initOauth() error {
	baseUrl, err := s.data.GetCfgString(ConfigHostAddress, "")
	if err != nil {
		return err
	}

	twitchOauthEnabled, err := s.data.GetCfgBool(ConfigTwitchOauthEnabled, DefaultTwitchOauthEnabled)
	if err != nil {
		return err
	}
	if twitchOauthEnabled {
		twitchClientID, err := s.data.GetCfgString(ConfigTwitchOauthClientID, DefaultTwitchOauthClientID)
		if err != nil {
			return err
		}

		twitchClientSecret, err := s.data.GetCfgString(ConfigTwitchOauthClientSecret, DefaultTwitchOauthClientSecret)
		if err != nil {
			return err
		}

		twitchOAuthConfig = &oauth2.Config{
			RedirectURL:  baseUrl + "/user/callback/twitch",
			ClientID:     twitchClientID,
			ClientSecret: twitchClientSecret,
			Scopes:       []string{"user:read:email"},
			Endpoint:     twitch.Endpoint, //this endpoint is predefined in the oauth2 package
		}
	}

	discordOAuthEnabled, err := s.data.GetCfgBool(ConfigDiscordOauthEnabled, DefaultDiscordOauthEnabled)
	if err != nil {
		return err
	}

	if discordOAuthEnabled {
		discordClientID, err := s.data.GetCfgString(ConfigDiscordOauthClientID, DefaultDiscordOauthClientID)
		if err != nil {
			return err
		}

		discordClientSecret, err := s.data.GetCfgString(ConfigDiscordOauthClientSecret, DefaultDiscordOauthClientSecret)
		if err != nil {
			return err
		}

		discordOAuthConfig = &oauth2.Config{
			RedirectURL:  baseUrl + "/user/callback/discord",
			ClientID:     discordClientID,
			ClientSecret: discordClientSecret,
			Scopes:       []string{"email", "identify"},
			Endpoint:     discordEndpoint,
		}
	}
	patreonOAuthEnabled, err := s.data.GetCfgBool(ConfigPatreonOauthEnabled, DefaultPatreonOauthEnabled)
	if err != nil {
		return err
	}

	if patreonOAuthEnabled {
		patreonClientID, err := s.data.GetCfgString(ConfigPatreonOauthClientID, DefaultPatreonOauthClientID)
		if err != nil {
			return err
		}

		patreonClientSecret, err := s.data.GetCfgString(ConfigPatreonOauthClientSecret, DefaultPatreonOauthClientSecret)
		if err != nil {
			return err
		}

		patreonOAuthConfig = &oauth2.Config{
			RedirectURL:  baseUrl + "/user/callback/patreon",
			ClientID:     patreonClientID,
			ClientSecret: patreonClientSecret,
			Scopes:       []string{"identity", "identity[email]"},
			Endpoint:     patreonEndpoint,
		}
	}
	return nil
}

// Removes the AuthType LOCAL AuthMethod from the currently logged in user
func (s *Server) handlerLocalAuthRemove(w http.ResponseWriter, r *http.Request) {
	s.l.Debug("local remove")

	user := s.getSessionUser(w, r)

	auth, err := user.GetAuthMethod(common.AUTH_LOCAL)

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

	user, err = s.RemoveAuthMethodFromUser(auth, user)

	if err != nil {
		s.l.Info("Could not remove password from user. %s", err.Error())
		http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
		return
	}

	err = s.data.UpdateUser(user)
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
	if _, err := user.GetAuthMethod(common.AUTH_TWITCH); err == nil {
		err = s.login(user, common.AUTH_TWITCH, w, r)
		if err != nil {
			s.l.Info("Could not login user %s", user.Name)
			http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
			return
		}
	} else if _, err := user.GetAuthMethod(common.AUTH_DISCORD); err == nil {
		err = s.login(user, common.AUTH_DISCORD, w, r)
		if err != nil {
			s.l.Info("Could not login user %s", user.Name)
			http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
			return
		}
	} else if _, err := user.GetAuthMethod(common.AUTH_PATREON); err == nil {
		err = s.login(user, common.AUTH_PATREON, w, r)
		if err != nil {
			s.l.Info("Could not login user %s", user.Name)
			http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
			return
		}
	}
	http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
	return
}

// Creates an OAuth request to log the user in using the TWITCH OAuth
func (s *Server) handlerTwitchOAuthLogin(w http.ResponseWriter, r *http.Request) {
	// TODO that might cause impersonation attacks (i.e. using the token of an other user)

	// Generate a new state string for each login attempt and store it in the state list
	oauthStateString := "login_" + getCryptRandKey(32)
	openStates = append(openStates, oauthStateString)

	// Handle the Oauth redirect
	url := twitchOAuthConfig.AuthCodeURL(oauthStateString)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)

	s.l.Debug("twitch login")
}

// Creates an OAuth request to sign up the user using the TWITCH OAuth
// A new user is created and a new AuthMethod struct is created and associated
func (s *Server) handlerTwitchOAuthSignup(w http.ResponseWriter, r *http.Request) {
	// TODO that might cause impersonation attacks (i.e. using the token of an other user)

	// Generate a new state string for each login attempt and store it in the state list
	oauthStateString := "signup_" + getCryptRandKey(32)
	openStates = append(openStates, oauthStateString)

	// Handle the Oauth redirect
	url := twitchOAuthConfig.AuthCodeURL(oauthStateString)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)

	s.l.Debug("twitch sign up")
}

// Creates an OAuth request to add a new Twitch AuthMethod to the currently logged in user
func (s *Server) handlerTwitchOAuthAdd(w http.ResponseWriter, r *http.Request) {
	// TODO that might cause impersonation attacks (i.e. using the token of an other user)

	// Generate a new state string for each login attempt and store it in the state list
	oauthStateString := "add_" + getCryptRandKey(32)
	openStates = append(openStates, oauthStateString)

	// Handle the Oauth redirect
	url := twitchOAuthConfig.AuthCodeURL(oauthStateString)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)

	s.l.Debug("twitch add")
}

// Removes the Twitch AuthMethod from the currently logged in user
func (s *Server) handlerTwitchOAuthRemove(w http.ResponseWriter, r *http.Request) {

	user := s.getSessionUser(w, r)

	auth, err := user.GetAuthMethod(common.AUTH_TWITCH)

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

	user, err = s.RemoveAuthMethodFromUser(auth, user)

	if err != nil {
		s.l.Info("Could not remove Twitch Oauth from user. %s", err.Error())
		http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
		return
	}

	err = s.data.UpdateUser(user)
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
	if _, err := user.GetAuthMethod(common.AUTH_LOCAL); err == nil {
		err = s.login(user, common.AUTH_LOCAL, w, r)
		if err != nil {
			s.l.Info("Could not login user %s", user.Name)
			http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
			return
		}
	} else if _, err := user.GetAuthMethod(common.AUTH_DISCORD); err == nil {
		err = s.login(user, common.AUTH_DISCORD, w, r)
		if err != nil {
			s.l.Info("Could not login user %s", user.Name)
			http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
			return
		}
	} else if _, err := user.GetAuthMethod(common.AUTH_PATREON); err == nil {
		err = s.login(user, common.AUTH_PATREON, w, r)
		if err != nil {
			s.l.Info("Could not login user %s", user.Name)
			http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
			return
		}
	}

	s.l.Debug("twitch remove")

	http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
	return
}

// This function handles all Twitch Callbacks (add/signup/login)
func (s *Server) handlerTwitchOAuthCallback(w http.ResponseWriter, r *http.Request) {
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
		auth := &common.AuthMethod{
			Type:         common.AUTH_TWITCH,
			ExtId:        data["data"][0]["id"].(string),
			AuthToken:    token.AccessToken,
			RefreshToken: token.RefreshToken,
			RefreshDate:  token.Expiry,
		}

		// check if Twitch Auth is already used
		if !s.data.CheckOauthUsage(auth.ExtId, auth.Type) {
			// Create a new user
			newUser := &common.User{
				Name:                data["data"][0]["display_name"].(string),
				Email:               data["data"][0]["email"].(string),
				NotifyCycleEnd:      false,
				NotifyVoteSelection: false,
			}

			// add this new server to the database
			newUser.Id, err = s.data.AddUser(newUser)

			if err != nil {
				s.l.Info(err.Error())
				http.Redirect(w, r, "/user/login", http.StatusTemporaryRedirect)
				return
			}

			// add the authmethod to the user
			newUser, err = s.AddAuthMethodToUser(auth, newUser)

			if err != nil {
				s.l.Info(err.Error())
				http.Redirect(w, r, "/user/login", http.StatusTemporaryRedirect)
				return
			}

			// update the user in the DB with the user having the AuthMethod associated
			err = s.data.UpdateUser(newUser)

			if err != nil {
				s.l.Info(err.Error())
				http.Redirect(w, r, "/user/login", http.StatusTemporaryRedirect)
				return
			}

			s.l.Debug("logging in %v", newUser.Name)
			s.login(newUser, common.AUTH_TWITCH, w, r)
			http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		} else {
			s.l.Debug("AuthMethod already used")
			http.Redirect(w, r, "/user/new", http.StatusTemporaryRedirect)
		}
	} else if strings.HasPrefix(state, "login_") {
		// Handle Twitch Login
		user, err := s.data.UserTwitchLogin(data["data"][0]["id"].(string))
		if err != nil {
			s.l.Info(err.Error())
			http.Redirect(w, r, "/user/login", http.StatusTemporaryRedirect)
			return
		}
		s.l.Debug("logging in %v", user.Name)
		s.login(user, common.AUTH_TWITCH, w, r)
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
	} else if strings.HasPrefix(state, "add_") {
		// Handle adding a Twitch AuthMethod to the logged in user

		// get the current user
		user := s.getSessionUser(w, r)

		auth := &common.AuthMethod{
			Type:         common.AUTH_TWITCH,
			ExtId:        data["data"][0]["id"].(string),
			AuthToken:    token.AccessToken,
			RefreshToken: token.RefreshToken,
			RefreshDate:  token.Expiry,
		}

		// check if this oauth is already used
		if !s.data.CheckOauthUsage(auth.ExtId, auth.Type) {
			// check if the user already has an other Twitch OAuth connected
			_, err = user.GetAuthMethod(auth.Type)
			if err != nil {
				_, err = s.AddAuthMethodToUser(auth, user)

				if err != nil {
					s.l.Info(err.Error())
					http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
					return
				}

				err = s.data.UpdateUser(user)

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
			http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
			return
		}
		http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
		return
	}
	return
}

// Creates an OAuth query to log a user in using Discord OAuth
func (s *Server) handlerDiscordOAuthLogin(w http.ResponseWriter, r *http.Request) {
	// TODO that might cause impersonation attacks (i.e. using the token of an other user)

	// Generate a new state string for each login attempt and store it in the state list
	oauthStateString := "login_" + getCryptRandKey(32)
	openStates = append(openStates, oauthStateString)

	// Handle the Oauth redirect
	url := discordOAuthConfig.AuthCodeURL(oauthStateString)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)

	s.l.Debug("discord login")
}

// Creates an OAuth query to sign up a user using Discord OAuth
func (s *Server) handlerDiscordOAuthSignup(w http.ResponseWriter, r *http.Request) {
	// TODO that might cause impersonation attacks (i.e. using the token of an other user)

	// Generate a new state string for each login attempt and store it in the state list
	oauthStateString := "signup_" + getCryptRandKey(32)
	openStates = append(openStates, oauthStateString)

	// Handle the Oauth redirect
	url := discordOAuthConfig.AuthCodeURL(oauthStateString)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)

	s.l.Debug("discord signup")
}

// Creates an OAuth query to add an Discord AuthMethod to the currently logged in user
func (s *Server) handlerDiscordOAuthAdd(w http.ResponseWriter, r *http.Request) {
	// TODO that might cause impersonation attacks (i.e. using the token of an other user)

	// Generate a new state string for each login attempt and store it in the state list
	oauthStateString := "add_" + getCryptRandKey(32)
	openStates = append(openStates, oauthStateString)

	// Handle the Oauth redirect
	url := discordOAuthConfig.AuthCodeURL(oauthStateString)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)

	s.l.Debug("discord add")
}

// Removes the Discord AuthMethod from the currently logged in user
func (s *Server) handlerDiscordOAuthRemove(w http.ResponseWriter, r *http.Request) {

	user := s.getSessionUser(w, r)

	auth, err := user.GetAuthMethod(common.AUTH_DISCORD)

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

	user, err = s.RemoveAuthMethodFromUser(auth, user)

	if err != nil {
		s.l.Info("Could not remove Discord Oauth from user. %s", err.Error())
		http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
		return
	}

	err = s.data.UpdateUser(user)
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
	if _, err := user.GetAuthMethod(common.AUTH_TWITCH); err == nil {
		err = s.login(user, common.AUTH_TWITCH, w, r)
		if err != nil {
			s.l.Info("Could not login user %s", user.Name)
			http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
			return
		}
	} else if _, err := user.GetAuthMethod(common.AUTH_LOCAL); err == nil {
		err = s.login(user, common.AUTH_LOCAL, w, r)
		if err != nil {
			s.l.Info("Could not login user %s", user.Name)
			http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
			return
		}
	} else if _, err := user.GetAuthMethod(common.AUTH_PATREON); err == nil {
		err = s.login(user, common.AUTH_PATREON, w, r)
		if err != nil {
			s.l.Info("Could not login user %s", user.Name)
			http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
			return
		}
	}

	s.l.Debug("discord remove")

	http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
	return
}

// Handler for the Discord OAuth Callbacks (add/signup/login)
func (s *Server) handlerDiscordOAuthCallback(w http.ResponseWriter, r *http.Request) {
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

		auth := &common.AuthMethod{
			Type:         common.AUTH_DISCORD,
			ExtId:        data["id"].(string),
			AuthToken:    token.AccessToken,
			RefreshToken: token.RefreshToken,
			RefreshDate:  token.Expiry,
		}

		if !s.data.CheckOauthUsage(auth.ExtId, auth.Type) {
			newUser := &common.User{
				Name:                data["username"].(string),
				Email:               data["email"].(string),
				NotifyCycleEnd:      false,
				NotifyVoteSelection: false,
			}

			newUser.Id, err = s.data.AddUser(newUser)

			if err != nil {
				s.l.Info(err.Error())
				http.Redirect(w, r, "/user/login", http.StatusTemporaryRedirect)
				return
			}

			newUser, err = s.AddAuthMethodToUser(auth, newUser)

			if err != nil {
				s.l.Info(err.Error())
				http.Redirect(w, r, "/user/login", http.StatusTemporaryRedirect)
				return
			}

			err = s.data.UpdateUser(newUser)

			if err != nil {
				s.l.Info(err.Error())
				http.Redirect(w, r, "/user/login", http.StatusTemporaryRedirect)
				return
			}

			s.l.Debug("logging in %v", newUser.Name)
			s.login(newUser, common.AUTH_DISCORD, w, r)
			http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		} else {
			s.l.Debug("AuthMethod already used")
			http.Redirect(w, r, "/user/new", http.StatusTemporaryRedirect)
		}
	} else if strings.HasPrefix(state, "login_") {
		user, err := s.data.UserDiscordLogin(data["id"].(string))
		if err != nil {
			s.l.Info(err.Error())
			http.Redirect(w, r, "/user/login", http.StatusTemporaryRedirect)
			return
		}
		s.l.Debug("logging in %v", user.Name)
		s.login(user, common.AUTH_DISCORD, w, r)
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
	} else if strings.HasPrefix(state, "add_") {
		user := s.getSessionUser(w, r)

		auth := &common.AuthMethod{
			Type:         common.AUTH_DISCORD,
			ExtId:        data["id"].(string),
			AuthToken:    token.AccessToken,
			RefreshToken: token.RefreshToken,
			RefreshDate:  token.Expiry,
		}

		if !s.data.CheckOauthUsage(auth.ExtId, auth.Type) {
			_, err = user.GetAuthMethod(auth.Type)
			if err != nil {
				_, err = s.AddAuthMethodToUser(auth, user)

				if err != nil {
					s.l.Info(err.Error())
					http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
					return
				}

				err = s.data.UpdateUser(user)

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
			http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
			return
		}
		http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
		return
	}
	return
}

func (s *Server) handlerPatreonOAuthLogin(w http.ResponseWriter, r *http.Request) {
	// TODO that might cause impersonation attacks (i.e. using the token of an other user)

	// Generate a new state string for each login attempt and store it in the state list
	oauthStateString := "login_" + getCryptRandKey(32)
	openStates = append(openStates, oauthStateString)

	// Handle the Oauth redirect
	url := patreonOAuthConfig.AuthCodeURL(oauthStateString)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)

	s.l.Debug("patreon login")
}

func (s *Server) handlerPatreonOAuthSignup(w http.ResponseWriter, r *http.Request) {
	// TODO that might cause impersonation attacks (i.e. using the token of an other user)

	// Generate a new state string for each login attempt and store it in the state list
	oauthStateString := "signup_" + getCryptRandKey(32)
	openStates = append(openStates, oauthStateString)

	// Handle the Oauth redirect
	url := patreonOAuthConfig.AuthCodeURL(oauthStateString)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)

	s.l.Debug("patreon signup")
}

func (s *Server) handlerPatreonOAuthAdd(w http.ResponseWriter, r *http.Request) {
	// TODO that might cause impersonation attacks (i.e. using the token of an other user)

	// Generate a new state string for each login attempt and store it in the state list
	oauthStateString := "add_" + getCryptRandKey(32)
	openStates = append(openStates, oauthStateString)

	// Handle the Oauth redirect
	url := patreonOAuthConfig.AuthCodeURL(oauthStateString)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)

	s.l.Debug("patreon add")
}

func (s *Server) handlerPatreonOAuthRemove(w http.ResponseWriter, r *http.Request) {

	user := s.getSessionUser(w, r)

	auth, err := user.GetAuthMethod(common.AUTH_PATREON)

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

	user, err = s.RemoveAuthMethodFromUser(auth, user)

	if err != nil {
		s.l.Info("Could not remove Patreon Oauth from user. %s", err.Error())
		http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
		return
	}

	err = s.data.UpdateUser(user)
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

	if _, err := user.GetAuthMethod(common.AUTH_TWITCH); err == nil {
		err = s.login(user, common.AUTH_TWITCH, w, r)
		if err != nil {
			s.l.Info("Could not login user %s", user.Name)
			http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
			return
		}
	} else if _, err := user.GetAuthMethod(common.AUTH_DISCORD); err == nil {
		err = s.login(user, common.AUTH_DISCORD, w, r)
		if err != nil {
			s.l.Info("Could not login user %s", user.Name)
			http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
			return
		}
	} else if _, err := user.GetAuthMethod(common.AUTH_LOCAL); err == nil {
		err = s.login(user, common.AUTH_LOCAL, w, r)
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

func (s *Server) handlerPatreonOAuthCallback(w http.ResponseWriter, r *http.Request) {
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

		auth := &common.AuthMethod{
			Type:         common.AUTH_PATREON,
			ExtId:        data["id"].(string),
			AuthToken:    token.AccessToken,
			RefreshToken: token.RefreshToken,
			RefreshDate:  token.Expiry,
		}

		if !s.data.CheckOauthUsage(auth.ExtId, auth.Type) {

			newUser := &common.User{
				Name:                data["attributes"].(map[string]interface{})["full_name"].(string),
				Email:               data["attributes"].(map[string]interface{})["email"].(string),
				NotifyCycleEnd:      false,
				NotifyVoteSelection: false,
			}

			newUser.Id, err = s.data.AddUser(newUser)

			if err != nil {
				s.l.Info(err.Error())
				http.Redirect(w, r, "/user/login", http.StatusTemporaryRedirect)
				return
			}

			newUser, err = s.AddAuthMethodToUser(auth, newUser)

			if err != nil {
				s.l.Info(err.Error())
				http.Redirect(w, r, "/user/login", http.StatusTemporaryRedirect)
				return
			}

			err = s.data.UpdateUser(newUser)

			if err != nil {
				s.l.Info(err.Error())
				http.Redirect(w, r, "/user/login", http.StatusTemporaryRedirect)
				return
			}

			s.l.Debug("logging in %v", newUser.Name)
			s.login(newUser, common.AUTH_PATREON, w, r)
			http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		} else {
			s.l.Info("AuthMethod already used")
			http.Redirect(w, r, "/user/new", http.StatusTemporaryRedirect)
		}
	} else if strings.HasPrefix(state, "login_") {
		user, err := s.data.UserPatreonLogin(data["id"].(string))
		if err != nil {
			s.l.Info(err.Error())
			http.Redirect(w, r, "/user/login", http.StatusTemporaryRedirect)
			return
		}
		s.l.Debug("logging in %v", user.Name)
		s.login(user, common.AUTH_PATREON, w, r)
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
	} else if strings.HasPrefix(state, "add_") {
		user := s.getSessionUser(w, r)

		auth := &common.AuthMethod{
			Type:         common.AUTH_PATREON,
			ExtId:        data["id"].(string),
			AuthToken:    token.AccessToken,
			RefreshToken: token.RefreshToken,
			RefreshDate:  token.Expiry,
		}

		if !s.data.CheckOauthUsage(auth.ExtId, auth.Type) {
			_, err = user.GetAuthMethod(auth.Type)
			if err != nil {
				_, err = s.AddAuthMethodToUser(auth, user)

				if err != nil {
					s.l.Info(err.Error())
					http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
					return
				}

				err = s.data.UpdateUser(user)

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
			http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
			return
		}
		http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
		return
	}
	return
}
