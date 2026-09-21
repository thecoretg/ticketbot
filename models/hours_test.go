package models

import (
	"testing"
	"time"
)

func TestParseBusinessWindow(t *testing.T) {
	w, err := ParseBusinessWindow("07:30", "19:00", "mon,tue,wed,thu,fri", "America/Chicago")
	if err != nil {
		t.Fatal(err)
	}
	cases := map[time.Time]bool{
		time.Date(2026, 9, 22, 15, 0, 0, 0, time.UTC):  true,  // Tuesday 10:00 Central
		time.Date(2026, 9, 22, 12, 29, 0, 0, time.UTC): false, // 07:29
		time.Date(2026, 9, 22, 12, 30, 0, 0, time.UTC): true,  // 07:30
		time.Date(2026, 9, 23, 0, 0, 0, 0, time.UTC):   false, // 19:00
		time.Date(2026, 9, 26, 15, 0, 0, 0, time.UTC):  false, // Saturday
	}
	for at, want := range cases {
		if got := w.Contains(at); got != want {
			t.Errorf("Contains(%s) = %v, want %v", at, got, want)
		}
	}

	for _, bad := range [][4]string{
		{"7:30am", "19:00", "mon", "America/Chicago"},
		{"19:00", "07:30", "mon", "America/Chicago"},
		{"07:30", "19:00", "", "America/Chicago"},
		{"07:30", "19:00", "mon,funday", "America/Chicago"},
		{"07:30", "19:00", "mon", "Mars/Olympus"},
	} {
		if _, err := ParseBusinessWindow(bad[0], bad[1], bad[2], bad[3]); err == nil {
			t.Errorf("%v should be rejected", bad)
		}
	}
	if NormalizeBusinessDays("fri, Mon,mon,sun") != "mon,fri,sun" {
		t.Errorf("normalize = %q", NormalizeBusinessDays("fri, Mon,mon,sun"))
	}
	if !(&Config{}).BusinessWindow().Contains(time.Date(2026, 9, 22, 15, 0, 0, 0, time.UTC)) {
		t.Error("an empty config falls back to the default window")
	}
}
