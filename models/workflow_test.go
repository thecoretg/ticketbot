package models

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestDecodeWorkflowDocumentUpgradesV1(t *testing.T) {
	v1 := `[
	  {"id":"r1","name":"New ticket","enabled":true,"trigger":"create","condition":"","stop_processing":false,
	   "actions":[{"kind":"notify","enabled":true,"notify":{"target":"room","recipient_id":3}},
	              {"kind":"notify","enabled":false,"notify":{"target":"resources_owner"}}]},
	  {"id":"r2","name":"Escalate","enabled":false,"trigger":"update","condition":"priority/name = 'Critical'","stop_processing":true,
	   "actions":[{"kind":"set_status","enabled":true,"set_status":{"status_id":10,"status_name":"Escalated"}}]},
	  {"id":"r3","name":"Always","enabled":true,"trigger":"both","condition":"","stop_processing":false,"actions":[]}
	]`
	var w Workflow
	if err := DecodeWorkflowDocument([]byte(v1), &w); err != nil {
		t.Fatal(err)
	}

	byID := map[string]Node{}
	kinds := map[NodeKind]int{}
	for _, n := range w.Nodes {
		byID[n.ID] = n
		kinds[n.Kind]++
	}
	if kinds[NodeTrigger] != 1 || kinds[NodeIf] != 3 || kinds["notify"] != 2 || kinds["set_status"] != 1 || len(w.Nodes) != 7 {
		t.Fatalf("nodes = %+v", w.Nodes)
	}

	var trigger Node
	for _, n := range w.Nodes {
		if n.Kind == NodeTrigger {
			trigger = n
		}
	}
	if len(trigger.Events) != 2 || !trigger.Listens(TriggerCreated) || !trigger.Listens(TriggerUpdated) || !trigger.Enabled {
		t.Errorf("trigger = %+v", trigger)
	}

	r1, r2, r3 := byID["r1"], byID["r2"], byID["r3"]
	if r1.Kind != NodeIf || r1.Title != "New ticket" || r1.Condition != "isNew = true" || !r1.Enabled {
		t.Errorf("r1 = %+v", r1)
	}
	if r2.Condition != "isNew = false and (priority/name = 'Critical')" || r2.Enabled {
		t.Errorf("r2 = %+v", r2)
	}
	if r3.Condition != "" {
		t.Errorf("r3 = %+v", r3)
	}

	// edges: trigger → r1; r1 yes → notify → notify → r2 (no stop); r1 no → r2;
	// r2 yes → set_status (stop: chain ends); r2 no → r3; r3 yes and no dangle
	next := func(from string, port Port) string {
		for _, e := range w.Edges {
			if e.From == from && e.Port == port {
				return e.To
			}
		}
		return ""
	}
	if next(trigger.ID, PortOut) != "r1" {
		t.Errorf("trigger should feed r1")
	}
	n1 := next("r1", PortYes)
	n2 := next(n1, PortOut)
	if byID[n1].Kind != "notify" || byID[n2].Kind != "notify" || next(n2, PortOut) != "r2" || next("r1", PortNo) != "r2" {
		t.Errorf("r1 chain wrong: yes→%s→%s→%s, no→%s", n1, n2, next(n2, PortOut), next("r1", PortNo))
	}
	if byID[n1].Notify == nil || *byID[n1].Notify.RecipientID != 3 || byID[n1].Title != "Notify" || byID[n2].Enabled {
		t.Errorf("action nodes = %+v %+v", byID[n1], byID[n2])
	}
	s := next("r2", PortYes)
	if byID[s].Kind != "set_status" || byID[s].SetStatus.StatusID != 10 || next(s, PortOut) != "" {
		t.Errorf("stop rule chain must end after its actions: %+v → %q", byID[s], next(s, PortOut))
	}
	if next("r2", PortNo) != "r3" || next("r3", PortYes) != "" || next("r3", PortNo) != "" {
		t.Errorf("r2 no → %s, r3 → %s/%s", next("r2", PortNo), next("r3", PortYes), next("r3", PortNo))
	}
	if len(w.Edges) != 7 {
		t.Errorf("edges = %d: %+v", len(w.Edges), w.Edges)
	}
	for _, e := range w.Edges {
		if len(e.ID) != 36 {
			t.Errorf("edge without id: %+v", e)
		}
	}
	// positions: if nodes in one column, actions to the right, nothing overlapping vertically
	if r1.X != 0 || r2.X != 0 || byID[n1].X == 0 || !(r1.Y < byID[n1].Y && byID[n1].Y < byID[n2].Y && byID[n2].Y < r2.Y) {
		t.Errorf("layout: r1=(%v,%v) n1=(%v,%v) n2=(%v,%v) r2=(%v,%v)", r1.X, r1.Y, byID[n1].X, byID[n1].Y, byID[n2].X, byID[n2].Y, r2.X, r2.Y)
	}
}

func TestDecodeWorkflowDocumentV2AndEmpty(t *testing.T) {
	var w Workflow
	for _, raw := range []string{"", "null", "[]", `{"version":2,"nodes":[],"edges":[]}`} {
		if err := DecodeWorkflowDocument([]byte(raw), &w); err != nil {
			t.Errorf("%q: %v", raw, err)
		}
		if raw == "[]" {
			// an empty v1 chain still gets its trigger so the canvas has something to start from
			if len(w.Nodes) != 1 || w.Nodes[0].Kind != NodeTrigger {
				t.Errorf("empty v1: nodes = %+v", w.Nodes)
			}
			continue
		}
		if w.Nodes == nil || w.Edges == nil || len(w.Nodes) != 0 {
			t.Errorf("%q: nodes=%v edges=%v", raw, w.Nodes, w.Edges)
		}
	}

	if err := DecodeWorkflowDocument([]byte(`{"version":3}`), &w); err == nil || !strings.Contains(err.Error(), "version 3") {
		t.Errorf("unknown version err = %v", err)
	}

	in := &Workflow{Nodes: []Node{{ID: "t", Kind: NodeTrigger, Title: "t", Enabled: true, Events: []TriggerEvent{TriggerCreated}, X: 10, Y: 20}}}
	raw, err := EncodeWorkflowDocument(in)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil || doc["version"] != float64(2) || doc["edges"] == nil {
		t.Errorf("encoded = %s (%v)", raw, err)
	}
	var out Workflow
	if err := DecodeWorkflowDocument(raw, &out); err != nil || len(out.Nodes) != 1 || out.Nodes[0].X != 10 || out.Nodes[0].Events[0] != TriggerCreated {
		t.Errorf("round trip = %+v (%v)", out.Nodes, err)
	}
}

func TestNodeJSONShape(t *testing.T) {
	n := Node{ID: "n", Kind: "notify", Title: "Notify", Enabled: true, ActionSettings: ActionSettings{Notify: &NotifyAction{Channel: ChannelWebexRoom}}}
	raw, _ := json.Marshal(n)
	s := string(raw)
	for _, want := range []string{`"kind":"notify"`, `"notify":{"channel":"webex_room"}`, `"x":0`} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %s in %s", want, s)
		}
	}
	for _, unwanted := range []string{"events", "condition", "add_note", "ActionSettings"} {
		if strings.Contains(s, unwanted) {
			t.Errorf("unexpected %s in %s", unwanted, s)
		}
	}
	a := n.Action()
	if a.Kind != ActionNotify || !a.Enabled || a.Notify != n.Notify {
		t.Errorf("Action() = %+v", a)
	}
}
