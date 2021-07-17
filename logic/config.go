package logic

import (
	"errors"
	"fmt"
	"strings"

	"github.com/zorchenhimer/MoviePolls/database"
	"github.com/zorchenhimer/MoviePolls/models"
)

type ConfigValue struct {
	Default interface{}
	Value   interface{}
	Type    ConfigValueType
	Error   bool
	Section string
}

type ConfigValueType int

const (
	ConfigInt ConfigValueType = iota
	ConfigString
	ConfigStringPriv
	ConfigBool
	ConfigKey
)

var ConfigValues = map[string]ConfigValue{}
var ConfigSections = []string{}

const GeneralSettings string = "General Settings"
const ConfigHostAddress string = "HostAddress"
const ConfigNoticeBanner string = "NoticeBanner"

const FieldLimits string = "Inputfield limitation Settings"
const ConfigMinNameLength string = "MinNameLength"
const ConfigMaxNameLength string = "MaxNameLength"
const ConfigMaxTitleLength string = "MaxTitleLength"
const ConfigMaxDescriptionLength string = "MaxDescriptionLength"
const ConfigMaxLinkLength string = "MaxLinkLength"
const ConfigMaxRemarksLength string = "MaxRemarksLength"

const MovieInput string = "Movie input Settings"
const ConfigFormfillEnabled string = "FormfillEnabled"
const ConfigJikanEnabled string = "JikanEnabled"
const ConfigJikanBannedTypes string = "JikanBannedTypes"
const ConfigJikanMaxEpisodes string = "JikanMaxEpisodes"
const ConfigTmdbEnabled string = "TmdbEnabled"
const ConfigTmdbToken string = "TmdbToken"
const ConfigMaxMultEpLength = "MaxMultEpLength"

const Authentication string = "Authentication Settings"
const ConfigLocalSignupEnabled string = "LocalSignupEnabled"
const ConfigTwitchOauthEnabled string = "TwitchOauthEnabled"
const ConfigTwitchOauthSignupEnabled string = "TwitchOauthSignupEnabled"
const ConfigTwitchOauthClientID string = "TwitchOauthClientID"
const ConfigTwitchOauthClientSecret string = "TwitchOauthClientSecret"
const ConfigDiscordOauthEnabled string = "DiscordOauthEnabled"
const ConfigDiscordOauthSignupEnabled string = "DiscordOauthSignupEnabled"
const ConfigDiscordOauthClientID string = "DiscordOauthClientID"
const ConfigDiscordOauthClientSecret string = "DiscordOauthClientSecret"
const ConfigPatreonOauthEnabled string = "PatreonOauthEnabled"
const ConfigPatreonOauthSignupEnabled string = "PatreonOauthSignupEnabled"
const ConfigPatreonOauthClientID string = "PatreonOauthClientID"
const ConfigPatreonOauthClientSecret string = "PatreonOauthClientSecret"

const Administration string = "Administration Settings"
const ConfigMaxUserVotes string = "MaxUserVotes"
const ConfigVotingEnabled string = "VotingEnabled"
const ConfigEntriesRequireApproval string = "EntriesRequireApproval"
const ConfigUnlimitedVotes string = "UnlimitedVotes"

func (b *backend) setupConfig() {
	// General Settings
	ConfigSections = append(ConfigSections, GeneralSettings)
	ConfigValues[ConfigHostAddress] = ConfigValue{Section: GeneralSettings, Default: "localhost", Type: ConfigString}
	ConfigValues[ConfigNoticeBanner] = ConfigValue{Section: GeneralSettings, Default: "", Type: ConfigString}

	// Field Limits
	ConfigSections = append(ConfigSections, FieldLimits)
	ConfigValues[ConfigMinNameLength] = ConfigValue{Section: FieldLimits, Default: 4, Type: ConfigInt}
	ConfigValues[ConfigMaxNameLength] = ConfigValue{Section: FieldLimits, Default: 100, Type: ConfigInt}
	ConfigValues[ConfigMaxTitleLength] = ConfigValue{Section: FieldLimits, Default: 100, Type: ConfigInt}
	ConfigValues[ConfigMaxDescriptionLength] = ConfigValue{Section: FieldLimits, Default: 1000, Type: ConfigInt}
	ConfigValues[ConfigMaxLinkLength] = ConfigValue{Section: FieldLimits, Default: 500, Type: ConfigInt}
	ConfigValues[ConfigMaxRemarksLength] = ConfigValue{Section: FieldLimits, Default: 200, Type: ConfigInt}

	// Movie Input
	ConfigSections = append(ConfigSections, MovieInput)
	ConfigValues[ConfigFormfillEnabled] = ConfigValue{Section: MovieInput, Default: true, Type: ConfigBool}
	ConfigValues[ConfigJikanEnabled] = ConfigValue{Section: MovieInput, Default: false, Type: ConfigBool}
	ConfigValues[ConfigJikanBannedTypes] = ConfigValue{Section: MovieInput, Default: "TV,music", Type: ConfigString}
	ConfigValues[ConfigJikanMaxEpisodes] = ConfigValue{Section: MovieInput, Default: 1, Type: ConfigInt}
	ConfigValues[ConfigTmdbEnabled] = ConfigValue{Section: MovieInput, Default: false, Type: ConfigBool}
	ConfigValues[ConfigTmdbToken] = ConfigValue{Section: MovieInput, Default: "", Type: ConfigStringPriv}
	ConfigValues[ConfigMaxMultEpLength] = ConfigValue{Section: MovieInput, Default: 120, Type: ConfigInt}

	// Authentication
	ConfigSections = append(ConfigSections, Authentication)
	ConfigValues[ConfigLocalSignupEnabled] = ConfigValue{Section: Authentication, Default: true, Type: ConfigBool}
	ConfigValues[ConfigTwitchOauthEnabled] = ConfigValue{Section: Authentication, Default: false, Type: ConfigBool}
	ConfigValues[ConfigTwitchOauthSignupEnabled] = ConfigValue{Section: Authentication, Default: false, Type: ConfigBool}
	ConfigValues[ConfigTwitchOauthClientID] = ConfigValue{Section: Authentication, Default: "", Type: ConfigStringPriv}
	ConfigValues[ConfigTwitchOauthClientSecret] = ConfigValue{Section: Authentication, Default: "", Type: ConfigStringPriv}
	ConfigValues[ConfigDiscordOauthEnabled] = ConfigValue{Section: Authentication, Default: false, Type: ConfigBool}
	ConfigValues[ConfigDiscordOauthSignupEnabled] = ConfigValue{Section: Authentication, Default: false, Type: ConfigBool}
	ConfigValues[ConfigDiscordOauthClientID] = ConfigValue{Section: Authentication, Default: "", Type: ConfigStringPriv}
	ConfigValues[ConfigDiscordOauthClientSecret] = ConfigValue{Section: Authentication, Default: "", Type: ConfigStringPriv}
	ConfigValues[ConfigPatreonOauthEnabled] = ConfigValue{Section: Authentication, Default: false, Type: ConfigBool}
	ConfigValues[ConfigPatreonOauthSignupEnabled] = ConfigValue{Section: Authentication, Default: false, Type: ConfigBool}
	ConfigValues[ConfigPatreonOauthClientID] = ConfigValue{Section: Authentication, Default: "", Type: ConfigStringPriv}
	ConfigValues[ConfigPatreonOauthClientSecret] = ConfigValue{Section: Authentication, Default: "", Type: ConfigStringPriv}

	// Administration
	ConfigSections = append(ConfigSections, Administration)
	ConfigValues[ConfigMaxUserVotes] = ConfigValue{Section: Administration, Default: 5, Type: ConfigInt}
	ConfigValues[ConfigVotingEnabled] = ConfigValue{Section: Administration, Default: false, Type: ConfigBool}
	ConfigValues[ConfigEntriesRequireApproval] = ConfigValue{Section: Administration, Default: false, Type: ConfigBool}
	ConfigValues[ConfigUnlimitedVotes] = ConfigValue{Section: Administration, Default: false, Type: ConfigBool}
}

func (b *backend) LoadDefaultsIfNotSet() error {
	for key, configValue := range ConfigValues {
		switch configValue.Type {
		case ConfigInt:
			_, err := b.data.GetCfgInt(key, configValue.Default.(int))
			if errors.Is(err, database.ErrNoValue) {
				err := b.data.SetCfgInt(key, configValue.Default.(int))
				if err != nil {
					return err
				}
			}
		case ConfigString:
			_, err := b.data.GetCfgString(key, configValue.Default.(string))
			if errors.Is(err, database.ErrNoValue) {
				err := b.data.SetCfgString(key, configValue.Default.(string))
				if err != nil {
					return err
				}
			}
		case ConfigStringPriv:
			_, err := b.data.GetCfgString(key, configValue.Default.(string))
			if errors.Is(err, database.ErrNoValue) {
				err := b.data.SetCfgString(key, configValue.Default.(string))
				if err != nil {
					return err
				}
			}
		case ConfigBool:
			_, err := b.data.GetCfgBool(key, configValue.Default.(bool))
			if errors.Is(err, database.ErrNoValue) {
				err := b.data.SetCfgBool(key, configValue.Default.(bool))
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (b *backend) CheckMovieExists(title string) (bool, error) {
	return b.data.CheckMovieExists(title)
}

func (b *backend) GetJikanEnabled() (bool, error) {
	key := ConfigJikanEnabled
	config, ok := ConfigValues[key]
	if !ok {
		return false, fmt.Errorf("Could not find ConfigValue named %s", key)
	}
	val, err := b.data.GetCfgBool(key, config.Default.(bool))
	if errors.Is(err, database.ErrNoValue) {
		err = b.data.SetCfgBool(key, config.Default.(bool))
		if err != nil {
			b.l.Error("Unable to set default value for %s: %v", key, err)
		}
		return val, nil
	}

	return val, err
}

func (b *backend) GetTmdbEnabled() (bool, error) {
	key := ConfigTmdbEnabled
	config, ok := ConfigValues[key]
	if !ok {
		return false, fmt.Errorf("Could not find ConfigValue named %s", key)
	}
	val, err := b.data.GetCfgBool(key, config.Default.(bool))
	if errors.Is(err, database.ErrNoValue) {
		err = b.data.SetCfgBool(key, config.Default.(bool))
		if err != nil {
			b.l.Error("Unable to set default value for %s: %v", key, err)
		}
		return val, nil
	}

	return val, err
}

func (b *backend) GetTmdbToken() (string, error) {
	key := ConfigTmdbToken
	config, ok := ConfigValues[key]
	if !ok {
		return "", fmt.Errorf("Could not find ConfigValue named %s", key)
	}
	val, err := b.data.GetCfgString(key, config.Default.(string))
	if errors.Is(err, database.ErrNoValue) {
		err = b.data.SetCfgString(key, config.Default.(string))
		if err != nil {
			b.l.Error("Unable to set default value for %s: %v", key, err)
		}
		return val, nil
	}

	return val, err
}

func (b *backend) GetFormFillEnabled() (bool, error) {
	key := ConfigFormfillEnabled
	config, ok := ConfigValues[key]
	if !ok {
		return false, fmt.Errorf("Could not find ConfigValue named %s", key)
	}
	val, err := b.data.GetCfgBool(key, config.Default.(bool))
	if errors.Is(err, database.ErrNoValue) {
		err = b.data.SetCfgBool(key, config.Default.(bool))
		if err != nil {
			b.l.Error("Unable to set default value for %s: %v", key, err)
		}
		return val, nil
	}

	return val, err
}

func (b *backend) GetJikanBannedTypes() ([]string, error) {
	key := ConfigJikanBannedTypes
	config, ok := ConfigValues[key]
	if !ok {
		return nil, fmt.Errorf("Could not find ConfigValue named %s", key)
	}
	val, err := b.data.GetCfgString(key, config.Default.(string))
	if errors.Is(err, database.ErrNoValue) {
		err = b.data.SetCfgString(key, config.Default.(string))
		if err != nil {
			b.l.Error("Unable to set default value for %s: %v", key, err)
		}
		return strings.Split(val, ","), nil
	}

	return strings.Split(val, ","), err
}

func (b *backend) GetMaxRemarksLength() (int, error) {
	key := ConfigMaxRemarksLength
	config, ok := ConfigValues[key]
	if !ok {
		return 0, fmt.Errorf("Could not find ConfigValue named %s", key)
	}
	val, err := b.data.GetCfgInt(key, config.Default.(int))
	if errors.Is(err, database.ErrNoValue) {
		err = b.data.SetCfgInt(key, config.Default.(int))
		if err != nil {
			b.l.Error("Unable to set default value for %s: %v", key, err)
		}
		return val, nil
	}

	return val, err
}

func (b *backend) GetJikanMaxEpisodes() (int, error) {
	key := ConfigJikanMaxEpisodes
	config, ok := ConfigValues[key]
	if !ok {
		return 0, fmt.Errorf("Could not find ConfigValue named %s", key)
	}
	val, err := b.data.GetCfgInt(key, config.Default.(int))
	if errors.Is(err, database.ErrNoValue) {
		err = b.data.SetCfgInt(key, config.Default.(int))
		if err != nil {
			b.l.Error("Unable to set default value for %s: %v", key, err)
		}
		return val, nil
	}

	return val, err
}

func (b *backend) GetMaxDuration() (int, error) {
	key := ConfigMaxMultEpLength
	config, ok := ConfigValues[key]
	if !ok {
		return 0, fmt.Errorf("Could not find ConfigValue named %s", key)
	}
	val, err := b.data.GetCfgInt(key, config.Default.(int))
	if errors.Is(err, database.ErrNoValue) {
		err = b.data.SetCfgInt(key, config.Default.(int))
		if err != nil {
			b.l.Error("Unable to set default value for %s: %v", key, err)
		}
		return val, nil
	}

	return val, err
}

func (b *backend) GetMaxLinkLength() (int, error) {
	key := ConfigMaxLinkLength
	config, ok := ConfigValues[key]
	if !ok {
		return 0, fmt.Errorf("Could not find ConfigValue named %s", key)
	}
	val, err := b.data.GetCfgInt(key, config.Default.(int))
	if errors.Is(err, database.ErrNoValue) {
		err = b.data.SetCfgInt(key, config.Default.(int))
		if err != nil {
			b.l.Error("Unable to set default value for %s: %v", key, err)
		}
		return val, nil
	}

	return val, err
}

func (b *backend) GetMaxTitleLength() (int, error) {
	key := ConfigMaxTitleLength
	config, ok := ConfigValues[key]
	if !ok {
		return 0, fmt.Errorf("Could not find ConfigValue named %s", key)
	}
	val, err := b.data.GetCfgInt(key, config.Default.(int))
	if errors.Is(err, database.ErrNoValue) {
		err = b.data.SetCfgInt(key, config.Default.(int))
		if err != nil {
			b.l.Error("Unable to set default value for %s: %v", key, err)
		}
		return val, nil
	}

	return val, err
}

func (b *backend) GetMaxNameLength() (int, error) {
	key := ConfigMaxNameLength
	config, ok := ConfigValues[key]
	if !ok {
		return 0, fmt.Errorf("Could not find ConfigValue named %s", key)
	}
	val, err := b.data.GetCfgInt(key, config.Default.(int))
	if errors.Is(err, database.ErrNoValue) {
		err = b.data.SetCfgInt(key, config.Default.(int))
		if err != nil {
			b.l.Error("Unable to set default value for %s: %v", key, err)
		}
		return val, nil
	}

	return val, err

}

func (b *backend) GetMinNameLength() (int, error) {
	key := ConfigMinNameLength
	config, ok := ConfigValues[key]
	if !ok {
		return 0, fmt.Errorf("Could not find ConfigValue named %s", key)
	}
	val, err := b.data.GetCfgInt(key, config.Default.(int))
	if errors.Is(err, database.ErrNoValue) {
		err = b.data.SetCfgInt(key, config.Default.(int))
		if err != nil {
			b.l.Error("Unable to set default value for %s: %v", key, err)
		}
		return val, nil
	}

	return val, err
}

func (b *backend) GetMaxDescriptionLength() (int, error) {
	key := ConfigMaxDescriptionLength
	config, ok := ConfigValues[key]
	if !ok {
		return 0, fmt.Errorf("Could not find ConfigValue named %s", key)
	}
	val, err := b.data.GetCfgInt(key, config.Default.(int))
	if errors.Is(err, database.ErrNoValue) {
		err = b.data.SetCfgInt(key, config.Default.(int))
		if err != nil {
			b.l.Error("Unable to set default value for %s: %v", key, err)
		}
		return val, nil
	}

	return val, err
}

func (b *backend) AddMovieToDB(movie *models.Movie) (int, error) {
	return b.data.AddMovie(movie)
}

// Oauth
func (b *backend) GetLocalSignupEnabled() (bool, error) {
	key := ConfigLocalSignupEnabled
	config, ok := ConfigValues[key]
	if !ok {
		return false, fmt.Errorf("Could not find ConfigValue named %s", key)
	}
	val, err := b.data.GetCfgBool(key, config.Default.(bool))
	if errors.Is(err, database.ErrNoValue) {
		err = b.data.SetCfgBool(key, config.Default.(bool))
		if err != nil {
			b.l.Error("Unable to set default value for %s: %v", key, err)
		}
		return val, nil
	}

	return val, err
}

func (b *backend) GetTwitchOauthSignupEnabled() (bool, error) {
	key := ConfigTwitchOauthSignupEnabled
	config, ok := ConfigValues[key]
	if !ok {
		return false, fmt.Errorf("Could not find ConfigValue named %s", key)
	}
	val, err := b.data.GetCfgBool(key, config.Default.(bool))
	if errors.Is(err, database.ErrNoValue) {
		err = b.data.SetCfgBool(key, config.Default.(bool))
		if err != nil {
			b.l.Error("Unable to set default value for %s: %v", key, err)
		}
		return val, nil
	}

	return val, err
}

func (b *backend) GetTwitchOauthEnabled() (bool, error) {
	key := ConfigTwitchOauthEnabled
	config, ok := ConfigValues[key]
	if !ok {
		return false, fmt.Errorf("Could not find ConfigValue named %s", key)
	}
	val, err := b.data.GetCfgBool(key, config.Default.(bool))
	if errors.Is(err, database.ErrNoValue) {
		err = b.data.SetCfgBool(key, config.Default.(bool))
		if err != nil {
			b.l.Error("Unable to set default value for %s: %v", key, err)
		}
		return val, nil
	}

	return val, err
}

func (b *backend) GetDiscordOauthSignupEnabled() (bool, error) {
	key := ConfigDiscordOauthSignupEnabled
	config, ok := ConfigValues[key]
	if !ok {
		return false, fmt.Errorf("Could not find ConfigValue named %s", key)
	}
	val, err := b.data.GetCfgBool(key, config.Default.(bool))
	if errors.Is(err, database.ErrNoValue) {
		err = b.data.SetCfgBool(key, config.Default.(bool))
		if err != nil {
			b.l.Error("Unable to set default value for %s: %v", key, err)
		}
		return val, nil
	}

	return val, err
}

func (b *backend) GetDiscordOauthEnabled() (bool, error) {
	key := ConfigDiscordOauthEnabled
	config, ok := ConfigValues[key]
	if !ok {
		return false, fmt.Errorf("Could not find ConfigValue named %s", key)
	}
	val, err := b.data.GetCfgBool(key, config.Default.(bool))
	if errors.Is(err, database.ErrNoValue) {
		err = b.data.SetCfgBool(key, config.Default.(bool))
		if err != nil {
			b.l.Error("Unable to set default value for %s: %v", key, err)
		}
		return val, nil
	}

	return val, err
}

func (b *backend) GetPatreonOauthSignupEnabled() (bool, error) {
	key := ConfigPatreonOauthSignupEnabled
	config, ok := ConfigValues[key]
	if !ok {
		return false, fmt.Errorf("Could not find ConfigValue named %s", key)
	}
	val, err := b.data.GetCfgBool(key, config.Default.(bool))
	if errors.Is(err, database.ErrNoValue) {
		err = b.data.SetCfgBool(key, config.Default.(bool))
		if err != nil {
			b.l.Error("Unable to set default value for %s: %v", key, err)
		}
		return val, nil
	}

	return val, err
}

func (b *backend) GetPatreonOauthEnabled() (bool, error) {
	key := ConfigPatreonOauthEnabled
	config, ok := ConfigValues[key]
	if !ok {
		return false, fmt.Errorf("Could not find ConfigValue named %s", key)
	}
	val, err := b.data.GetCfgBool(key, config.Default.(bool))
	if errors.Is(err, database.ErrNoValue) {
		err = b.data.SetCfgBool(key, config.Default.(bool))
		if err != nil {
			b.l.Error("Unable to set default value for %s: %v", key, err)
		}
		return val, nil
	}

	return val, err
}

func (b *backend) GetHostAddress() (string, error) {
	key := ConfigHostAddress
	config, ok := ConfigValues[key]
	if !ok {
		return "", fmt.Errorf("Could not find ConfigValue named %s", key)
	}
	val, err := b.data.GetCfgString(key, config.Default.(string))
	if errors.Is(err, database.ErrNoValue) {
		err = b.data.SetCfgString(key, config.Default.(string))
		if err != nil {
			b.l.Error("Unable to set default value for %s: %v", key, err)
		}
		return val, nil
	}

	return val, err
}

func (b *backend) GetTwitchOauthClientID() (string, error) {
	key := ConfigTwitchOauthClientID
	config, ok := ConfigValues[key]
	if !ok {
		return "", fmt.Errorf("Could not find ConfigValue named %s", key)
	}
	val, err := b.data.GetCfgString(key, config.Default.(string))
	if errors.Is(err, database.ErrNoValue) {
		err = b.data.SetCfgString(key, config.Default.(string))
		if err != nil {
			b.l.Error("Unable to set default value for %s: %v", key, err)
		}
		return val, nil
	}

	return val, err
}

func (b *backend) GetTwitchOauthClientSecret() (string, error) {
	key := ConfigTwitchOauthClientSecret
	config, ok := ConfigValues[key]
	if !ok {
		return "", fmt.Errorf("Could not find ConfigValue named %s", key)
	}
	val, err := b.data.GetCfgString(key, config.Default.(string))
	if errors.Is(err, database.ErrNoValue) {
		err = b.data.SetCfgString(key, config.Default.(string))
		if err != nil {
			b.l.Error("Unable to set default value for %s: %v", key, err)
		}
		return val, nil
	}

	return val, err
}

func (b *backend) GetDiscordOauthClientID() (string, error) {
	key := ConfigDiscordOauthClientID
	config, ok := ConfigValues[key]
	if !ok {
		return "", fmt.Errorf("Could not find ConfigValue named %s", key)
	}
	val, err := b.data.GetCfgString(key, config.Default.(string))
	if errors.Is(err, database.ErrNoValue) {
		err = b.data.SetCfgString(key, config.Default.(string))
		if err != nil {
			b.l.Error("Unable to set default value for %s: %v", key, err)
		}
		return val, nil
	}

	return val, err
}

func (b *backend) GetDiscordOauthClientSecret() (string, error) {
	key := ConfigDiscordOauthClientSecret
	config, ok := ConfigValues[key]
	if !ok {
		return "", fmt.Errorf("Could not find ConfigValue named %s", key)
	}
	val, err := b.data.GetCfgString(key, config.Default.(string))
	if errors.Is(err, database.ErrNoValue) {
		err = b.data.SetCfgString(key, config.Default.(string))
		if err != nil {
			b.l.Error("Unable to set default value for %s: %v", key, err)
		}
		return val, nil
	}

	return val, err
}

func (b *backend) GetPatreonOauthClientID() (string, error) {
	key := ConfigPatreonOauthClientID
	config, ok := ConfigValues[key]
	if !ok {
		return "", fmt.Errorf("Could not find ConfigValue named %s", key)
	}
	val, err := b.data.GetCfgString(key, config.Default.(string))
	if errors.Is(err, database.ErrNoValue) {
		err = b.data.SetCfgString(key, config.Default.(string))
		if err != nil {
			b.l.Error("Unable to set default value for %s: %v", key, err)
		}
		return val, nil
	}

	return val, err
}

func (b *backend) GetPatreonOauthClientSecret() (string, error) {
	key := ConfigPatreonOauthClientSecret
	config, ok := ConfigValues[key]
	if !ok {
		return "", fmt.Errorf("Could not find ConfigValue named %s", key)
	}
	val, err := b.data.GetCfgString(key, config.Default.(string))
	if errors.Is(err, database.ErrNoValue) {
		err = b.data.SetCfgString(key, config.Default.(string))
		if err != nil {
			b.l.Error("Unable to set default value for %s: %v", key, err)
		}
		return val, nil
	}

	return val, err
}

func (b *backend) AddUser(user *models.User) (int, error) {
	return b.data.AddUser(user)
}

func (b *backend) UpdateUser(user *models.User) error {
	return b.data.UpdateUser(user)
}

func (b *backend) CheckOauthUsage(id string, authType models.AuthType) bool {
	return b.data.CheckOauthUsage(id, authType)
}

func (b *backend) UserLocalLogin(name string, passwd string) (*models.User, error) {
	return b.data.UserLocalLogin(name, passwd)
}

func (b *backend) UserDiscordLogin(extid string) (*models.User, error) {
	return b.data.UserDiscordLogin(extid)
}

func (b *backend) UserPatreonLogin(extid string) (*models.User, error) {
	return b.data.UserPatreonLogin(extid)
}

func (b *backend) UserTwitchLogin(extid string) (*models.User, error) {
	return b.data.UserTwitchLogin(extid)
}

func (b *backend) GetConfigBanner() (string, error) {
	key := ConfigNoticeBanner
	config, ok := ConfigValues[key]
	if !ok {
		return "", fmt.Errorf("Could not find ConfigValue named %s", key)
	}
	val, err := b.data.GetCfgString(key, config.Default.(string))
	if errors.Is(err, database.ErrNoValue) {
		err = b.data.SetCfgString(key, config.Default.(string))
		if err != nil {
			b.l.Error("Unable to set default value for %s: %v", key, err)
		}
		return val, nil
	}

	return val, err
}

func (b *backend) SetHostAddress(host string) error {
	return b.data.SetCfgString(ConfigHostAddress, host)
}

func (b *backend) SetCfgInt(key string, value int) error {
	return b.data.SetCfgInt(key, value)
}

func (b *backend) SetCfgBool(key string, value bool) error {
	return b.data.SetCfgBool(key, value)
}

func (b *backend) SetCfgString(key string, value string) error {
	return b.data.SetCfgString(key, value)
}

func (b *backend) GetCfgInt(key string, defVal int) (int, error) {
	return b.data.GetCfgInt(key, defVal)
}

func (b *backend) GetCfgBool(key string, defVal bool) (bool, error) {
	return b.data.GetCfgBool(key, defVal)
}

func (b *backend) GetCfgString(key string, defVal string) (string, error) {
	return b.data.GetCfgString(key, defVal)
}

func (b *backend) GetEntriesRequireApproval() (bool, error) {
	key := ConfigEntriesRequireApproval
	config, ok := ConfigValues[key]
	if !ok {
		return false, fmt.Errorf("Could not find ConfigValue named %s", key)
	}
	val, err := b.data.GetCfgBool(key, config.Default.(bool))
	if errors.Is(err, database.ErrNoValue) {
		err = b.data.SetCfgBool(key, config.Default.(bool))
		if err != nil {
			b.l.Error("Unable to set default value for %s: %v", key, err)
		}
		return val, nil
	}

	return val, err
}

func (b *backend) GetAutofillEnabled() (bool, error) {
	jikan, err := b.GetJikanEnabled()
	if err != nil {
		return false, err
	}
	tmdb, err := b.GetTmdbEnabled()
	if err != nil {
		return false, err
	}
	return jikan || tmdb, nil
}
