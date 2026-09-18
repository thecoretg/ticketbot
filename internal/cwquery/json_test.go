package cwquery

import (
	"encoding/json"
	"testing"
)

func TestToNode(t *testing.T) {
	e, err := Parse("status/name = 'New' and (priority/id in (1, 2) or not closedFlag = true) and owner/id != null")
	if err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(ToNode(e))
	if err != nil {
		t.Fatal(err)
	}
	want := `{"kind":"and","left":{"kind":"and","left":{"kind":"compare","path":"status/name","op":"=","value":{"type":"string","value":"New"}},"right":{"kind":"or","left":{"kind":"in","path":"priority/id","values":[{"type":"number","value":1},{"type":"number","value":2}]},"right":{"kind":"not","expr":{"kind":"compare","path":"closedFlag","op":"=","value":{"type":"bool","value":true}}}}},"right":{"kind":"compare","path":"owner/id","op":"!=","value":{"type":"null","value":null}}}`
	if string(b) != want {
		t.Errorf("got  %s\nwant %s", b, want)
	}

	e, _ = Parse("")
	if b, _ := json.Marshal(ToNode(e)); string(b) != `{"kind":"always"}` {
		t.Errorf("empty = %s", b)
	}
	e, _ = Parse("contact/id not in list 3")
	if b, _ := json.Marshal(ToNode(e)); string(b) != `{"kind":"in_list","path":"contact/id","negate":true,"list_id":3}` {
		t.Errorf("in_list = %s", b)
	}
	e, _ = Parse("summary like 'vpn*'")
	if b, _ := json.Marshal(ToNode(e)); string(b) != `{"kind":"compare","path":"summary","op":"like","value":{"type":"string","value":"vpn*"}}` {
		t.Errorf("like = %s", b)
	}
}
