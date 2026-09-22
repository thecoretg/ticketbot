package mcp

import (
	"fmt"
	"strings"

	"github.com/thecoretg/ticketbot/models"
)

// describeWorkflow renders the graph as lanes: one block per trigger, walking the wires and
// indenting under each if branch. It is what ticketbot_get_workflow returns by default, because
// the stored document is nodes plus edges and reads poorly in a context window.
func describeWorkflow(w *models.Workflow) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Workflow %d %q on board %d", w.ID, w.Name, w.BoardID)
	if w.BoardName != "" {
		fmt.Fprintf(&b, " (%s)", w.BoardName)
	}
	b.WriteString("\n")
	fmt.Fprintf(&b, "enabled: %t, dry_run: %t, nodes: %d, edges: %d\n", w.Enabled, w.DryRun, len(w.Nodes), len(w.Edges))

	byID := make(map[string]models.Node, len(w.Nodes))
	for _, n := range w.Nodes {
		byID[n.ID] = n
	}
	next := func(from string, port models.Port) (models.Node, bool) {
		for _, e := range w.Edges {
			if e.From == from && e.Port == port {
				n, ok := byID[e.To]
				return n, ok
			}
		}
		return models.Node{}, false
	}

	lane := 0
	for _, n := range w.Nodes {
		if n.Kind != models.NodeTrigger {
			continue
		}
		lane++
		fmt.Fprintf(&b, "\nLane %d: %s", lane, title(n))
		if !n.Enabled {
			b.WriteString(" [disabled: never fires]")
		}
		b.WriteString("\n")
		events := make([]string, 0, len(n.Events))
		for _, ev := range n.Events {
			events = append(events, string(ev))
		}
		fmt.Fprintf(&b, "  fires on: %s\n", strings.Join(events, ", "))
		if c := strings.TrimSpace(n.Condition); c != "" {
			fmt.Fprintf(&b, "  only when: %s\n", c)
		}
		start, ok := next(n.ID, models.PortOut)
		if !ok {
			b.WriteString("  (nothing wired)\n")
			continue
		}
		walk(&b, start, next, 1, map[string]bool{})
	}
	if lane == 0 {
		b.WriteString("\n(no trigger nodes)\n")
	}
	return b.String()
}

func walk(b *strings.Builder, n models.Node, next func(string, models.Port) (models.Node, bool), depth int, seen map[string]bool) {
	pad := strings.Repeat("  ", depth)
	if seen[n.ID] {
		fmt.Fprintf(b, "%s-> (joins %s, already described)\n", pad, title(n))
		return
	}
	seen[n.ID] = true

	switch n.Kind {
	case models.NodeIf:
		fmt.Fprintf(b, "%sif %s", pad, condOr(n.Condition, "(no condition)"))
		if !n.Enabled {
			b.WriteString(" [disabled: always takes no]")
		}
		b.WriteString("\n")
		for _, port := range []models.Port{models.PortYes, models.PortNo} {
			nx, ok := next(n.ID, port)
			fmt.Fprintf(b, "%s  %s:", pad, port)
			if !ok {
				b.WriteString(" (end)\n")
				continue
			}
			b.WriteString("\n")
			walk(b, nx, next, depth+2, seen)
		}
	default:
		fmt.Fprintf(b, "%s%s", pad, describeAction(n))
		if !n.Enabled {
			b.WriteString(" [disabled: passes through]")
		}
		b.WriteString("\n")
		if nx, ok := next(n.ID, models.PortOut); ok {
			walk(b, nx, next, depth, seen)
		} else {
			fmt.Fprintf(b, "%s(end)\n", pad)
		}
	}
}

func title(n models.Node) string {
	if strings.TrimSpace(n.Title) != "" {
		return fmt.Sprintf("%q [%s]", n.Title, n.ID)
	}
	return fmt.Sprintf("%s [%s]", n.Kind, n.ID)
}

func condOr(c, fallback string) string {
	if strings.TrimSpace(c) == "" {
		return fallback
	}
	return c
}

// describeAction is one line per action node with the settings that matter.
func describeAction(n models.Node) string {
	a := n.Action()
	head := fmt.Sprintf("%s %s", a.Kind, title(n))
	switch a.Kind {
	case models.ActionNotify:
		if a.Notify == nil {
			return head
		}
		to := string(a.Notify.Channel)
		if a.Notify.RecipientID != nil {
			to += fmt.Sprintf(" recipient %d", *a.Notify.RecipientID)
		}
		return fmt.Sprintf("%s -> %s: %s", head, to, truncate(a.Notify.Message, 160))
	case models.ActionAddNote:
		if a.AddNote == nil {
			return head
		}
		var flags []string
		if a.AddNote.Internal {
			flags = append(flags, "internal")
		}
		if a.AddNote.Discussion {
			flags = append(flags, "discussion")
		}
		if a.AddNote.Resolution {
			flags = append(flags, "resolution")
		}
		return fmt.Sprintf("%s (%s): %s", head, strings.Join(flags, ","), truncate(a.AddNote.Text, 160))
	case models.ActionSetStatus:
		if a.SetStatus == nil {
			return head
		}
		return fmt.Sprintf("%s -> %q (status %d)", head, a.SetStatus.StatusName, a.SetStatus.StatusID)
	case models.ActionSetPriority:
		if a.SetPriority == nil {
			return head
		}
		return fmt.Sprintf("%s -> %q (priority %d)", head, a.SetPriority.PriorityName, a.SetPriority.PriorityID)
	case models.ActionSetOwner:
		if a.SetOwner == nil {
			return head
		}
		return fmt.Sprintf("%s -> %s (member %d)", head, a.SetOwner.Identifier, a.SetOwner.MemberID)
	case models.ActionAddResource:
		if a.AddResource == nil {
			return head
		}
		return fmt.Sprintf("%s -> %s (member %d)", head, a.AddResource.Identifier, a.AddResource.MemberID)
	case models.ActionPatch:
		if a.Patch == nil {
			return head
		}
		return fmt.Sprintf("%s ops=%s", head, truncate(string(a.Patch.Ops), 200))
	case models.ActionSkipNotify:
		return head + " (silences the notifies after it on this lane)"
	}
	return head
}

func truncate(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
