package cfg

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
	"github.com/thecoretg/ticketbot/internal/logger"
)

type Cfg struct {
	General  GeneralCfg `json:"general" mapstructure:"general"`
	Logging  LoggingCfg `json:"logging" mapstructure:"logging"`
	Creds    CredsCfg   `json:"creds" mapstucture:"creds"`
	Messages MessageCfg `json:"messages" mapstructure:"messages"`
}

type GeneralCfg struct {
	RootURL           string `json:"root_url" mapstructure:"root_url"`
	UseAutoTLS        bool   `json:"use_auto_tls" mapstructure:"use_auto_tls"`
	InitialAdminEmail string `json:"initial_admin_email" mapstructure:"initial_admin_email"`
	ExitOnError       bool   `json:"exit_on_error" mapstructure:"exit_on_error"`
}

type LoggingCfg struct {
	VerboseLogging bool `json:"verbose" mapstructure:"verbose"`
	Debug          bool `json:"debug" mapstructure:"debug"`

	LogToFile   bool   `json:"log_to_file" mapstructure:"log_to_file"`
	LogFilePath string `json:"log_file_path" mapstructure:"log_file_path"`
}

type CredsCfg struct {
	CW          CWCreds `json:"connectwise" mapstructure:"connectwise"`
	WebexSecret string  `json:"webex_secret" mapstructure:"webex_secret"`
	PostgresDSN string  `json:"postgres_dsn" mapstructure:"postgres_dsn"`
}

type CWCreds struct {
	PubKey    string `json:"pub_key" mapstructure:"pub_key"`
	PrivKey   string `json:"priv_key" mapstructure:"priv_key"`
	ClientID  string `json:"client_id" mapstructure:"client_id"`
	CompanyID string `json:"company_id" mapstructure:"company_id"`
}

type MessageCfg struct {
	AttemptNotify bool `json:"attempt_notify" mapstructure:"attempt_notify"`

	// Max message length before ticket notifications get a "..." at the end instead of the whole message.
	MaxMsgLength int `json:"max_msg_length" mapstructure:"max_msg_length"`

	// Members who we don't want to receive Webex messages.
	ExcludedCWMembers []string `json:"excluded_cw_members" mapstructure:"excluded_cw_members"`
}

func InitCfg(configPath string) (*Cfg, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("getting user config directory: %w", err)
	}

	cfgDir := filepath.Join(home, ".config", "ticketbot")
	// TODO: right permissions?
	if err := os.MkdirAll(cfgDir, 0700); err != nil {
		return nil, fmt.Errorf("creating config directory: %w", err)
	}

	setConfigDefaults()
	if configPath != "" {
		viper.SetConfigFile(configPath)
	} else {
		viper.SetConfigName("config")
		viper.SetConfigType("json")
		viper.AddConfigPath(cfgDir)
	}

	if err := viper.ReadInConfig(); err != nil {
		if errors.Is(err, viper.ConfigFileNotFoundError{}); err != nil {
			slog.Info("config file not found, creating now")
			if configPath != "" {
				if err := viper.WriteConfigAs(configPath); err != nil {
					return nil, fmt.Errorf("creating config file at %s: %w", configPath, err)
				}
			} else {
				if err := viper.SafeWriteConfig(); err != nil {
					return nil, fmt.Errorf("creating config file: %w", err)
				}
			}
		} else {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	var c Cfg
	if err := viper.Unmarshal(&c); err != nil {
		return nil, fmt.Errorf("unmarshaling config to json: %w", err)
	}

	if err := logger.SetLogger(c.Logging.VerboseLogging, c.Logging.Debug, c.Logging.LogToFile, c.Logging.LogFilePath); err != nil {
		return nil, fmt.Errorf("error setting logger: %w", err)
	}

	slog.Info("logger set", "debug", c.Logging.Debug, "log_to_file", c.Logging.LogToFile, "log_file_path", c.Logging.LogFilePath)

	slog.Info("config initialized", "debug", c.Logging.Debug, "exit_on_error", c.General.ExitOnError,
		"log_to_file", c.Logging.LogToFile, "log_file_path", c.Logging.LogFilePath,
		"root_url", c.General.RootURL, "max_msg_length", c.Messages.MaxMsgLength,
		"excluded_cw_members", c.Messages.ExcludedCWMembers,
		"attempt_notify", c.Messages.AttemptNotify)

	if !c.isValid() {
		return nil, errors.New("config is missing required fields, please verify env variables")
	}
	slog.Info("config fields validated successfully")

	return &c, nil
}

func (cfg *Cfg) isValid() bool {
	vals := map[string]string{
		"root_url":            cfg.General.RootURL,
		"initial_admin_email": cfg.General.InitialAdminEmail,
		"cw_pub_key":          cfg.Creds.CW.PubKey,
		"cw_priv_key":         cfg.Creds.CW.PrivKey,
		"cw_client_id":        cfg.Creds.CW.ClientID,
		"cw_company_id":       cfg.Creds.CW.CompanyID,
		"postgres_dsn":        cfg.Creds.PostgresDSN,
		"webex_secret":        cfg.Creds.WebexSecret,
	}

	var empty []string
	for k, v := range vals {
		if isEmpty(v) {
			empty = append(empty, k)
		}
	}

	if len(empty) > 0 {
		fmt.Println("Empty required fields found in config:")
		for _, f := range empty {
			fmt.Println(f)
		}
		return false
	}

	return true
}

func setConfigDefaults() {
	viper.SetDefault("general.exit_on_error", false)
	viper.SetDefault("general.initial_admin_email", "")
	viper.SetDefault("general.root_url", "")
	viper.SetDefault("general.use_auto_tls", false)
	viper.SetDefault("logging.log_to_file", false)
	viper.SetDefault("logging.log_file_path", "ticketbot.log")
	viper.SetDefault("logging.verbose", false)
	viper.SetDefault("logging.debug", false)
	viper.SetDefault("creds.connectwise.pub_key", "")
	viper.SetDefault("creds.connectwise.priv_key", "")
	viper.SetDefault("creds.connectwise.client_id", "")
	viper.SetDefault("creds.connectwise.company_id", "")
	viper.SetDefault("creds.webex_secret", "")
	viper.SetDefault("creds.postgres_dsn", "")
	viper.SetDefault("messages.attempt_notify", false)
	viper.SetDefault("messages.max_msg_length", 300)
	viper.SetDefault("messages.excluded_cw_members", []string{})
}

func isEmpty(s string) bool {
	return s == ""
}
