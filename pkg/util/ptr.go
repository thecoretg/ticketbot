package util

import "time"

func StrToPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func IntToPtr(i int) *int {
	if i == 0 {
		return nil
	}
	return &i
}

func TimeToPtr(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}
