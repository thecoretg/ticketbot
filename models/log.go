package models

// Config implements the narrow config interfaces of the persister, the intake service and the
// purgers, so it can be passed to each directly.
func (c *Config) GetLogRetentionDays() int       { return c.LogRetentionDays }
func (c *Config) GetStaleAlertMinutes() int      { return c.StaleAlertMinutes }
func (c *Config) GetHistoryRetentionDays() int   { return c.HistoryRetentionDays }
func (c *Config) GetIntakeRetentionDays() int    { return c.IntakeRetentionDays }
func (c *Config) GetCatchupIntervalMinutes() int { return c.CatchupIntervalMinutes }

// ClosedTicketRetention returns the closed-ticket retention in days, or 0 while it is off.
func (c *Config) ClosedTicketRetention() int {
	if !c.ClosedTicketRetentionEnabled {
		return 0
	}
	return c.ClosedTicketRetentionDays
}
