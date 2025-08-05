package ticketbot

import (
	"errors"
	"fmt"
	"github.com/spf13/viper"
)

type Cfg struct {
	Debug       bool   `json:"debug,omitempty" mapstructure:"debug"`
	ExitOnError bool   `json:"exit_on_error,omitempty" mapstructure:"exit_on_error"`
	LogToFile   bool   `json:"log_to_file,omitempty" mapstructure:"log_to_file"`
	LogFilePath string `json:"log_file_path,omitempty" mapstructure:"log_file_path"`

	RootURL     string `json:"root_url,omitempty" mapstructure:"root_url"`
	UseAutocert bool   `json:"use_autocert" mapstructure:"use_autocert"`

	OPSvcToken string `json:"op_svc_token" mapstructure:"op_svc_token"`

	// Max message length before ticket notifications get a "..." at the end instead of the whole message.
	MaxMsgLength int `json:"max_msg_length,omitempty" mapstructure:"max_msg_length"`

	// Members who we don't want to receive Webex messages.
	ExcludedCWMembers []string `json:"excluded_cw_members" mapstructure:"excluded_cw_members"`

	StopIfUpdatedBy []string `json:"stop_if_updated_by" mapstructure:"stop_if_updated_by"`
	PreloadBoards   bool
	PreloadTickets  bool
}

func InitCfg() (*Cfg, error) {
	viper.SetConfigType("json")
	viper.SetConfigName("config")
	viper.AddConfigPath(".")

	setConfigDefaults()
	if err := viper.ReadInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if !errors.As(err, &configFileNotFoundError) {
			return nil, fmt.Errorf("reading in config: %w", err)
		}
	}

	var c Cfg
	if err := viper.Unmarshal(&c); err != nil {
		return nil, fmt.Errorf("unmarshaling config: %w", err)
	}

	if err := viper.WriteConfigAs("config.json"); err != nil {
		return nil, fmt.Errorf("writing config file: %w", err)
	}

	if err := c.validateFields(); err != nil {
		return nil, fmt.Errorf("validating config fields: %w", err)
	}

	return &c, nil
}

func (cfg *Cfg) validateFields() error {
	vals := map[string]string{
		"root_url":     cfg.RootURL,
		"op_svc_token": cfg.OPSvcToken,
	}

	if err := checkEmptyFields(vals); err != nil {
		return fmt.Errorf("validating fields: %w", err)
	}

	return nil
}

func checkEmptyFields(vals map[string]string) error {
	for k, v := range vals {
		if isEmpty(v) {
			return fmt.Errorf("required field is empty: %s, please go to config.json and fill out required fields", k)
		}
	}
	return nil
}

func setConfigDefaults() {
	viper.SetDefault("root_url", "")
	viper.SetDefault("op_svc_token", "")
}

func isEmpty(s string) bool {
	return s == ""
}
