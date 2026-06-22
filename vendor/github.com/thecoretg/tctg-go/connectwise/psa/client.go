package psa

import (
	"context"
	"fmt"
	"os"
	"strings"

	"resty.dev/v3"
)

type Config struct {
	PublicKey  string `json:"public_key,omitempty" mapstructure:"public_key"`
	PrivateKey string `json:"private_key,omitempty" mapstructure:"private_key"`
	ClientID   string `json:"client_id,omitempty" mapstructure:"client_id"`
	CompanyID  string `json:"company_id,omitempty" mapstructure:"company_id"` // The company name you enter when you log in to the PSA
}

type Client struct {
	restClient *resty.Client
}

// NewClient builds a ConnectWise PSA client from cfg. The ctx parameter is
// accepted for signature consistency with the other tctg-go clients; PSA uses
// static basic-auth credentials and performs no setup that requires it.
func NewClient(_ context.Context, cfg Config) (*Client, error) {
	var missing []string
	if cfg.PublicKey == "" {
		missing = append(missing, "PublicKey")
	}
	if cfg.PrivateKey == "" {
		missing = append(missing, "PrivateKey")
	}
	if cfg.ClientID == "" {
		missing = append(missing, "ClientID")
	}
	if cfg.CompanyID == "" {
		missing = append(missing, "CompanyID")
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("missing psa config fields: %s", strings.Join(missing, ", "))
	}

	c := resty.New()
	c.SetBasicAuth(fmt.Sprintf("%s+%s", cfg.CompanyID, cfg.PublicKey), cfg.PrivateKey)
	c.SetHeader("Content-Type", "application/json")
	c.SetHeader("Accept", "application/json")
	c.SetHeader("clientId", cfg.ClientID)
	c.SetRetryCount(3)
	c.SetLoggerWarnLevel(false)

	return &Client{restClient: c}, nil
}

func NewClientFromEnv(ctx context.Context) (*Client, error) {
	return NewClient(ctx, Config{
		PublicKey:  os.Getenv("CW_PUB_KEY"),
		PrivateKey: os.Getenv("CW_PRIV_KEY"),
		ClientID:   os.Getenv("CW_CLIENT_ID"),
		CompanyID:  os.Getenv("CW_COMPANY_ID"),
	})
}
