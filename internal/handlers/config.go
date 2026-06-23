package handlers

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/thecoretg/ticketbot/internal/service/config"
	"github.com/thecoretg/ticketbot/models"
)

type ConfigHandler struct {
	Service *config.Service
}

func NewConfigHandler(svc *config.Service) *ConfigHandler {
	return &ConfigHandler{Service: svc}
}

// configEnvVars maps a config JSON key to the environment variable that, when set,
// takes precedence over the stored value (and locks the field in the panel). Kept
// in sync with server.mergeEnvConfig.
var configEnvVars = map[string]string{
	"root_url":                 "ROOT_URL",
	"cw_company_id":            "CW_COMPANY_ID",
	"cw_client_id":             "CW_CLIENT_ID",
	"cw_public_key":            "CW_PUB_KEY",
	"cw_private_key":           "CW_PRIV_KEY",
	"webex_secret":             "WEBEX_SECRET",
	"cw_bot_member_identifier": "CW_BOT_MEMBER_IDENTIFIER",
	"attempt_notify":           "ATTEMPT_NOTIFY",
	"attempt_workflow":         "ATTEMPT_WORKFLOW",
	"debug_logging":            "DEBUG_LOGGING",
}

// secretKeys are credential fields never returned to the browser. The response
// reports only whether each is configured.
var secretKeys = []string{"cw_private_key", "webex_secret"}

func (h *ConfigHandler) Get(c *gin.Context) {
	cfg, err := h.Service.Get(c.Request.Context())
	if err != nil {
		internalServerError(c, err)
		return
	}

	outputJSON(c, sanitizeConfig(cfg))
}

// sanitizeConfig blanks secret fields, reports which secrets are configured, and
// lists the fields locked by an environment variable so the panel can disable them.
func sanitizeConfig(cfg *models.Config) map[string]any {
	b, _ := json.Marshal(cfg)
	m := map[string]any{}
	_ = json.Unmarshal(b, &m)

	for _, k := range secretKeys {
		set := false
		if v, ok := m[k].(string); ok {
			set = v != ""
		}
		m[k] = ""
		m[k+"_set"] = set
	}

	locked := []string{}
	for key, env := range configEnvVars {
		if os.Getenv(env) != "" {
			locked = append(locked, key)
		}
	}
	m["env_locked"] = locked

	return m
}

func (h *ConfigHandler) Update(c *gin.Context) {
	p := &models.ConfigUpdateParams{}
	if err := c.ShouldBindJSON(p); err != nil {
		badPayloadError(c, err)
		return
	}

	cfg, err := h.Service.Update(c.Request.Context(), p)
	if err != nil {
		internalServerError(c, fmt.Errorf("updating config: %w", err))
		return
	}

	outputJSON(c, sanitizeConfig(cfg))
}

// TestConnections checks the stored Connectwise and Webex credentials against
// their APIs with a cheap authenticated request.
func (h *ConfigHandler) TestConnections(c *gin.Context) {
	res, err := h.Service.TestConnections(c.Request.Context())
	if err != nil {
		internalServerError(c, err)
		return
	}

	outputJSON(c, res)
}
