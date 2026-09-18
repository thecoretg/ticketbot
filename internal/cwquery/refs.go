package cwquery

// ListRef is one `in list N` reference inside a condition.
type ListRef struct {
	Path   []string
	ListID int
	Pos    int
}

// ListRefs returns every list reference in e, in source order.
func ListRefs(e Expr) []ListRef {
	var out []ListRef
	walk(e, func(n InList) { out = append(out, ListRef{Path: n.Path, ListID: n.ListID, Pos: n.Pos}) })
	return out
}

func walk(e Expr, visit func(InList)) {
	switch n := e.(type) {
	case Not:
		walk(n.X, visit)
	case Binary:
		walk(n.L, visit)
		walk(n.R, visit)
	case InList:
		visit(n)
	}
}
