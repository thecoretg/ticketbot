package cwsvc

import (
	"log/slog"
	"time"
)

func (s *Service) withinTTL(updatedOn time.Time, entity string, id any) bool {
	expiry := updatedOn.Add(s.TTL)
	stale := time.Now().After(expiry)
	if stale {
		slog.Debug("entity stale, refreshing",
			"entity", entity,
			"id", id,
			"updated_on", updatedOn,
			"ttl", s.TTL.Seconds(),
			"expiry", expiry,
			"stale_for", time.Since(expiry).Seconds(),
		)
		return true
	}
	return false
}
