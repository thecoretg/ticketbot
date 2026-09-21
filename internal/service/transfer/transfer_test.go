package transfer

import "testing"

func TestRewriteListsReplacesEveryReferenceInPlace(t *testing.T) {
	got, err := rewriteLists("company/id in list 3 and (contact/id not in list 12 or status/id = 3)", map[int]int{3: 41, 12: 7})
	if err != nil {
		t.Fatal(err)
	}
	want := "company/id in list 41 and (contact/id not in list 7 or status/id = 3)"
	if got != want {
		t.Fatalf("got %q\nwant %q", got, want)
	}
}

func TestRewriteListsRejectsUnknownList(t *testing.T) {
	if _, err := rewriteLists("company/id in list 9", map[int]int{}); err == nil {
		t.Fatal("expected an error for a list the bundle does not carry")
	}
}

func TestListIDs(t *testing.T) {
	ids := listIDs("a in list 2 or b in list 5")
	if len(ids) != 2 || ids[0] != 2 || ids[1] != 5 {
		t.Fatalf("ids = %v", ids)
	}
	if listIDs("") != nil || listIDs("not a condition (") != nil {
		t.Fatal("empty or broken conditions reference nothing")
	}
}
