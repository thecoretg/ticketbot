package workflow

import "testing"

func TestConditionFieldsCarryTheirTransitionCompanions(t *testing.T) {
	cases := map[string][2]string{
		"status/id":       {"changed/status", "old/status/id"},
		"status/name":     {"changed/status", "old/status/name"},
		"owner/id":        {"changed/owner", "old/owner/id"},
		"summary":         {"changed/summary", "old/summary"},
		"type/name":       {"changed/type", ""}, // no previous type is recorded
		"id":              {"", ""},
		"latestNote/text": {"", ""},
		"changed/status":  {"", ""},
		"old/status/id":   {"", ""},
		"resources":       {"", ""}, // a list, not a value you pick
		"closedFlag":      {"", ""}, // a flag
	}
	for path, want := range cases {
		f, ok := FieldByPath(path)
		if !ok {
			t.Fatalf("%s missing", path)
		}
		if f.ChangedPath != want[0] || f.OldPath != want[1] {
			t.Errorf("%s: changed=%q old=%q, want %q %q", path, f.ChangedPath, f.OldPath, want[0], want[1])
		}
	}
}
