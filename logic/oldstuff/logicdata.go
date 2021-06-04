package logic

import (
	mpd "github.com/zorchenhimer/MoviePolls/data"
	mpm "github.com/zorchenhimer/MoviePolls/models"
)

const SessionName string = "moviepoll-session"

// defaults
const (
	DefaultMaxUserVotes           int    = 5
	DefaultEntriesRequireApproval bool   = false
	DefaultFormfillEnabled        bool   = true
	DefaultVotingEnabled          bool   = false
	DefaultJikanEnabled           bool   = false
	DefaultJikanBannedTypes       string = "TV,music"
	DefaultJikanMaxEpisodes       int    = 1
	DefaultTmdbEnabled            bool   = false
	DefaultTmdbToken              string = ""
	DefaultMaxNameLength          int    = 100
	DefaultMinNameLength          int    = 4
	DefaultUnlimitedVotes         bool   = false

	DefaultMaxTitleLength       int = 100
	DefaultMaxDescriptionLength int = 1000
	DefaultMaxLinkLength        int = 500 // length of all links combined
	DefaultMaxRemarksLength     int = 200

	DefaultMaxMultEpLength int = 120 // length of multiple episode entries in minutes

	DefaultLocalSignupEnabled        bool   = true
	DefaultTwitchOauthEnabled        bool   = false
	DefaultTwitchOauthSignupEnabled  bool   = false
	DefaultTwitchOauthClientID       string = ""
	DefaultTwitchOauthClientSecret   string = ""
	DefaultDiscordOauthEnabled       bool   = false
	DefaultDiscordOauthSignupEnabled bool   = false
	DefaultDiscordOauthClientID      string = ""
	DefaultDiscordOauthClientSecret  string = ""
	DefaultPatreonOauthEnabled       bool   = false
	DefaultPatreonOauthSignupEnabled bool   = false
	DefaultPatreonOauthClientID      string = ""
	DefaultPatreonOauthClientSecret  string = ""
)

// configuration keys
const (
	ConfigVotingEnabled          string = "VotingEnabled"
	ConfigMaxUserVotes           string = "MaxUserVotes"
	ConfigEntriesRequireApproval string = "EntriesRequireApproval"
	ConfigFormfillEnabled        string = "FormfillEnabled"
	ConfigTmdbToken              string = "TmdbToken"
	ConfigJikanEnabled           string = "JikanEnabled"
	ConfigJikanBannedTypes       string = "JikanBannedTypes"
	ConfigJikanMaxEpisodes       string = "JikanMaxEpisodes"
	ConfigTmdbEnabled            string = "TmdbEnabled"
	ConfigMaxNameLength          string = "MaxNameLength"
	ConfigMinNameLength          string = "MinNameLength"
	ConfigNoticeBanner           string = "NoticeBanner"
	ConfigHostAddress            string = "HostAddress"
	ConfigUnlimitedVotes         string = "UnlimitedVotes"

	ConfigMaxTitleLength       string = "MaxTitleLength"
	ConfigMaxDescriptionLength string = "MaxDescriptionLength"
	ConfigMaxLinkLength        string = "MaxLinkLength"
	ConfigMaxRemarksLength     string = "MaxRemarksLength"

	ConfigMaxMultEpLength string = "ConfigMaxMultEpLength"

	ConfigLocalSignupEnabled        string = "LocalSignupEnabled"
	ConfigTwitchOauthEnabled        string = "TwitchOauthEnabled"
	ConfigTwitchOauthSignupEnabled  string = "TwitchOauthSignupEnabled"
	ConfigTwitchOauthClientID       string = "TwitchOauthClientID"
	ConfigTwitchOauthClientSecret   string = "TwitchOauthSecret"
	ConfigDiscordOauthEnabled       string = "DiscordOauthEnabled"
	ConfigDiscordOauthSignupEnabled string = "DiscordOauthSignupEnabled"
	ConfigDiscordOauthClientID      string = "DiscordOauthClientID"
	ConfigDiscordOauthClientSecret  string = "DiscordOauthClientSecret"
	ConfigPatreonOauthEnabled       string = "PatreonOauthEnabled"
	ConfigPatreonOauthSignupEnabled string = "PatreonOauthSignupEnabled"
	ConfigPatreonOauthClientID      string = "PatreonOauthClientID"
	ConfigPatreonOauthClientSecret  string = "PatreonOauthClientSecret"
)

var ReleaseVersion string

type LogicData struct {
	Database     mpd.DataConnector
	Logger       *mpm.Logger
	PasswordSalt string
	Debug        bool
}
