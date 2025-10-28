package server

type appConfig struct {
	Debug                  bool `json:"debug"`
	AttemptNotify          bool `json:"attempt_notify"`
	MaxMessageLength       int  `json:"max_message_length"`
	MaxConcurrentSyncCalls int  `json:"max_concurrent_sync_calls"`
}

var defaultAppConfig = &appConfig{
	Debug:            true,
	AttemptNotify:    false,
	MaxMessageLength: 300,
}
