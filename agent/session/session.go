// Package session integrates tempo-agent with local graphical sessions.
package session

import "context"

// Session is the subset of logind session metadata needed by the daemon.
type Session struct {
	ID     string
	User   string
	Type   string
	Class  string
	State  string
	Remote bool
}

// IsLocalGraphical reports whether usage in this session counts for phase 3.
func (s Session) IsLocalGraphical() bool {
	graphical := s.Type == "x11" || s.Type == "wayland"
	userSession := s.Class == "user" || s.Class == "user-early"
	alive := s.State != "closing"
	return graphical && userSession && alive && !s.Remote
}

// Manager discovers and terminates sessions through systemd-logind.
type Manager interface {
	Sessions(context.Context, string) ([]Session, error)
	Terminate(context.Context, string) error
}
