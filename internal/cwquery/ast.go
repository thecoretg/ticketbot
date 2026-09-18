package cwquery

import (
	"regexp"
	"time"
)

// Expr is a node in a parsed condition.
type Expr interface {
	isExpr()
}

// Always matches every document. It is what an empty condition compiles to.
type Always struct{}

// Not negates X.
type Not struct{ X Expr }

// BoolOp joins two expressions.
type BoolOp int

const (
	And BoolOp = iota
	Or
)

// Binary is `L and R` or `L or R`.
type Binary struct {
	Op   BoolOp
	L, R Expr
}

// CmpOp is a comparison operator.
type CmpOp int

const (
	Eq CmpOp = iota
	Ne
	Lt
	Gt
	Le
	Ge
	Like
	Contains
)

func (op CmpOp) String() string {
	switch op {
	case Eq:
		return "="
	case Ne:
		return "!="
	case Lt:
		return "<"
	case Gt:
		return ">"
	case Le:
		return "<="
	case Ge:
		return ">="
	case Like:
		return "like"
	case Contains:
		return "contains"
	default:
		return "?"
	}
}

// Compare is `path op value`.
type Compare struct {
	Path []string
	Op   CmpOp
	Val  Value

	re *regexp.Regexp // compiled wildcard pattern; set for Like
}

// In is `path [not] in (v1, v2, ...)`.
type In struct {
	Path   []string
	Vals   []Value
	Negate bool
}

// InList is `path [not] in list N`: membership in an admin-defined list, supplied at evaluation
// time through Env.Lists. Pos is the byte offset of the list id token, for validation messages.
type InList struct {
	Path   []string
	ListID int
	Negate bool
	Pos    int
}

// ValKind is the literal type of a Value.
type ValKind int

const (
	KString ValKind = iota
	KNumber
	KBool
	KNull
	KTime
)

// Value is a literal on the right-hand side of a comparison.
type Value struct {
	Kind ValKind
	S    string
	N    float64
	B    bool
	T    time.Time
}

func (Always) isExpr()  {}
func (Not) isExpr()     {}
func (Binary) isExpr()  {}
func (Compare) isExpr() {}
func (In) isExpr()      {}
func (InList) isExpr()  {}
