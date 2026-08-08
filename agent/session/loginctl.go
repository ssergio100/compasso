package session

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"unicode"
)

// Logind uses loginctl, systemd's command-line client for logind's D-Bus API.
// Commands are executed directly without a shell.
type Logind struct {
	path string
}

// NewLogind creates a logind-backed session manager.
func NewLogind(loginctlPath string) (*Logind, error) {
	if loginctlPath == "" {
		return nil, errors.New("loginctl path cannot be empty")
	}
	return &Logind{path: loginctlPath}, nil
}

// Sessions returns all logind sessions owned by username. Detailed properties
// are requested only for matching accounts to keep each polling cycle small.
func (l *Logind) Sessions(ctx context.Context, username string) ([]Session, error) {
	output, err := exec.CommandContext(ctx, l.path, "list-sessions", "--no-legend", "--no-pager").CombinedOutput()
	if err != nil {
		return nil, commandError("list logind sessions", output, err)
	}

	var sessions []Session
	for _, line := range strings.Split(string(output), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 3 || fields[2] != username {
			continue
		}
		id := fields[0]
		if err := validateSessionID(id); err != nil {
			return nil, err
		}
		details, err := exec.CommandContext(ctx, l.path, "show-session", id, "--no-pager",
			"--property=Name", "--property=Remote", "--property=Type",
			"--property=Class", "--property=State").CombinedOutput()
		if err != nil {
			return nil, commandError("inspect logind session "+id, details, err)
		}
		session, err := parseProperties(id, details)
		if err != nil {
			return nil, err
		}
		if session.User == username {
			sessions = append(sessions, session)
		}
	}
	return sessions, nil
}

// Terminate asks logind to terminate one complete session.
func (l *Logind) Terminate(ctx context.Context, sessionID string) error {
	if err := validateSessionID(sessionID); err != nil {
		return err
	}
	output, err := exec.CommandContext(ctx, l.path, "terminate-session", sessionID).CombinedOutput()
	if err != nil {
		return commandError("terminate logind session "+sessionID, output, err)
	}
	return nil
}

func parseProperties(id string, output []byte) (Session, error) {
	values := make(map[string]string)
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		key, value, found := strings.Cut(line, "=")
		if found {
			values[key] = value
		}
	}
	remote, err := parseLogindBool(values["Remote"])
	if err != nil {
		return Session{}, fmt.Errorf("session %s has invalid Remote property %q", id, values["Remote"])
	}
	if values["Name"] == "" || values["Type"] == "" || values["Class"] == "" || values["State"] == "" {
		return Session{}, fmt.Errorf("session %s has incomplete logind properties", id)
	}
	return Session{
		ID: id, User: values["Name"], Type: values["Type"], Class: values["Class"],
		State: values["State"], Remote: remote,
	}, nil
}

func parseLogindBool(value string) (bool, error) {
	switch value {
	case "yes":
		return true, nil
	case "no":
		return false, nil
	default:
		return strconv.ParseBool(value)
	}
}

func validateSessionID(id string) error {
	if id == "" || strings.HasPrefix(id, "-") {
		return fmt.Errorf("invalid logind session ID %q", id)
	}
	for _, character := range id {
		if !unicode.IsLetter(character) && !unicode.IsDigit(character) && character != '_' && character != '-' && character != '.' {
			return fmt.Errorf("invalid logind session ID %q", id)
		}
	}
	return nil
}

func commandError(action string, output []byte, err error) error {
	message := strings.TrimSpace(string(output))
	if message == "" {
		return fmt.Errorf("%s: %w", action, err)
	}
	return fmt.Errorf("%s: %w: %s", action, err, message)
}
