package moviepoll

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
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

func (s *Server) initOauth() error {
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
			RedirectURL:  "http://localhost:8090/user/login/twitch/callback",
			ClientID:     twitchClientID,
			ClientSecret: twitchClientSecret,
			Scopes:       []string{"user:read:email"},
			Endpoint:     twitch.Endpoint,
		}
	}
	// TODO cry in a corner and figure out how to do this stuff for discord and patreon

	return nil
}

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

func (s *Server) handlerTwitchOAuthSignup(w http.ResponseWriter, r *http.Request) {
	// TODO that might cause impersonation attacks (i.e. using the token of an other user)

	// Generate a new state string for each login attempt and store it in the state list
	oauthStateString := "signup_" + getCryptRandKey(32)
	openStates = append(openStates, oauthStateString)

	// Handle the Oauth redirect
	url := twitchOAuthConfig.AuthCodeURL(oauthStateString)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)

	s.l.Debug("twitch signup")
}

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

func (s *Server) handlerTwitchOAuthRemove(w http.ResponseWriter, r *http.Request) {
	s.l.Debug("twitch remove")

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

	http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
	return
}

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
		s.l.Error("Status Code is not 200, its %v", resp.Status)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		s.l.Error(err.Error())
	}

	var data map[string][]map[string]interface{}

	if err := json.Unmarshal(body, &data); err != nil {
		s.l.Error(err.Error())
		s.l.Debug("%v", data)
		http.Redirect(w, r, "/user/login", http.StatusTemporaryRedirect)
		return
	}

	if strings.HasPrefix(state, "signup_") {

		s.l.Debug("signup prefix")

		auth := &common.AuthMethod{
			Type:         common.AUTH_TWITCH,
			ExtId:        data["data"][0]["id"].(string),
			AuthToken:    token.AccessToken,
			RefreshToken: token.RefreshToken,
			RefreshDate:  token.Expiry,
		}

		if !s.data.CheckOauthUsage(auth.ExtId, auth.Type) {

			newUser := &common.User{
				Name:                data["data"][0]["display_name"].(string),
				Email:               data["data"][0]["email"].(string),
				NotifyCycleEnd:      false,
				NotifyVoteSelection: false,
			}

			newUser.Id, err = s.data.AddUser(newUser)

			if err != nil {
				s.l.Error(err.Error())
				http.Redirect(w, r, "/user/login", http.StatusTemporaryRedirect)
				return
			}

			newUser, err = s.AddAuthMethodToUser(auth, newUser)

			if err != nil {
				s.l.Error(err.Error())
				http.Redirect(w, r, "/user/login", http.StatusTemporaryRedirect)
				return
			}

			err = s.data.UpdateUser(newUser)

			if err != nil {
				s.l.Error(err.Error())
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
		s.l.Debug("login prefix")
		user, err := s.data.UserTwitchLogin(data["data"][0]["id"].(string))
		if err != nil {
			s.l.Error(err.Error())
			http.Redirect(w, r, "/user/login", http.StatusTemporaryRedirect)
			return
		}
		s.l.Debug("logging in %v", user.Name)
		s.login(user, common.AUTH_TWITCH, w, r)
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
	} else if strings.HasPrefix(state, "add_") {
		s.l.Debug("add prefix")

		user := s.getSessionUser(w, r)

		auth := &common.AuthMethod{
			Type:         common.AUTH_TWITCH,
			ExtId:        data["data"][0]["id"].(string),
			AuthToken:    token.AccessToken,
			RefreshToken: token.RefreshToken,
			RefreshDate:  token.Expiry,
		}

		if !s.data.CheckOauthUsage(auth.ExtId, auth.Type) {
			_, err = user.GetAuthMethod(auth.Type)
			if err != nil {
				_, err = s.AddAuthMethodToUser(auth, user)

				if err != nil {
					s.l.Error(err.Error())
					http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
					return
				}

				err = s.data.UpdateUser(user)

				if err != nil {
					s.l.Error(err.Error())
					http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
					return
				}
			} else {
				s.l.Error("User %s already has %s Oauth associated", user.Name, auth.Type)
				http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
				return
			}
		} else {
			s.l.Error("The provided Oauth login is already used")
			http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
			return
		}
		http.Redirect(w, r, "/user", http.StatusTemporaryRedirect)
		return
	}
	return
}

func (s *Server) handlerDiscordOAuth() {

}

func (s *Server) handlerPatreonOAuth() {

}
