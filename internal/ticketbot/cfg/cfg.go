package cfg

import (
	"fmt"
	"tctg-automation/pkg/connectwise"

	"github.com/spf13/viper"
)

type Cfg struct {
	Debug       bool   `json:"debug,omitempty" mapstructure:"debug"`
	ExitOnError bool   `json:"exit_on_error,omitempty" mapstructure:"exit_on_error"`
	LogToFile   bool   `json:"log_to_file,omitempty" mapstructure:"log_to_file"`
	RootURL     string `json:"root_url,omitempty" mapstructure:"root_url"`
	DBConn      string `json:"db_conn,omitempty" mapstructure:"db_conn"`

	CWCreds        connectwise.Creds `json:"cw_creds,omitempty" mapstructure:"cw_creds"`
	WebexBotEmail  string            `json:"webex_bot_email,omitempty" mapstructure:"webex_bot_email"`
	WebexBotSecret string            `json:"webex_bot_secret,omitempty" mapstructure:"webex_bot_secret"`

	MaxMsgLength      int      `json:"max_msg_length,omitempty" mapstructure:"max_msg_length"`
	ExcludedCWMembers []string `json:"excluded_cw_members" mapstructure:"excluded_cw_members"`
}

func InitCfg() (*Cfg, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("json")
	viper.AddConfigPath(".")
	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("reading in config: %w", err)
	}

	setConfigDefaults()
	cfg := &Cfg{}
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshaling config: %w", err)
	}

	if err := cfg.validateFields(); err != nil {
		return nil, fmt.Errorf("validating config fields: %w", err)
	}

	return cfg, nil
}

func (cfg *Cfg) validateFields() error {
	vals := map[string]string{
		"root_url":             cfg.RootURL,
		"cw_creds.public_key":  cfg.CWCreds.PublicKey,
		"cw_creds.private_key": cfg.CWCreds.PrivateKey,
		"cw_creds.client_id":   cfg.CWCreds.ClientId,
		"cw_creds.company_id":  cfg.CWCreds.CompanyId,
	}

	if err := checkEmptyFields(vals); err != nil {
		return fmt.Errorf("validating fields: %w", err)
	}

	return nil
}

func checkEmptyFields(vals map[string]string) error {
	for k, v := range vals {
		if isEmpty(v) {
			return fmt.Errorf("required field is empty: %s", k)
		}
	}
	return nil
}

func setConfigDefaults() {
	viper.SetDefault("max_msg_length", 300)
}

func isEmpty(s string) bool {
	return s == ""
}
