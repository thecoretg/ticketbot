package cwquery

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Eval reports whether doc satisfies the query. doc is the shape produced by NewDocument:
// nested map[string]any / []any / string / float64 / bool / nil, as decoded from JSON.
//
// Semantics:
//   - Path segments match keys case-insensitively; a missing key resolves to null.
//   - String equality, "in" and "contains" are case-insensitive. "like" supports '*' / '%' (any run)
//     and '_' (single character) wildcards and is anchored.
//   - A numeric string is coerced to a number when compared against a number literal.
//   - Zero values are frequently omitted from ConnectWise JSON, so null compares loosely:
//     null = false, null = 0 and null = ” are all true. Ordering operators against null are false.
//   - Type mismatches evaluate to false rather than erroring.
func (q *Query) Eval(doc map[string]any) (bool, error) {
	return eval(q.Expr, doc)
}

func eval(e Expr, doc map[string]any) (bool, error) {
	switch n := e.(type) {
	case Always:
		return true, nil
	case Not:
		v, err := eval(n.X, doc)
		return !v, err
	case Binary:
		l, err := eval(n.L, doc)
		if err != nil {
			return false, err
		}
		if n.Op == And && !l {
			return false, nil
		}
		if n.Op == Or && l {
			return true, nil
		}
		return eval(n.R, doc)
	case Compare:
		return compare(n, resolve(doc, n.Path)), nil
	case In:
		field := resolve(doc, n.Path)
		for _, v := range n.Vals {
			if equal(field, v) {
				return !n.Negate, nil
			}
		}
		return n.Negate, nil
	default:
		return false, fmt.Errorf("cwquery: unknown expression %T", e)
	}
}

// resolve walks path through doc with case-insensitive key matching. Missing keys yield nil.
func resolve(doc map[string]any, path []string) any {
	var cur any = doc
	for _, seg := range path {
		m, ok := cur.(map[string]any)
		if !ok {
			return nil
		}
		cur = lookup(m, seg)
		if cur == nil {
			return nil
		}
	}

	return cur
}

func lookup(m map[string]any, key string) any {
	if v, ok := m[key]; ok {
		return v
	}
	for k, v := range m {
		if strings.EqualFold(k, key) {
			return v
		}
	}

	return nil
}

func compare(c Compare, field any) bool {
	switch c.Op {
	case Eq:
		return equal(field, c.Val)
	case Ne:
		return !equal(field, c.Val)
	case Lt, Gt, Le, Ge:
		return order(field, c.Val, c.Op)
	case Contains:
		s, ok := asString(field)
		return ok && strings.Contains(strings.ToLower(s), strings.ToLower(c.Val.S))
	case Like:
		s, ok := asString(field)
		return ok && c.re != nil && c.re.MatchString(s)
	}

	return false
}

func equal(field any, v Value) bool {
	if field == nil {
		return isZeroValue(v)
	}

	switch f := field.(type) {
	case string:
		switch v.Kind {
		case KString:
			return strings.EqualFold(f, v.S)
		case KNumber:
			n, err := strconv.ParseFloat(strings.TrimSpace(f), 64)
			return err == nil && n == v.N
		case KBool:
			b, err := strconv.ParseBool(strings.ToLower(strings.TrimSpace(f)))
			return err == nil && b == v.B
		case KTime:
			t, ok := asTime(f)
			return ok && t.Equal(v.T)
		case KNull:
			return f == ""
		}
	case float64:
		switch v.Kind {
		case KNumber:
			return f == v.N
		case KString:
			n, err := strconv.ParseFloat(strings.TrimSpace(v.S), 64)
			return err == nil && n == f
		case KNull:
			return f == 0
		}
	case bool:
		switch v.Kind {
		case KBool:
			return f == v.B
		case KString:
			b, err := strconv.ParseBool(strings.ToLower(strings.TrimSpace(v.S)))
			return err == nil && b == f
		case KNull:
			return !f
		}
	case map[string]any, []any:
		// object/array present; only "= null" makes sense and it is false
		return false
	}

	return false
}

// isZeroValue implements null looseness: an absent field equals null and any zero literal.
func isZeroValue(v Value) bool {
	switch v.Kind {
	case KNull:
		return true
	case KBool:
		return !v.B
	case KNumber:
		return v.N == 0
	case KString:
		return v.S == ""
	}
	return false
}

func order(field any, v Value, op CmpOp) bool {
	if field == nil {
		return false
	}

	// time comparison when the literal is a date
	if v.Kind == KTime {
		s, ok := field.(string)
		if !ok {
			return false
		}
		t, ok := asTime(s)
		if !ok {
			return false
		}
		return cmpResult(compareTimes(t, v.T), op)
	}

	// numeric when both sides are numeric
	if fn, ok := asNumber(field); ok {
		if vn, ok := valueNumber(v); ok {
			return cmpResult(compareFloats(fn, vn), op)
		}
	}

	// fall back to case-insensitive lexical order on strings
	fs, ok := asString(field)
	if !ok || v.Kind != KString {
		return false
	}
	return cmpResult(strings.Compare(strings.ToLower(fs), strings.ToLower(v.S)), op)
}

func cmpResult(c int, op CmpOp) bool {
	switch op {
	case Lt:
		return c < 0
	case Gt:
		return c > 0
	case Le:
		return c <= 0
	case Ge:
		return c >= 0
	}
	return false
}

func compareFloats(a, b float64) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}

func compareTimes(a, b time.Time) int {
	switch {
	case a.Before(b):
		return -1
	case a.After(b):
		return 1
	default:
		return 0
	}
}

func asString(field any) (string, bool) {
	switch f := field.(type) {
	case string:
		return f, true
	case float64:
		return strconv.FormatFloat(f, 'f', -1, 64), true
	case bool:
		return strconv.FormatBool(f), true
	}
	return "", false
}

func asNumber(field any) (float64, bool) {
	switch f := field.(type) {
	case float64:
		return f, true
	case string:
		n, err := strconv.ParseFloat(strings.TrimSpace(f), 64)
		return n, err == nil
	}
	return 0, false
}

func valueNumber(v Value) (float64, bool) {
	switch v.Kind {
	case KNumber:
		return v.N, true
	case KString:
		n, err := strconv.ParseFloat(strings.TrimSpace(v.S), 64)
		return n, err == nil
	}
	return 0, false
}

func asTime(s string) (time.Time, bool) {
	t, err := parseDate(strings.TrimSpace(s))
	return t, err == nil
}
