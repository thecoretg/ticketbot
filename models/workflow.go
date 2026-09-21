package models

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrWorkflowNotFound       = errors.New("workflow not found")
	ErrWorkflowExistsForBoard = errors.New("a workflow already exists for this board")
)

// TriggerEvent is what a trigger node listens for: a ticket ticketbot sees for the first time, or
// one it already knows changing.
type TriggerEvent string

const (
	TriggerCreated TriggerEvent = "created"
	TriggerUpdated TriggerEvent = "updated"
)

func (e TriggerEvent) Valid() bool {
	return e == TriggerCreated || e == TriggerUpdated
}

// EventFor names the event an intake raises.
func EventFor(isNew bool) TriggerEvent {
	if isNew {
		return TriggerCreated
	}
	return TriggerUpdated
}

type ActionKind string

const (
	ActionNotify      ActionKind = "notify"
	ActionAddNote     ActionKind = "add_note"
	ActionSkipNotify  ActionKind = "skip_notify"
	ActionSetStatus   ActionKind = "set_status"
	ActionSetPriority ActionKind = "set_priority"
	ActionSetOwner    ActionKind = "set_owner"
	ActionAddResource ActionKind = "add_resource"
	ActionPatch       ActionKind = "patch"
)

// ActionKinds lists every action kind, in the order the editor offers them.
var ActionKinds = []ActionKind{
	ActionNotify, ActionAddNote, ActionSkipNotify, ActionSetStatus, ActionSetPriority,
	ActionSetOwner, ActionAddResource, ActionPatch,
}

func (k ActionKind) Known() bool {
	for _, known := range ActionKinds {
		if k == known {
			return true
		}
	}
	return false
}

// Mutates reports whether the action writes to the ConnectWise ticket.
func (k ActionKind) Mutates() bool {
	switch k {
	case ActionAddNote, ActionSetStatus, ActionSetPriority, ActionSetOwner, ActionAddResource, ActionPatch:
		return true
	}
	return false
}

// NodeKind is what a node on the canvas does. Besides the two structural kinds every ActionKind is
// also a NodeKind: an action node's kind is the action it runs.
type NodeKind string

const (
	NodeTrigger NodeKind = "trigger"
	NodeIf      NodeKind = "if"
)

// IsAction reports whether the kind is an action kind rather than a structural one.
func (k NodeKind) IsAction() bool { return ActionKind(k).Known() }

func (k NodeKind) Known() bool { return k == NodeTrigger || k == NodeIf || k.IsAction() }

// Port names an output on a node. Triggers and actions have one (out); an if node has two.
type Port string

const (
	PortOut Port = "out"
	PortYes Port = "yes"
	PortNo  Port = "no"
)

// NotifyChannel is where a notify node delivers. Webex is the only transport today; the channel
// is the seam a Slack or Teams room would be added at, as a new kind with its own params rather
// than a new action kind.
type NotifyChannel string

const (
	ChannelWebexRoom      NotifyChannel = "webex_room"
	ChannelWebexPerson    NotifyChannel = "webex_person"
	ChannelResourcesOwner NotifyChannel = "resources_owner" // the ticket's resources and owner, as Webex people
)

// RecipientType is the webex_recipient row type a channel's RecipientID must point at, when it
// names one.
func (c NotifyChannel) RecipientType() (WebexRecipientType, bool) {
	switch c {
	case ChannelWebexRoom:
		return RecipientTypeRoom, true
	case ChannelWebexPerson:
		return RecipientTypePerson, true
	}
	return "", false
}

// NotifyTarget is the pre-channel field name. It is read so stored and exported documents from
// before channels still load; NotifyAction.Normalize maps it and clears it.
type NotifyTarget string

const (
	TargetRoom           NotifyTarget = "room"
	TargetPerson         NotifyTarget = "person"
	TargetResourcesOwner NotifyTarget = "resources_owner"
)

// Workflow is the per-board flow graph. Nodes and edges are stored as one JSON document and
// replaced as a whole.
type Workflow struct {
	ID        int       `json:"id"`
	BoardID   int       `json:"board_id"`
	BoardName string    `json:"board_name,omitempty"`
	Name      string    `json:"name"`
	Enabled   bool      `json:"enabled"`
	DryRun    bool      `json:"dry_run"`
	Nodes     []Node    `json:"nodes"`
	Edges     []Edge    `json:"edges"`
	CreatedOn time.Time `json:"created_on"`
	UpdatedOn time.Time `json:"updated_on"`
}

// Node is one step on the canvas. Which optional fields apply depends on Kind: a trigger has
// Events, an if has Condition, and an action node has exactly the settings block for its kind.
type Node struct {
	// ID is a server-assigned UUID so history and list references survive renames and moves.
	ID      string   `json:"id"`
	Kind    NodeKind `json:"kind"`
	Title   string   `json:"title"`
	Enabled bool     `json:"enabled"`
	// X and Y are the card's canvas position; the canvas persists them so the layout survives.
	X float64 `json:"x"`
	Y float64 `json:"y"`

	Events    []TriggerEvent `json:"events,omitempty"`    // trigger: which intakes enter here
	Condition string         `json:"condition,omitempty"` // if: cwquery source; empty always matches
	ActionSettings
}

// Ports lists the node's output ports.
func (n Node) Ports() []Port {
	if n.Kind == NodeIf {
		return []Port{PortYes, PortNo}
	}
	return []Port{PortOut}
}

// HasPort reports whether p is one of the node's output ports.
func (n Node) HasPort(p Port) bool {
	for _, q := range n.Ports() {
		if p == q {
			return true
		}
	}
	return false
}

// Listens reports whether a trigger node accepts the event.
func (n Node) Listens(ev TriggerEvent) bool {
	for _, e := range n.Events {
		if e == ev {
			return true
		}
	}
	return false
}

// Action views an action node as the action it runs.
func (n Node) Action() Action {
	return Action{Kind: ActionKind(n.Kind), Enabled: n.Enabled, ActionSettings: n.ActionSettings}
}

// Edge wires one node's output port to another node's input.
type Edge struct {
	ID   string `json:"id"`
	From string `json:"from"`
	To   string `json:"to"`
	Port Port   `json:"port"`
}

// Action is a tagged union: exactly the settings block matching Kind is set.
type Action struct {
	Kind    ActionKind `json:"kind"`
	Enabled bool       `json:"enabled"`
	ActionSettings
}

// ActionSettings holds every action kind's settings block; at most one is set.
type ActionSettings struct {
	Notify      *NotifyAction      `json:"notify,omitempty"`
	AddNote     *AddNoteAction     `json:"add_note,omitempty"`
	SetStatus   *SetStatusAction   `json:"set_status,omitempty"`
	SetPriority *SetPriorityAction `json:"set_priority,omitempty"`
	SetOwner    *SetOwnerAction    `json:"set_owner,omitempty"`
	AddResource *AddResourceAction `json:"add_resource,omitempty"`
	Patch       *PatchAction       `json:"patch,omitempty"`
}

type NotifyAction struct {
	Channel     NotifyChannel `json:"channel"`
	RecipientID *int          `json:"recipient_id,omitempty"` // webex_recipient.id for the webex_* channels
	// Message, when set, replaces the default notification body. It may use {{placeholder}}
	// tokens; see msgtemplate.Placeholders.
	Message string `json:"message,omitempty"`
	// Target is the pre-channel name of Channel. Never written; see Normalize.
	Target NotifyTarget `json:"target,omitempty"`
}

// Normalize upgrades a pre-channel action in place: target room / person / resources_owner
// becomes the matching channel. A document that already has a channel is left alone.
func (n *NotifyAction) Normalize() {
	if n == nil {
		return
	}
	if n.Channel == "" {
		switch n.Target {
		case TargetRoom:
			n.Channel = ChannelWebexRoom
		case TargetPerson:
			n.Channel = ChannelWebexPerson
		case TargetResourcesOwner:
			n.Channel = ChannelResourcesOwner
		}
	}
	n.Target = ""
}

// NormalizeNodes upgrades every notify node in place.
func NormalizeNodes(nodes []Node) {
	for i := range nodes {
		nodes[i].Notify.Normalize()
	}
}

// SetStatusAction moves the ticket to a status on its board. StatusName is display-only and is
// refreshed from the status table on save.
type SetStatusAction struct {
	StatusID   int    `json:"status_id"`
	StatusName string `json:"status_name,omitempty"`
}

// SetPriorityAction changes the ticket priority. PriorityName is display-only.
type SetPriorityAction struct {
	PriorityID   int    `json:"priority_id"`
	PriorityName string `json:"priority_name,omitempty"`
}

// SetOwnerAction assigns the ticket owner. Identifier is filled from the member table on save.
type SetOwnerAction struct {
	MemberID   int    `json:"member_id"`
	Identifier string `json:"identifier,omitempty"`
}

// AddResourceAction appends a member to the ticket's resources. Identifier is filled from the
// member table on save and is what ConnectWise's resources field stores.
type AddResourceAction struct {
	MemberID   int    `json:"member_id"`
	Identifier string `json:"identifier,omitempty"`
}

// PatchAction sends admin-supplied JSON Patch operations to the ticket. Ops is a JSON array of
// {"op", "path", "value"} objects as ConnectWise's PATCH endpoint accepts them.
type PatchAction struct {
	Ops json.RawMessage `json:"ops"`
}

// AddNoteAction posts a plain-text note to the ticket. At least one flag must be set.
type AddNoteAction struct {
	Text       string `json:"text"`
	Internal   bool   `json:"internal"`   // InternalAnalysisFlag
	Discussion bool   `json:"discussion"` // DetailDescriptionFlag
	Resolution bool   `json:"resolution"` // ResolutionFlag
}

// WorkflowDocumentVersion is the shape of the stored JSON document. Version 1 was a bare array of
// rules (see Rule); version 2 is a WorkflowDocument object.
const WorkflowDocumentVersion = 2

// WorkflowDocument is the stored form of a workflow's graph.
type WorkflowDocument struct {
	Version int    `json:"version"`
	Nodes   []Node `json:"nodes"`
	Edges   []Edge `json:"edges"`
}

// EncodeWorkflowDocument serializes the graph in the current document shape.
func EncodeWorkflowDocument(w *Workflow) ([]byte, error) {
	doc := WorkflowDocument{Version: WorkflowDocumentVersion, Nodes: w.Nodes, Edges: w.Edges}
	if doc.Nodes == nil {
		doc.Nodes = []Node{}
	}
	if doc.Edges == nil {
		doc.Edges = []Edge{}
	}
	b, err := json.Marshal(doc)
	if err != nil {
		return nil, fmt.Errorf("marshalling workflow document: %w", err)
	}
	return b, nil
}

// DecodeWorkflowDocument fills w.Nodes and w.Edges from a stored document of either version. A
// version 1 rule list is upgraded to a graph in memory; it is stored as version 2 on the next save.
func DecodeWorkflowDocument(raw []byte, w *Workflow) error {
	w.Nodes, w.Edges = []Node{}, []Edge{}

	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return nil
	}

	if strings.HasPrefix(trimmed, "[") {
		var rules []Rule
		if err := json.Unmarshal(raw, &rules); err != nil {
			return fmt.Errorf("unmarshalling v1 rules: %w", err)
		}
		w.Nodes, w.Edges = UpgradeRules(rules)
		NormalizeNodes(w.Nodes)
		return nil
	}

	var doc WorkflowDocument
	if err := json.Unmarshal(raw, &doc); err != nil {
		return fmt.Errorf("unmarshalling workflow document: %w", err)
	}
	if doc.Version != WorkflowDocumentVersion {
		return fmt.Errorf("unsupported workflow document version %d", doc.Version)
	}
	if doc.Nodes != nil {
		w.Nodes = doc.Nodes
	}
	if doc.Edges != nil {
		w.Edges = doc.Edges
	}
	NormalizeNodes(w.Nodes)
	return nil
}

// ValidationError points at one problem in a workflow document. NodeID or EdgeID says where;
// both empty means the workflow as a whole.
type ValidationError struct {
	NodeID  string `json:"node_id,omitempty"`
	EdgeID  string `json:"edge_id,omitempty"`
	Field   string `json:"field"`
	Message string `json:"message"`
	Pos     *int   `json:"pos,omitempty"` // character offset for condition syntax errors
}

func (e ValidationError) Error() string {
	var sb strings.Builder
	switch {
	case e.NodeID != "":
		fmt.Fprintf(&sb, "node %s", e.NodeID)
	case e.EdgeID != "":
		fmt.Fprintf(&sb, "edge %s", e.EdgeID)
	default:
		sb.WriteString("workflow")
	}
	fmt.Fprintf(&sb, ": %s: %s", e.Field, e.Message)
	return sb.String()
}

type ValidationErrors []ValidationError

func (v ValidationErrors) Error() string {
	msgs := make([]string, 0, len(v))
	for _, e := range v {
		msgs = append(msgs, e.Error())
	}
	return "workflow validation failed: " + strings.Join(msgs, "; ")
}

// NewID returns a random RFC 4122 version 4 UUID for nodes and edges.
func NewID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(fmt.Sprintf("crypto/rand unavailable: %v", err))
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	h := hex.EncodeToString(b[:])
	return h[0:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:]
}
