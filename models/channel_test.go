package models

import (
	"encoding/json"
	"testing"
)

func TestDecodeWorkflowDocumentUpgradesTargetToChannel(t *testing.T) {
	raw := []byte(`{"version":2,"nodes":[
		{"id":"n1","kind":"notify","title":"Room","enabled":true,"notify":{"target":"room","recipient_id":4}},
		{"id":"n2","kind":"notify","title":"People","enabled":true,"notify":{"target":"resources_owner"}},
		{"id":"n3","kind":"notify","title":"New","enabled":true,"notify":{"channel":"webex_person","recipient_id":9}}
	],"edges":[]}`)
	var w Workflow
	if err := DecodeWorkflowDocument(raw, &w); err != nil {
		t.Fatal(err)
	}
	want := []NotifyChannel{ChannelWebexRoom, ChannelResourcesOwner, ChannelWebexPerson}
	for i, n := range w.Nodes {
		if n.Notify.Channel != want[i] || n.Notify.Target != "" {
			t.Errorf("node %d: channel=%q target=%q, want %q and no target", i, n.Notify.Channel, n.Notify.Target, want[i])
		}
	}
	out, _ := json.Marshal(w.Nodes[0].Notify)
	if string(out) != `{"channel":"webex_room","recipient_id":4}` {
		t.Errorf("re-encoded = %s", out)
	}
}

func TestNotifyChannelRecipientType(t *testing.T) {
	if rt, ok := ChannelWebexRoom.RecipientType(); !ok || rt != RecipientTypeRoom {
		t.Error("webex_room should need a room recipient")
	}
	if _, ok := ChannelResourcesOwner.RecipientType(); ok {
		t.Error("resources_owner names no recipient row")
	}
}
