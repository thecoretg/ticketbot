package models

import "fmt"

// Trigger is the version 1 per-rule event filter. It survives only to decode stored v1 documents.
type Trigger string

const (
	TriggerCreate Trigger = "create"
	TriggerUpdate Trigger = "update"
	TriggerBoth   Trigger = "both"
)

// Rule is one step of a version 1 workflow: a flat, ordered chain evaluated top to bottom. It
// survives only so DecodeWorkflowDocument can upgrade stored v1 documents.
type Rule struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	Enabled        bool     `json:"enabled"`
	Trigger        Trigger  `json:"trigger"`
	Condition      string   `json:"condition"`
	Actions        []Action `json:"actions"`
	StopProcessing bool     `json:"stop_processing"`
}

// DefaultTitle is the title an action node gets when the admin has not named it.
func (k ActionKind) DefaultTitle() string {
	switch k {
	case ActionNotify:
		return "Notify"
	case ActionAddNote:
		return "Add note"
	case ActionSkipNotify:
		return "Skip notify"
	case ActionSetStatus:
		return "Set status"
	case ActionSetPriority:
		return "Set priority"
	case ActionSetOwner:
		return "Set owner"
	case ActionAddResource:
		return "Add resource"
	case ActionPatch:
		return "Patch ticket"
	}
	return "Step"
}

// Canvas geometry shared with the frontend: card width and the fixed height per kind. The
// upgrade only needs them to leave the converted graph readable before the first auto-arrange.
const (
	legacyCardW  = 248
	legacyIfH    = 136
	legacyCardH  = 88
	legacyGapY   = 56
	legacyColGap = 72
)

// UpgradeRules converts a version 1 rule chain into an equivalent graph.
//
// One trigger node listens for every event any rule used. The rules become a column of if nodes:
// each rule's condition is prefixed with an isNew guard when its trigger was create or update, so
// the single trigger still reproduces per-rule triggers. A rule's actions hang off its yes port in
// a chain; both the chain's end and the no port flow into the next rule's if node, which is what
// "evaluate every rule in order" means as a graph. A stop_processing rule's chain simply ends. The
// rule ids are kept as the if node ids so history and list references still resolve.
func UpgradeRules(rules []Rule) ([]Node, []Edge) {
	events := legacyEvents(rules)
	trigger := Node{ID: NewID(), Kind: NodeTrigger, Title: "Ticket event", Enabled: true, Events: events}

	nodes := []Node{trigger}
	edges := []Edge{}
	link := func(from string, port Port, to string) {
		edges = append(edges, Edge{ID: NewID(), From: from, To: to, Port: port})
	}

	// pending are the (node, port) pairs waiting to be wired to the next rule's if node
	pending := []Edge{{From: trigger.ID, Port: PortOut}}
	y := float64(legacyCardH + legacyGapY)

	for _, r := range rules {
		id := r.ID
		if id == "" {
			id = NewID()
		}
		cond := Node{ID: id, Kind: NodeIf, Title: r.Name, Enabled: r.Enabled, Condition: legacyCondition(r), X: 0, Y: y}
		if cond.Title == "" {
			cond.Title = "Condition"
		}
		nodes = append(nodes, cond)
		for _, p := range pending {
			link(p.From, p.Port, id)
		}

		// the action chain sits one column right, below the if node
		ay := y + legacyIfH + legacyGapY
		prev, prevPort := id, PortYes
		for _, a := range r.Actions {
			an := Node{ID: NewID(), Kind: NodeKind(a.Kind), Title: a.Kind.DefaultTitle(), Enabled: a.Enabled, ActionSettings: a.ActionSettings, X: legacyCardW + legacyColGap, Y: ay}
			nodes = append(nodes, an)
			link(prev, prevPort, an.ID)
			prev, prevPort = an.ID, PortOut
			ay += legacyCardH + legacyGapY
		}

		pending = []Edge{{From: id, Port: PortNo}}
		if !r.StopProcessing {
			pending = append(pending, Edge{From: prev, Port: prevPort})
		}

		y = max(y+legacyIfH+legacyGapY, ay)
	}

	return nodes, edges
}

// legacyEvents is the union of the rules' triggers; an empty chain listens for both.
func legacyEvents(rules []Rule) []TriggerEvent {
	created, updated := false, false
	for _, r := range rules {
		switch r.Trigger {
		case TriggerCreate:
			created = true
		case TriggerUpdate:
			updated = true
		default:
			created, updated = true, true
		}
	}
	if !created && !updated {
		return []TriggerEvent{TriggerCreated, TriggerUpdated}
	}
	var out []TriggerEvent
	if created {
		out = append(out, TriggerCreated)
	}
	if updated {
		out = append(out, TriggerUpdated)
	}
	return out
}

// legacyCondition folds a rule's trigger into its condition so one trigger node can serve rules
// that used to pick create or update individually.
func legacyCondition(r Rule) string {
	var guard string
	switch r.Trigger {
	case TriggerCreate:
		guard = "isNew = true"
	case TriggerUpdate:
		guard = "isNew = false"
	default:
		return r.Condition
	}
	if r.Condition == "" {
		return guard
	}
	return fmt.Sprintf("%s and (%s)", guard, r.Condition)
}
