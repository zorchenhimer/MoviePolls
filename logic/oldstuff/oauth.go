package logic

import (
	"fmt"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/twitch"

	mpm "github.com/zorchenhimer/MoviePolls/models"
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
func (l *LogicData) initOauth() error {

	twitchOauthEnabled, err := l.Database.GetCfgBool(ConfigTwitchOauthEnabled, DefaultTwitchOauthEnabled)
	if err != nil {
		return err
	}

	discordOAuthEnabled, err := l.Database.GetCfgBool(ConfigDiscordOauthEnabled, DefaultDiscordOauthEnabled)
	if err != nil {
		return err
	}

	patreonOAuthEnabled, err := l.Database.GetCfgBool(ConfigPatreonOauthEnabled, DefaultPatreonOauthEnabled)
	if err != nil {
		return err
	}

	baseUrl, err := l.Database.GetCfgString(ConfigHostAddress, "")
	if err != nil {
		return err
	}

	if twitchOauthEnabled || discordOAuthEnabled || patreonOAuthEnabled {
		if baseUrl == "" {
			return fmt.Errorf("Config Value for HostAddress cannot be empty to use OAuth")
		}
	}

	if twitchOauthEnabled {
		twitchClientID, err := l.Database.GetCfgString(ConfigTwitchOauthClientID, DefaultTwitchOauthClientID)
		if err != nil {
			return err
		}
		if twitchClientID == "" {
			return fmt.Errorf("Config Value for TwitchOauthClientID cannot be empty to use OAuth")
		}

		twitchClientSecret, err := l.Database.GetCfgString(ConfigTwitchOauthClientSecret, DefaultTwitchOauthClientSecret)
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
		discordClientID, err := l.Database.GetCfgString(ConfigDiscordOauthClientID, DefaultDiscordOauthClientID)
		if err != nil {
			return err
		}

		if discordClientID == "" {
			return fmt.Errorf("Config Value for DiscordOauthClientID cannot be empty to use OAuth")
		}

		discordClientSecret, err := l.Database.GetCfgString(ConfigDiscordOauthClientSecret, DefaultDiscordOauthClientSecret)
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
		patreonClientID, err := l.Database.GetCfgString(ConfigPatreonOauthClientID, DefaultPatreonOauthClientID)
		if err != nil {
			return err
		}

		if patreonClientID == "" {
			return fmt.Errorf("Config Value for PatreonOauthClientSecret cannot be empty to use OAuth")
		}

		patreonClientSecret, err := l.Database.GetCfgString(ConfigPatreonOauthClientSecret, DefaultPatreonOauthClientSecret)
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

func (l *LogicData) discordOauth(action string, user *mpm.User) string {
	switch action {
	case "login":
		// Generate a new state string for each login attempt and store it in the state list
		oauthStateString := "login_" + l.getCryptRandKey(32)
		openStates = append(openStates, oauthStateString)

		// Handle the Oauth redirect
		url := discordOAuthConfig.AuthCodeURL(oauthStateString)
		l.Logger.Debug("discord login")
		return url

	case "signup":
		// Generate a new state string for each login attempt and store it in the state list
		oauthStateString := "signup_" + l.getCryptRandKey(32)
		openStates = append(openStates, oauthStateString)

		// Handle the Oauth redirect
		url := discordOAuthConfig.AuthCodeURL(oauthStateString)
		l.Logger.Debug("discord signup")
		return url

	case "add":
		// Generate a new state string for each login attempt and store it in the state list
		oauthStateString := "add_" + l.getCryptRandKey(32)
		openStates = append(openStates, oauthStateString)

		// Handle the Oauth redirect
		url := discordOAuthConfig.AuthCodeURL(oauthStateString)
		l.Logger.Debug("discord add")
		return url

	case "remove":
		auth, err := user.GetAuthMethod(mpm.AUTH_DISCORD)

		if err != nil {
			l.Logger.Info("User %s does not have Discord Oauth associated with him", user.Name)
			return "/user"
		}

		if len(user.AuthMethods) == 1 {
			l.Logger.Info("User %v only has Discord Oauth associated with him", user.Name)
			return "/user"
		}

		user, err = l.RemoveAuthMethodFromUser(auth, user)

		if err != nil {
			l.Logger.Info("Could not remove Discord Oauth from user. %s", err.Error())
			return "/user"
		}

		err = l.Database.UpdateUser(user)
		if err != nil {
			l.Logger.Info("Could not update user %s", user.Name)
			return "/user"
		}

		// Log the user out to ensure he is logged in with an existing AuthMethod
		err = l.logout(user)
		if err != nil {
			l.Logger.Info("Could not logout user %s", user.Name)
			return "/user"
		}

		// Try to log the user back in
		if _, err := user.GetAuthMethod(mpm.AUTH_TWITCH); err == nil {
			err = l.login(user, mpm.AUTH_TWITCH)
			if err != nil {
				l.Logger.Info("Could not login user %s", user.Name)
				return "/user"
			}
		} else if _, err := user.GetAuthMethod(mpm.AUTH_LOCAL); err == nil {
			err = l.login(user, mpm.AUTH_LOCAL)
			if err != nil {
				l.Logger.Info("Could not login user %s", user.Name)
				return "/user"
			}
		} else if _, err := user.GetAuthMethod(mpm.AUTH_PATREON); err == nil {
			err = l.login(user, mpm.AUTH_PATREON)
			if err != nil {
				l.Logger.Info("Could not login user %s", user.Name)
				return "/user"
			}
		}

		l.Logger.Debug("discord remove")

		return "/user"
	}

}

func (l *LogicData) discordOauthCallback() {

}

func (l *LogicData) patreonOauth() {

}

func (l *LogicData) patreonOauthCallback() {

}

func (l *LogicData) twitchOauth() {

}

func (l *LogicData) twitchOauthCallback() {

}
