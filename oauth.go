package moviepoll

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

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

func (s *Server) handlerTwitchOAuth(w http.ResponseWriter, r *http.Request) {
	// TODO that might cause impersonation attacks (i.e. using the token of an other user)

	// Generate a new state string for each login attempt and store it in the state list
	oauthStateString := getCryptRandKey(32)
	openStates = append(openStates, oauthStateString)

	// Handle the Oauth redirect
	url := twitchOAuthConfig.AuthCodeURL(oauthStateString)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
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

	auth := &common.AuthMethod{
		Type:         common.AUTH_TWITCH,
		AuthToken:    token.AccessToken,
		RefreshToken: token.RefreshToken,
		RefreshDate:  token.Expiry,
	}

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

	http.Redirect(w, r, "/user/login", http.StatusTemporaryRedirect)
	return
}

func (s *Server) handlerDiscordOAuth() {

}

func (s *Server) handlerPatreonOAuth() {

}
