package cwquery

import (
	"strings"
	"time"
)

// Node is the JSON form of a parsed condition, for editors that render conditions visually.
// Kind is one of: always, not, and, or, compare, in, in_list.
type Node struct {
	Kind   string    `json:"kind"`
	Expr   *Node     `json:"expr,omitempty"`    // not
	Left   *Node     `json:"left,omitempty"`    // and / or
	Right  *Node     `json:"right,omitempty"`   // and / or
	Path   string    `json:"path,omitempty"`    // compare / in, segments joined with "/"
	Op     string    `json:"op,omitempty"`      // compare: = != < > <= >= like contains
	Value  *Literal  `json:"value,omitempty"`   // compare
	Values []Literal `json:"values,omitempty"`  // in
	Negate bool      `json:"negate,omitempty"`  // in / in_list: `not in`
	ListID int       `json:"list_id,omitempty"` // in_list
}

// Literal is a right-hand-side value. Type is string, number, bool, null or time.
type Literal struct {
	Type  string `json:"type"`
	Value any    `json:"value"`
}

// ToNode converts a parsed expression into its JSON form.
func ToNode(e Expr) *Node {
	switch n := e.(type) {
	case Always:
		return &Node{Kind: "always"}
	case Not:
		return &Node{Kind: "not", Expr: ToNode(n.X)}
	case Binary:
		kind := "and"
		if n.Op == Or {
			kind = "or"
		}
		return &Node{Kind: kind, Left: ToNode(n.L), Right: ToNode(n.R)}
	case Compare:
		v := literal(n.Val)
		return &Node{Kind: "compare", Path: strings.Join(n.Path, "/"), Op: n.Op.String(), Value: &v}
	case In:
		vals := make([]Literal, 0, len(n.Vals))
		for _, v := range n.Vals {
			vals = append(vals, literal(v))
		}
		return &Node{Kind: "in", Path: strings.Join(n.Path, "/"), Values: vals, Negate: n.Negate}
	case InList:
		return &Node{Kind: "in_list", Path: strings.Join(n.Path, "/"), ListID: n.ListID, Negate: n.Negate}
	}
	return nil
}

func literal(v Value) Literal {
	switch v.Kind {
	case KString:
		return Literal{Type: "string", Value: v.S}
	case KNumber:
		return Literal{Type: "number", Value: v.N}
	case KBool:
		return Literal{Type: "bool", Value: v.B}
	case KTime:
		return Literal{Type: "time", Value: v.T.Format(time.RFC3339)}
	default:
		return Literal{Type: "null", Value: nil}
	}
}
