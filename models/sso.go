package models

import "errors"

var ErrSSORoleMappingNotFound = errors.New("sso role mapping not found")

// SSORoleMapping grants a ticketbot Role to anyone whose Entra ID token carries the app role
// value EntraRole.
type SSORoleMapping struct {
	ID        int    `json:"id"`
	EntraRole string `json:"entra_role"`
	Role      Role   `json:"role"`
}

// SSOStatus is what the dashboard's single sign-on page shows.
type SSOStatus struct {
	// Configured is true when the ENTRA_* environment variables are present.
	Configured bool `json:"configured"`
	// Enabled mirrors Config.SSOEnabled.
	Enabled bool `json:"enabled"`
	// PasswordLoginEnabled mirrors Config.PasswordLoginEnabled.
	PasswordLoginEnabled bool `json:"password_login_enabled"`
	// RedirectURI is the exact value to register on the app registration.
	RedirectURI string            `json:"redirect_uri"`
	TenantID    string            `json:"tenant_id"`
	ClientID    string            `json:"client_id"`
	Mappings    []*SSORoleMapping `json:"mappings"`
}

// AuthMethods tells the login page which sign-in options to show.
type AuthMethods struct {
	SSO      bool `json:"sso"`
	Password bool `json:"password"`
}
