package moviepoll

import (
	"encoding/json"
	"fmt"
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

	twitchOauthEnabled, err := s.data.GetCfgBool(ConfigTwitchOauthEnabled, DefaultTwitchOauthEnabled)
	if err != nil {
		return err
	}

	discordOAuthEnabled, err := s.data.GetCfgBool(ConfigDiscordOauthEnabled, DefaultDiscordOauthEnabled)
	if err != nil {
		return err
	}

	patreonOAuthEnabled, err := s.data.GetCfgBool(ConfigPatreonOauthEnabled, DefaultPatreonOauthEnabled)
	if err != nil {
		return err
	}

	baseUrl, err := s.data.GetCfgString(ConfigHostAddress, "")
	if err != nil {
		return err
	}

	if twitchOauthEnabled || discordOAuthEnabled || patreonOAuthEnabled {
		if baseUrl == "" {
			return fmt.Errorf("Config Value for HostAddress cannot be empty to use OAuth")
		}
	}

	if twitchOauthEnabled {
		twitchClientID, err := s.data.GetCfgString(ConfigTwitchOauthClientID, DefaultTwitchOauthClientID)
		if err != nil {
			return err
		}
		if twitchClientID == "" {
			return fmt.Errorf("Config Value for TwitchOauthClientID cannot be empty to use OAuth")
		}

		twitchClientSecret, err := s.data.GetCfgString(ConfigTwitchOauthClientSecret, DefaultTwitchOauthClientSecret)
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
		discordClientID, err := s.data.GetCfgString(ConfigDiscordOauthClientID, DefaultDiscordOauthClientID)
		if err != nil {
			return err
		}

		if discordClientID == "" {
			return fmt.Errorf("Config Value for DiscordOauthClientID cannot be empty to use OAuth")
		}

		discordClientSecret, err := s.data.GetCfgString(ConfigDiscordOauthClientSecret, DefaultDiscordOauthClientSecret)
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
		patreonClientID, err := s.data.GetCfgString(ConfigPatreonOauthClientID, DefaultPatreonOauthClientID)
		if err != nil {
			return err
		}

		if patreonClientID == "" {
			return fmt.Errorf("Config Value for PatreonOauthClientSecret cannot be empty to use OAuth")
		}

		patreonClientSecret, err := s.data.GetCfgString(ConfigPatreonOauthClientSecret, DefaultPatreonOauthClientSecret)
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
