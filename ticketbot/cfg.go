package ticketbot

import (
	"errors"
	"fmt"
	"github.com/spf13/viper"
	"log/slog"
)

type Cfg struct {
	Debug       bool   `json:"debug" mapstructure:"debug"`
	ExitOnError bool   `json:"exit_on_error" mapstructure:"exit_on_error"`
	LogToFile   bool   `json:"log_to_file" mapstructure:"log_to_file"`
	LogFilePath string `json:"log_file_path" mapstructure:"log_file_path"`

	RootURL     string `json:"root_url" mapstructure:"root_url"`
	UseAutocert bool   `json:"use_autocert" mapstructure:"use_autocert"`

	Creds       *creds
	WebexSecret string `json:"webex_secret" mapstructure:"webex_secret"`
	CwPubKey    string `json:"cw_pub_key" mapstructure:"cw_pub_key"`
	CwPrivKey   string `json:"cw_priv_key" mapstructure:"cw_priv_key"`
	CwClientID  string `json:"cw_client_id" mapstructure:"cw_client_id"`
	CwCompanyID string `json:"cw_company_id" mapstructure:"cw_company_id"`
	PostgresDSN string `json:"postgres_dsn" mapstructure:"postgres_dsn"`

	// Max message length before ticket notifications get a "..." at the end instead of the whole message.
	MaxMsgLength int `json:"max_msg_length" mapstructure:"max_msg_length"`

	// Members who we don't want to receive Webex messages.
	ExcludedCWMembers []string `json:"excluded_cw_members" mapstructure:"excluded_cw_members"`
}

func InitCfg() (*Cfg, error) {
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

	var c Cfg
	if err := viper.Unmarshal(&c); err != nil {
		slog.Error("unmarshaling config to struct", "error", err)
		return nil, fmt.Errorf("unmarshaling config: %w", err)
	}

	c.Creds = &creds{
		WebexSecret: viper.GetString("webex_secret"),
		CwPubKey:    viper.GetString("cw_pub_key"),
		CwPrivKey:   viper.GetString("cw_priv_key"),
		CwClientID:  viper.GetString("cw_client_id"),
		CwCompanyID: viper.GetString("cw_company_id"),
		PostgresDSN: viper.GetString("postgres_dsn"),
	}

	if !c.validateFields() {
		return nil, errors.New("config is still missing required fields after fetching from 1Password, please check config.json")
	}
	slog.Debug("config fields validated successfully")

	return &c, nil
}

func (cfg *Cfg) validateFields() bool {
	vals := map[string]string{
		"root_url":      cfg.RootURL,
		"cw_pub_key":    cfg.Creds.CwPubKey,
		"cw_priv_key":   cfg.Creds.CwPrivKey,
		"cw_client_id":  cfg.Creds.CwClientID,
		"cw_company_id": cfg.Creds.CwCompanyID,
		"postgres_dsn":  cfg.Creds.PostgresDSN,
		"webex_secret":  cfg.Creds.WebexSecret,
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
			return fmt.Errorf("required field is empty: %s, please go to config.json and fill out required fields", k)
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
	viper.SetDefault("use_autocert", false)
	viper.SetDefault("max_msg_length", 300)
	viper.SetDefault("excluded_cw_members", []string{})
	viper.SetDefault("op_svc_token", "")
	viper.SetDefault("creds", &creds{})
}

func isEmpty(s string) bool {
	return s == ""
}
