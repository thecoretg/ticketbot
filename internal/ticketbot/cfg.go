package ticketbot

import (
	"fmt"
	"github.com/spf13/viper"
)

type Cfg struct {
	Debug       bool   `json:"debug,omitempty" mapstructure:"debug"`
	ExitOnError bool   `json:"exit_on_error,omitempty" mapstructure:"exit_on_error"`
	LogToFile   bool   `json:"log_to_file,omitempty" mapstructure:"log_to_file"`
	LogFilePath string `json:"log_file_path,omitempty" mapstructure:"log_file_path"`

	RootURL     string `json:"root_url,omitempty" mapstructure:"root_url"`
	PostgresDSN string `json:"postgres_dsn,omitempty" mapstructure:"postgres_dsn"`

	CWPublicKey    string `json:"cw_public_key" mapstructure:"cw_public_key"`
	CWPrivateKey   string `json:"cw_private_key" mapstructure:"cw_private_key"`
	CWClientID     string `json:"cw_client_id" mapstructure:"cw_client_id"`
	CWCompanyID    string `json:"cw_company_id" mapstructure:"cw_company_id"`
	WebexBotSecret string `json:"webex_secret,omitempty" mapstructure:"webex_secret"`

	// Max message length before ticket notifications get a "..." at the end instead of the whole message.
	MaxMsgLength int `json:"max_msg_length,omitempty" mapstructure:"max_msg_length"`

	// Members who we don't want to receive Webex messages.
	ExcludedCWMembers []string `json:"excluded_cw_members" mapstructure:"excluded_cw_members"`

	StopIfUpdatedBy []string `json:"stop_if_updated_by" mapstructure:"stop_if_updated_by"`
}

func InitCfg() (*Cfg, error) {
	viper.AutomaticEnv()
	setConfigDefaults()
	cfg := &Cfg{
		Debug:          viper.GetBool("DEBUG"),
		ExitOnError:    viper.GetBool("EXIT_ON_ERROR"),
		LogToFile:      viper.GetBool("LOG_TO_FILE"),
		LogFilePath:    viper.GetString("LOG_FILE_PATH"),
		RootURL:        viper.GetString("ROOT_URL"),
		PostgresDSN:    viper.GetString("POSTGRES_DSN"),
		CWPublicKey:    viper.GetString("CW_PUBLIC_KEY"),
		CWPrivateKey:   viper.GetString("CW_PRIVATE_KEY"),
		CWClientID:     viper.GetString("CW_CLIENT_ID"),
		CWCompanyID:    viper.GetString("CW_COMPANY_ID"),
		WebexBotSecret: viper.GetString("WEBEX_SECRET"),
		MaxMsgLength:   viper.GetInt("MAX_MSG_LENGTH"),
	}

	if err := cfg.validateFields(); err != nil {
		return nil, fmt.Errorf("validating config fields: %w", err)
	}

	return cfg, nil
}

func (cfg *Cfg) validateFields() error {
	vals := map[string]string{
		"root_url":       cfg.RootURL,
		"cw_public_key":  cfg.CWPublicKey,
		"cw_private_key": cfg.CWPrivateKey,
		"cw_client_id":   cfg.CWClientID,
		"cw_company_id":  cfg.CWCompanyID,
		"webex_secret":   cfg.WebexBotSecret,
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
