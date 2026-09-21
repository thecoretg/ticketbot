package models

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// Weekday keys as stored in app_config.business_days, Monday first.
var weekdayKeys = []struct {
	key string
	day time.Weekday
}{
	{"mon", time.Monday}, {"tue", time.Tuesday}, {"wed", time.Wednesday}, {"thu", time.Thursday},
	{"fri", time.Friday}, {"sat", time.Saturday}, {"sun", time.Sunday},
}

// BusinessWindow is the parsed form of the business-hours settings: when silence from
// ConnectWise is worth an alert.
type BusinessWindow struct {
	OpenMin  int // minutes after midnight
	CloseMin int
	Days     map[time.Weekday]bool
	Loc      *time.Location
}

// Contains reports whether t falls on a business day between open and close in the window's zone.
func (w BusinessWindow) Contains(t time.Time) bool {
	if w.Loc != nil {
		t = t.In(w.Loc)
	}
	if !w.Days[t.Weekday()] {
		return false
	}
	m := t.Hour()*60 + t.Minute()
	return m >= w.OpenMin && m < w.CloseMin
}

// BusinessWindow parses the config's hours. Settings that fail to parse fall back to the
// defaults, so a bad row degrades the alert rather than disabling it; Validate rejects them on save.
func (c *Config) BusinessWindow() BusinessWindow {
	w, err := ParseBusinessWindow(c.BusinessOpen, c.BusinessClose, c.BusinessDays, c.BusinessZone)
	if err != nil {
		w, _ = ParseBusinessWindow(DefaultConfig.BusinessOpen, DefaultConfig.BusinessClose, DefaultConfig.BusinessDays, DefaultConfig.BusinessZone)
	}
	return w
}

// ParseBusinessWindow validates the four settings: HH:MM times with open before close, a
// comma-separated list of mon..sun with at least one day, and an IANA zone name.
func ParseBusinessWindow(open, close, days, zone string) (BusinessWindow, error) {
	var w BusinessWindow
	var err error
	if w.OpenMin, err = parseHHMM(open); err != nil {
		return w, fmt.Errorf("business open: %w", err)
	}
	if w.CloseMin, err = parseHHMM(close); err != nil {
		return w, fmt.Errorf("business close: %w", err)
	}
	if w.OpenMin >= w.CloseMin {
		return w, fmt.Errorf("business hours: open (%s) must be before close (%s)", open, close)
	}
	w.Days = map[time.Weekday]bool{}
	for _, part := range strings.Split(days, ",") {
		key := strings.ToLower(strings.TrimSpace(part))
		if key == "" {
			continue
		}
		found := false
		for _, wk := range weekdayKeys {
			if wk.key == key {
				w.Days[wk.day] = true
				found = true
			}
		}
		if !found {
			return w, fmt.Errorf("business days: unknown day %q (use mon..sun)", part)
		}
	}
	if len(w.Days) == 0 {
		return w, fmt.Errorf("business days: pick at least one day")
	}
	if w.Loc, err = time.LoadLocation(strings.TrimSpace(zone)); err != nil || strings.TrimSpace(zone) == "" {
		return w, fmt.Errorf("business zone: %q is not a known time zone", zone)
	}
	return w, nil
}

func parseHHMM(s string) (int, error) {
	t, err := time.Parse("15:04", strings.TrimSpace(s))
	if err != nil {
		return 0, fmt.Errorf("%q is not a time like 07:30", s)
	}
	return t.Hour()*60 + t.Minute(), nil
}

// NormalizeBusinessDays returns the days in Monday-first order with one spelling.
func NormalizeBusinessDays(days string) string {
	set := map[string]bool{}
	for _, part := range strings.Split(days, ",") {
		if k := strings.ToLower(strings.TrimSpace(part)); k != "" {
			set[k] = true
		}
	}
	var out []string
	for _, wk := range weekdayKeys {
		if set[wk.key] {
			out = append(out, wk.key)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return false })
	return strings.Join(out, ",")
}
