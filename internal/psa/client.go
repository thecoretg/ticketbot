package psa

import (
	"fmt"

	"resty.dev/v3"
)

type Creds struct {
	PublicKey  string `json:"public_key,omitempty" mapstructure:"public_key"`
	PrivateKey string `json:"private_key,omitempty" mapstructure:"private_key"`
	ClientId   string `json:"client_id,omitempty" mapstructure:"client_id"`
	CompanyId  string `json:"company_id,omitempty" mapstructure:"company_id"` // The company name you enter when you log in to the PSA
}

type Client struct {
	restClient *resty.Client
	creds      *Creds
}

func NewClient(creds *Creds) *Client {
	c := resty.New()
	c.SetBasicAuth(fmt.Sprintf("%s+%s", creds.CompanyId, creds.PublicKey), creds.PrivateKey)
	c.SetHeader("Content-Type", "application/json")
	c.SetHeader("Accept", "application/json")
	c.SetHeader("clientId", creds.ClientId)
	c.SetRetryCount(3)

	return &Client{restClient: c, creds: creds}
}
