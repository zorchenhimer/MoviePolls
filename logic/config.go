package logic

import (
	"errors"

	"github.com/zorchenhimer/MoviePolls/database"
	//"github.com/zorchenhimer/MoviePolls/common"
)

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

func (b *backend) GetFormFillEnabled() (bool, error) {
	val, err := b.data.GetCfgBool(ConfigFormfillEnabled, DefaultFormfillEnabled)
	if errors.Is(err, database.ErrNoValue) {
		err = b.data.SetCfgBool(ConfigFormfillEnabled, DefaultFormfillEnabled)
		if err != nil {
			b.l.Error("Unable to set default value for %s: %v", ConfigFormfillEnabled, err)
		}
		return val, nil
	}

	return val, err
}

func (b *backend) GetMaxRemarksLength() (int, error) {
	val, err := b.data.GetCfgInt(ConfigMaxRemarksLength, DefaultMaxRemarksLength)
	if errors.Is(err, database.ErrNoValue) {
		err = b.data.SetCfgInt(ConfigMaxRemarksLength, DefaultMaxRemarksLength)
		if err != nil {
			b.l.Error("Unable to set default value for %s: %v", ConfigMaxRemarksLength, err)
		}
		return val, nil
	}

	return val, err
}

func (b *backend) GetMaxLinkLength() (int, error) {
	val, err := b.data.GetCfgInt(ConfigMaxLinkLength, DefaultMaxLinkLength)
	if errors.Is(err, database.ErrNoValue) {
		err = b.data.SetCfgInt(ConfigMaxLinkLength, DefaultMaxLinkLength)
		if err != nil {
			b.l.Error("Unable to set default value for %s: %v", ConfigMaxLinkLength, err)
		}
		return val, nil
	}

	return val, err
}



