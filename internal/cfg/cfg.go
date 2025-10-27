package cfg

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/spf13/viper"
	"github.com/thecoretg/ticketbot/internal/logger"
)

type Cfg struct {
	CWPubKey    string `mapstructure:"cw_pub_key"`
	CWPrivKey   string `mapstructure:"cw_priv_key"`
	CWClientID  string `mapstructure:"cw_client_id"`
	CWCompanyID string `mapstructure:"cw_company_id"`
	RootURL     string `mapstructure:"root_url"`
	WebexSecret string `mapstructure:"webex_secret"`
	PostgresDSN string `mapstructure:"postgres_dsn"`

	MaxConcurrentPreloads int    `mapstructure:"max_concurrent_preloads"`
	UseAutoTLS            bool   `mapstructure:"use_auto_tls"`
	InitialAdminEmail     string `mapstructure:"initial_admin_email"`
	ExitOnError           bool   `mapstructure:"exit_on_error"`

	VerboseLogging bool   `mapstructure:"verbose"`
	Debug          bool   `mapstructure:"debug"`
	LogToFile      bool   `mapstructure:"log_to_file"`
	LogFilePath    string `mapstructure:"log_file_path"`

	AttemptNotify bool `mapstructure:"attempt_notify"`

	// Max message length before ticket notifications get a "..." at the end instead of the whole message.
	MaxMsgLength int `mapstructure:"max_msg_length"`

	// Members who we don't want to receive Webex
	ExcludedCWMembers []string `mapstructure:"excluded_cw_members"`
}

func InitCfg() (*Cfg, error) {
	setConfigDefaults()
	viper.AutomaticEnv()

	var c Cfg
	if err := viper.Unmarshal(&c); err != nil {
		return nil, fmt.Errorf("unmarshaling config to json: %w", err)
	}

	if err := logger.SetLogger(c.VerboseLogging, c.Debug, c.LogToFile, c.LogFilePath); err != nil {
		return nil, fmt.Errorf("error setting logger: %w", err)
	}

	slog.Debug("logger set", "debug", c.Debug, "log_to_file", c.LogToFile, "log_file_path", c.LogFilePath)

	slog.Debug("config initialized", "debug", c.Debug, "exit_on_error", c.ExitOnError,
		"log_to_file", c.LogToFile, "log_file_path", c.LogFilePath,
		"root_url", c.RootURL, "max_msg_length", c.MaxMsgLength,
		"excluded_cw_members", c.ExcludedCWMembers,
		"attempt_notify", c.AttemptNotify)

	if !c.isValid() {
		return nil, errors.New("config is missing required fields, please open file and fill any empty fields")
	}
	slog.Debug("config fields validated successfully")

	return &c, nil
}

func (cfg *Cfg) isValid() bool {
	vals := map[string]string{
		"ROOT_URL":            cfg.RootURL,
		"INITIAL_ADMIN_EMAIL": cfg.InitialAdminEmail,
		"CW_PUB_KEY":          cfg.CWPubKey,
		"CW_PRIV_KEY":         cfg.CWPrivKey,
		"CW_CLIENT_ID":        cfg.CWClientID,
		"CW_COMPANY_ID":       cfg.CWCompanyID,
		"POSTGRES_DSN":        cfg.PostgresDSN,
		"WEBEX_SECRET":        cfg.WebexSecret,
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
	viper.SetDefault("exit_on_error", false)
	viper.SetDefault("initial_admin_email", "")
	viper.SetDefault("root_url", "")
	viper.SetDefault("max_concurrent_preloads", 5)
	viper.SetDefault("use_auto_tls", false)
	viper.SetDefault("log_to_file", false)
	viper.SetDefault("log_file_path", "ticketbot.log")
	viper.SetDefault("verbose", false)
	viper.SetDefault("debug", false)
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
