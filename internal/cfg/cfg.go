package cfg

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/spf13/viper"
	"github.com/thecoretg/ticketbot/internal/logger"
)

type Cfg struct {
	General  GeneralCfg `mapstructure:"general"`
	Logging  LoggingCfg `mapstructure:"logging"`
	Creds    CredsCfg   `mapstucture:"creds"`
	Messages MessageCfg `mapstructure:"messages"`
}

type GeneralCfg struct {
	RootURL           string `mapstructure:"root_url"`
	UseAutoTLS        bool   `mapstructure:"use_auto_tls"`
	InitialAdminEmail string `mapstructure:"initial_admin_email"`
	ExitOnError       bool   `mapstructure:"exit_on_error"`
}

type LoggingCfg struct {
	VerboseLogging bool `mapstructure:"verbose"`
	Debug          bool `mapstructure:"debug"`

	LogToFile   bool   `mapstructure:"log_to_file"`
	LogFilePath string `mapstructure:"log_file_path"`
}

type CredsCfg struct {
	CW          CWCreds `mapstructure:"connectwise"`
	WebexSecret string  `mapstructure:"webex_secret"`
	PostgresDSN string  `mapstructure:"postgres_dsn"`
}

type CWCreds struct {
	PubKey    string `mapstructure:"pub_key"`
	PrivKey   string `mapstructure:"priv_key"`
	ClientID  string `mapstructure:"client_id"`
	CompanyID string `mapstructure:"company_id"`
}

type MessageCfg struct {
	AttemptNotify bool `mapstructure:"attempt_notify"`

	// Max message length before ticket notifications get a "..." at the end instead of the whole message.
	MaxMsgLength int `mapstructure:"max_msg_length"`

	// Members who we don't want to receive Webex messages.
	ExcludedCWMembers []string `mapstructure:"excluded_cw_members"`
}

func InitCfg() (*Cfg, error) {
	viper.SetEnvPrefix("TBOT")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	setConfigDefaults()

	var c Cfg
	if err := viper.Unmarshal(&c); err != nil {
		return nil, fmt.Errorf("unmarshaling config: %w", err)
	}

	if err := logger.SetLogger(c.Logging.VerboseLogging, c.Logging.Debug, c.Logging.LogToFile, c.Logging.LogFilePath); err != nil {
		return nil, fmt.Errorf("error setting logger: %w", err)
	}
	slog.Debug("logger set", "debug", c.Logging.Debug, "log_to_file", c.Logging.LogToFile, "log_file_path", c.Logging.LogFilePath)

	slog.Debug("config initialized", "debug", c.Logging.Debug, "exit_on_error", c.General.ExitOnError,
		"log_to_file", c.Logging.LogToFile, "log_file_path", c.Logging.LogFilePath,
		"root_url", c.General.RootURL, "max_msg_length", c.Messages.MaxMsgLength,
		"excluded_cw_members", c.Messages.ExcludedCWMembers,
		"attempt_notify", c.Messages.AttemptNotify)

	if !c.isValid() {
		return nil, errors.New("config is missing required fields, please set the missing environment variables")
	}
	slog.Debug("config fields validated successfully")

	return &c, nil
}

func (cfg *Cfg) isValid() bool {
	vals := map[string]string{
		"TBOT_GENERAL_ROOT_URL":             cfg.General.RootURL,
		"TBOT_GENERAL_INITIAL_ADMIN_EMAIL":  cfg.General.InitialAdminEmail,
		"TBOT_CREDS_CONNECTWISE_PUB_KEY":    cfg.Creds.CW.PubKey,
		"TBOT_CREDS_CONNECTWISE_PRIV_KEY":   cfg.Creds.CW.PrivKey,
		"TBOT_CREDS_CONNECTWISE_CLIENT_ID":  cfg.Creds.CW.ClientID,
		"TBOT_CREDS_CONNECTWISE_COMPANY_ID": cfg.Creds.CW.CompanyID,
		"TBOT_CREDS_POSTGRES_DSN":           cfg.Creds.PostgresDSN,
		"TBOT_CREDS_WEBEX_SECRET":           cfg.Creds.WebexSecret,
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
