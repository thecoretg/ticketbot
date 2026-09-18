package models

import (
	"errors"
	"fmt"
)

// Role is a ticketbot access level. Roles form a hierarchy: admin includes editor, editor
// includes viewer.
type Role string

const (
	// RoleViewer can read workflows, tickets, forwards and lists.
	RoleViewer Role = "viewer"
	// RoleEditor can also create, change and delete workflows, forwards and lists.
	RoleEditor Role = "editor"
	// RoleAdmin can also manage users, API keys, config, sync, logs and SSO.
	RoleAdmin Role = "admin"
)

var ErrInvalidRole = errors.New("invalid role")

// Roles lists every role from least to most privileged.
var Roles = []Role{RoleViewer, RoleEditor, RoleAdmin}

func (r Role) level() int {
	switch r {
	case RoleAdmin:
		return 3
	case RoleEditor:
		return 2
	case RoleViewer:
		return 1
	}
	return 0
}

// Valid reports whether r is one of the known roles.
func (r Role) Valid() bool { return r.level() > 0 }

// AtLeast reports whether r grants everything min grants.
func (r Role) AtLeast(min Role) bool { return r.level() >= min.level() }

// ParseRole returns the Role named by s or ErrInvalidRole.
func ParseRole(s string) (Role, error) {
	r := Role(s)
	if !r.Valid() {
		return "", fmt.Errorf("%w: %q", ErrInvalidRole, s)
	}
	return r, nil
}

// HighestRole returns the most privileged role in rs, and false when rs is empty.
func HighestRole(rs []Role) (Role, bool) {
	var best Role
	for _, r := range rs {
		if r.level() > best.level() {
			best = r
		}
	}
	return best, best.Valid()
}
