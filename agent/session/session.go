// Package session integrates tempo-agent with local graphical sessions.
package session

import "context"

// Session is the subset of logind session metadata needed by the daemon.
type Session struct {
	ID              string
	AuthorizationID string
	User            string
	Type            string
	Class           string
	State           string
	Remote          bool
}

// BalanceAuthorizationID distinguishes a logind session across agent runtime
// namespaces. Test doubles and alternate managers may omit it and fall back to
// the raw ID.
func (s Session) BalanceAuthorizationID() string {
	if s.AuthorizationID != "" {
		return s.AuthorizationID
	}
	return s.ID
}

// IsLocalGraphical reports whether usage in this session counts for phase 3.
func (s Session) IsLocalGraphical() bool {
	graphical := s.Type == "x11" || s.Type == "wayland"
	userSession := s.Class == "user" || s.Class == "user-early"
	alive := s.State != "closing"
	return graphical && userSession && alive && !s.Remote
}

// Manager discovers sessions through logind and requests an orderly desktop
// logout through the controlled user's session bus.
type Manager interface {
	Sessions(context.Context, string) ([]Session, error)
	Logout(context.Context, Session) error
}
