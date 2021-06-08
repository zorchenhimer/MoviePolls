package logic

import (
	"errors"
	"strings"

	"github.com/zorchenhimer/MoviePolls/database"
	"github.com/zorchenhimer/MoviePolls/models"
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

func (b *backend) CheckMovieExists(title string) (bool, error) {
	return b.data.CheckMovieExists(title)
}

func (b *backend) GetJikanEnabled() (bool, error) {
	val, err := b.data.GetCfgBool(ConfigJikanEnabled, DefaultJikanEnabled)
	if errors.Is(err, database.ErrNoValue) {
		err = b.data.SetCfgBool(ConfigJikanEnabled, DefaultJikanEnabled)
		if err != nil {
			b.l.Error("Unable to set default value for %s: %v", ConfigJikanEnabled, err)
		}
		return val, nil
	}

	return val, err
}

func (b *backend) GetTmdbEnabled() (bool, error) {
	val, err := b.data.GetCfgBool(ConfigTmdbEnabled, DefaultTmdbEnabled)
	if errors.Is(err, database.ErrNoValue) {
		err = b.data.SetCfgBool(ConfigTmdbEnabled, DefaultTmdbEnabled)
		if err != nil {
			b.l.Error("Unable to set default value for %s: %v", ConfigTmdbEnabled, err)
		}
		return val, nil
	}

	return val, err
}

func (b *backend) GetTmdbToken() (string, error) {
	val, err := b.data.GetCfgString(ConfigTmdbToken, DefaultTmdbToken)
	if errors.Is(err, database.ErrNoValue) {
		err = b.data.SetCfgString(ConfigTmdbToken, DefaultTmdbToken)
		if err != nil {
			b.l.Error("Unable to set default value for %s: %v", ConfigTmdbToken, err)
		}
		return val, nil
	}
	return val, nil
}

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

func (b *backend) GetJikanBannedTypes() ([]string, error) {
	val, err := b.data.GetCfgString(ConfigJikanBannedTypes, DefaultJikanBannedTypes)
	ret := strings.Split(val, ",")
	if errors.Is(err, database.ErrNoValue) {
		err = b.data.SetCfgString(ConfigJikanBannedTypes, DefaultJikanBannedTypes)
		if err != nil {
			b.l.Error("Unable to set default value for %s: %v", ConfigJikanBannedTypes, err)
		}
		return strings.Split(DefaultJikanBannedTypes, ","), nil
	}
	return ret, nil
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

func (b *backend) GetJikanMaxEpisodes() (int, error) {
	val, err := b.data.GetCfgInt(ConfigJikanMaxEpisodes, DefaultJikanMaxEpisodes)
	if errors.Is(err, database.ErrNoValue) {
		err = b.data.SetCfgInt(ConfigJikanMaxEpisodes, DefaultJikanMaxEpisodes)
		if err != nil {
			b.l.Error("Unable to set default value for %s: %v", ConfigJikanMaxEpisodes, err)
		}
		return val, nil
	}
	return val, err
}

func (b *backend) GetMaxDuration() (int, error) {
	val, err := b.data.GetCfgInt(ConfigMaxMultEpLength, DefaultMaxMultEpLength)
	if errors.Is(err, database.ErrNoValue) {
		err = b.data.SetCfgInt(ConfigMaxMultEpLength, DefaultMaxMultEpLength)
		if err != nil {
			b.l.Error("Unable to set default value for %s: %v", ConfigMaxMultEpLength, err)
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

func (b *backend) GetMaxTitleLength() (int, error) {
	val, err := b.data.GetCfgInt(ConfigMaxTitleLength, DefaultMaxTitleLength)
	if errors.Is(err, database.ErrNoValue) {
		err = b.data.SetCfgInt(ConfigMaxTitleLength, DefaultMaxTitleLength)
		if err != nil {
			b.l.Error("Unable to set default value for %s: %v", ConfigMaxTitleLength, err)
		}
		return val, nil
	}

	return val, err
}

func (b *backend) GetMaxDescriptionLength() (int, error) {
	val, err := b.data.GetCfgInt(ConfigMaxDescriptionLength, DefaultMaxDescriptionLength)
	if errors.Is(err, database.ErrNoValue) {
		err = b.data.SetCfgInt(ConfigMaxDescriptionLength, DefaultMaxDescriptionLength)
		if err != nil {
			b.l.Error("Unable to set default value for %s: %v", ConfigMaxDescriptionLength, err)
		}
		return val, nil
	}

	return val, err
}

func (b *backend) AddMovieToDB(movie *models.Movie) (int, error) {
	return b.data.AddMovie(movie)
}
