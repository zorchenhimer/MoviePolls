package moviepoll

import (
	"net/http"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/twitch"
)

var twitchOAuthConfig = &oauth2.Config{}
var discordOAuthConfig = &oauth2.Config{}
var patreonOAuthConfig = &oauth2.Config{}
var oauthStateString string
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
			Scopes:       []string{},
			Endpoint:     twitch.Endpoint,
		}
	}
	// TODO cry in a corner and figure out how to do this stuff for discord and patreon

	return nil
}

func (s *Server) handlerTwitchOAuth(w http.ResponseWriter, r *http.Request) {
	// TODO that might cause impersonation attacks (i.e. using the token of an other user)
	oauthStateString := getCryptRandKey(32)
	openStates = append(openStates, oauthStateString)
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
		s.l.Info("Invalid OAuth state: '%s'", state)
		http.Redirect(w, r, "/user/login", http.StatusTemporaryRedirect)
		return
	}

	code := r.FormValue("code")
	token, err := twitchOAuthConfig.Exchange(oauth2.NoContext, code)
	if err != nil {
		s.l.Info("Code exchange failed with '%s'", err)
		http.Redirect(w, r, "/user/login", http.StatusTemporaryRedirect)
		return
	}
	s.l.Debug("Token: %s", token)
	http.Redirect(w, r, "/user/login", http.StatusTemporaryRedirect)
	return
}

func (s *Server) handlerDiscordOAuth() {

}

func (s *Server) handlerPatreonOAuth() {

}
