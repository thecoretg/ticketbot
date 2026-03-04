package models

// Config implements logging.LogConfig so it can be passed directly to the persister.
func (c *Config) GetLogRetentionDays() int        { return c.LogRetentionDays }
func (c *Config) GetLogCleanupIntervalHours() int { return c.LogCleanupIntervalHours }
