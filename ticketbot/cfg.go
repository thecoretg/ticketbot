package ticketbot

import (
	"errors"
	"fmt"
	"github.com/joho/godotenv"
	"github.com/spf13/viper"
	"log/slog"
)

type Cfg struct {
	Debug       bool   `json:"debug" mapstructure:"debug"`
	ExitOnError bool   `json:"exit_on_error" mapstructure:"exit_on_error"`
	LogToFile   bool   `json:"log_to_file" mapstructure:"log_to_file"`
	LogFilePath string `json:"log_file_path" mapstructure:"log_file_path"`

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
	if err := godotenv.Load(); err != nil {
		return nil, fmt.Errorf("loading dotenv: %w", err)
	}

	setConfigDefaults()
	viper.AutomaticEnv()
	viper.SetEnvPrefix("TICKETBOT")

	debug := viper.GetBool("DEBUG")
	ltf := viper.GetBool("LOG_TO_FILE")
	lfp := viper.GetString("LOG_FILE_PATH")
	if err := setLogger(debug, ltf, lfp); err != nil {
		return nil, fmt.Errorf("error setting logger: %w", err)
	}
	slog.Info("logger set", "debug", debug, "log_to_file", ltf, "log_file_path", lfp)

	c := &Cfg{
		Debug:             viper.GetBool("debug"),
		ExitOnError:       viper.GetBool("exit_on_error"),
		LogToFile:         viper.GetBool("log_to_file"),
		LogFilePath:       viper.GetString("log_file_path"),
		RootURL:           viper.GetString("root_url"),
		WebexSecret:       viper.GetString("webex_secret"),
		CwPubKey:          viper.GetString("cw_pub_key"),
		CwPrivKey:         viper.GetString("cw_priv_key"),
		CwClientID:        viper.GetString("cw_client_id"),
		CwCompanyID:       viper.GetString("cw_company_id"),
		PostgresDSN:       viper.GetString("postgres_dsn"),
		AttemptNotify:     viper.GetBool("attempt_notify"),
		MaxMsgLength:      viper.GetInt("max_msg_length"),
		ExcludedCWMembers: viper.GetStringSlice("excluded_cw_members"),
	}

	if !c.validateFields() {
		return nil, errors.New("config is missing required fields, please verify env variables")
	}
	slog.Debug("config fields validated successfully")

	return c, nil
}

func (cfg *Cfg) validateFields() bool {
	vals := map[string]string{
		"root_url":      cfg.RootURL,
		"cw_pub_key":    cfg.CwPubKey,
		"cw_priv_key":   cfg.CwPrivKey,
		"cw_client_id":  cfg.CwClientID,
		"cw_company_id": cfg.CwCompanyID,
		"postgres_dsn":  cfg.PostgresDSN,
		"webex_secret":  cfg.WebexSecret,
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
	viper.SetDefault("debug", false)
	viper.SetDefault("exit_on_error", false)
	viper.SetDefault("log_to_file", false)
	viper.SetDefault("log_file_path", "ticketbot.log")
	viper.SetDefault("root_url", "")
	viper.SetDefault("max_msg_length", 300)
	viper.SetDefault("excluded_cw_members", []string{})
}

func isEmpty(s string) bool {
	return s == ""
}
