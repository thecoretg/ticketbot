package connectwise

import (
	"context"
	"encoding/base64"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"net/http"
	"tctg-automation/pkg/aws"
)

type Client struct {
	httpClient   *http.Client
	encodedCreds string
	clientId     string
}

type Creds struct {
	PublicKey  string
	PrivateKey string
	ClientId   string
	CompanyId  string // The company name you enter when you log in to the PSA
}

func NewClient(creds Creds, httpClient *http.Client) *Client {
	username := fmt.Sprintf("%s+%s", creds.CompanyId, creds.PublicKey)
	return &Client{
		httpClient:   httpClient,
		encodedCreds: basicAuth(username, creds.PrivateKey),
		clientId:     creds.ClientId,
	}
}

func NewClientFromAWS(ctx context.Context, httpClient *http.Client, s *ssm.Client, paramName string, withDecryption bool) (*Client, error) {
	creds, err := GetCredsFromAWS(ctx, s, paramName, withDecryption)
	if err != nil {
		return nil, fmt.Errorf("getting creds from AWS: %w", err)
	}

	username := fmt.Sprintf("%s+%s", creds.CompanyId, creds.PublicKey)
	return &Client{
		httpClient:   httpClient,
		encodedCreds: basicAuth(username, creds.PrivateKey),
		clientId:     creds.ClientId,
	}, nil
}

func GetCredsFromAWS(ctx context.Context, s *ssm.Client, paramName string, withDecryption bool) (*Creds, error) {
	c := &Creds{}
	if err := aws.GetAndUnmarshalParam(ctx, s, paramName, withDecryption, c); err != nil {
		return nil, fmt.Errorf("getting connectwise creds from AWS: %w", err)
	}

	return c, nil
}

func basicAuth(username, password string) string {
	auth := username + ":" + password
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(auth))
}
