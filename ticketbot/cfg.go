package ticketbot

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

type Cfg struct {
	VerboseLogging    bool   `json:"verbose" mapstructure:"verbose"`
	Debug             bool   `json:"debug" mapstructure:"debug"`
	ExitOnError       bool   `json:"exit_on_error" mapstructure:"exit_on_error"`
	LogToFile         bool   `json:"log_to_file" mapstructure:"log_to_file"`
	LogFilePath       string `json:"log_file_path" mapstructure:"log_file_path"`
	InitialAdminEmail string `json:"initial_admin_email" mapstructure:"initial_admin_email"`

	RootURL string `json:"root_url" mapstructure:"root_url"`

	WebexSecret string `json:"webex_secret" mapstructure:"webex_secret"`
	CwPubKey    string `json:"cw_pub_key" mapstructure:"cw_pub_key"`
	CwPrivKey   string `json:"cw_priv_key" mapstructure:"cw_priv_key"`
	CwClientID  string `json:"cw_client_id" mapstructure:"cw_client_id"`
	CwCompanyID string `json:"cw_company_id" mapstructure:"cw_company_id"`
	PostgresDSN string `json:"postgres_dsn" mapstructure:"postgres_dsn"`

	AttemptNotify bool `json:"attempt_notify" mapstructure:"attempt_notify"`
	// Max message length before ticket notifications get a "..." at the end instead of the whole message.
	MaxMsgLength int `json:"max_msg_length" mapstructure:"max_msg_length"`

	// Members who we don't want to receive Webex messages.
	ExcludedCWMembers []string `json:"excluded_cw_members" mapstructure:"excluded_cw_members"`
}

func InitCfg() (*Cfg, error) {
	_ = godotenv.Load()

	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("getting user home directory: %w", err)
	}

	cfgDir := filepath.Join(home, ".config", "ticketbot")
	// TODO: right permissions?
	if err := os.MkdirAll(cfgDir, 0700); err != nil {
		return nil, fmt.Errorf("creating config directory: %w", err)
	}

	setConfigDefaults()
	viper.SetConfigName("config")
	viper.SetConfigType("json")
	viper.AddConfigPath(cfgDir)
	if err := viper.ReadInConfig(); err != nil {
		if errors.Is(err, viper.ConfigFileNotFoundError{}); err != nil {
			slog.Info("config file not found, creating now", "file_path", filepath.Join(cfgDir, "config.json"))
			if err := viper.SafeWriteConfig(); err != nil {
				return nil, fmt.Errorf("creating config file: %w", err)
			}
		} else {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	var c Cfg
	if err := viper.Unmarshal(&c); err != nil {
		return nil, fmt.Errorf("unmarshaling config to json: %w", err)
	}

	if err := setLogger(c.VerboseLogging, c.Debug, c.LogToFile, c.LogFilePath); err != nil {
		return nil, fmt.Errorf("error setting logger: %w", err)
	}

	slog.Info("logger set", "debug", c.Debug, "log_to_file", c.LogToFile, "log_file_path", c.LogFilePath)

	slog.Info("config initialized", "debug", c.Debug, "exit_on_error", c.ExitOnError,
		"log_to_file", c.LogToFile, "log_file_path", c.LogFilePath,
		"root_url", c.RootURL, "max_msg_length", c.MaxMsgLength,
		"excluded_cw_members", c.ExcludedCWMembers,
		"attempt_notify", c.AttemptNotify)

	if !c.validateFields() {
		return nil, errors.New("config is missing required fields, please verify env variables")
	}
	slog.Info("config fields validated successfully")

	return &c, nil
}

func (cfg *Cfg) validateFields() bool {
	vals := map[string]string{
		"root_url":            cfg.RootURL,
		"initial_admin_email": cfg.InitialAdminEmail,
		"cw_pub_key":          cfg.CwPubKey,
		"cw_priv_key":         cfg.CwPrivKey,
		"cw_client_id":        cfg.CwClientID,
		"cw_company_id":       cfg.CwCompanyID,
		"postgres_dsn":        cfg.PostgresDSN,
		"webex_secret":        cfg.WebexSecret,
	}

	if err := checkEmptyFields(vals); err != nil {
		slog.Error(err.Error())
		return false
	}

	return true
}

func checkEmptyFields(vals map[string]string) error {
	for k, v := range vals {
		if isEmpty(v) {
			slog.Error("required field is empty", "key", k)
			return fmt.Errorf("required field is empty: %s", k)
		}
	}
	return nil
}

func setConfigDefaults() {
	viper.SetDefault("verbose", false)
	viper.SetDefault("debug", false)
	viper.SetDefault("exit_on_error", false)
	viper.SetDefault("log_to_file", false)
	viper.SetDefault("log_file_path", "ticketbot.log")
	viper.SetDefault("initial_admin_email", "")
	viper.SetDefault("root_url", "")
	viper.SetDefault("cw_pub_key", "")
	viper.SetDefault("cw_priv_key", "")
	viper.SetDefault("cw_client_id", "")
	viper.SetDefault("cw_company_id", "")
	viper.SetDefault("webex_secret", "")
	viper.SetDefault("postgres_dsn", "")
	viper.SetDefault("attempt_notify", false)
	viper.SetDefault("max_msg_length", 300)
	viper.SetDefault("excluded_cw_members", []string{})
}

func isEmpty(s string) bool {
	return s == ""
}
