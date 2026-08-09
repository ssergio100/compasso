package session

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestSessionNamespacePersistsInsideRuntimeDirectory(t *testing.T) {
	runtimeDirectory := t.TempDir()
	firstIdentifier, err := loadOrCreateSessionNamespace(runtimeDirectory)
	if err != nil {
		t.Fatal(err)
	}
	secondIdentifier, err := loadOrCreateSessionNamespace(runtimeDirectory)
	if err != nil {
		t.Fatal(err)
	}
	if firstIdentifier != secondIdentifier || len(firstIdentifier) != 32 {
		t.Fatalf("session namespace was not preserved: first=%q second=%q", firstIdentifier, secondIdentifier)
	}
	information, err := os.Stat(filepath.Join(runtimeDirectory, "session-namespace-id"))
	if err != nil {
		t.Fatal(err)
	}
	if permissions := information.Mode().Perm(); permissions != 0o600 {
		t.Fatalf("session namespace permissions=%#o, want 0600", permissions)
	}
}

func TestSessionNamespaceRejectsUnsafeStoredFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session-namespace-id")
	if err := os.WriteFile(path, []byte("0123456789abcdef0123456789abcdef\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := readSessionNamespace(path); err == nil {
		t.Fatal("unsafe session namespace permissions were accepted")
	}
}

func TestParsePropertiesAndGraphicalFilter(t *testing.T) {
	session, err := parseProperties("3", []byte("Name=child\nRemote=no\nType=wayland\nClass=user\nState=active\n"))
	if err != nil {
		t.Fatal(err)
	}
	if session.ID != "3" || session.User != "child" || !session.IsLocalGraphical() {
		t.Fatalf("unexpected session: %+v", session)
	}

	nonGraphical := []Session{
		{Type: "tty", Class: "user", State: "active"},
		{Type: "x11", Class: "greeter", State: "active"},
		{Type: "wayland", Class: "user", State: "active", Remote: true},
		{Type: "x11", Class: "user", State: "closing"},
	}
	for _, candidate := range nonGraphical {
		if candidate.IsLocalGraphical() {
			t.Fatalf("session should not count as local graphical: %+v", candidate)
		}
	}
}

func TestParsePropertiesRejectsIncompleteOutput(t *testing.T) {
	if _, err := parseProperties("3", []byte("Name=child\nRemote=no\nType=wayland\n")); err == nil {
		t.Fatal("expected incomplete properties to fail")
	}
}

func TestBalanceAuthorizationIDFallsBackOnlyForAlternateManagers(t *testing.T) {
	current := Session{ID: "7", AuthorizationID: "runtime-namespace_7"}
	if current.BalanceAuthorizationID() != "runtime-namespace_7" {
		t.Fatalf("authorization id=%q", current.BalanceAuthorizationID())
	}
	current.AuthorizationID = ""
	if current.BalanceAuthorizationID() != "7" {
		t.Fatalf("fallback authorization id=%q", current.BalanceAuthorizationID())
	}
}

func TestLogoutDelegatesToCapabilityBasedUserSessionHelper(t *testing.T) {
	manager, err := newLogind("/usr/bin/loginctl", "testnamespace")
	if err != nil {
		t.Fatal(err)
	}
	var commandPath string
	var commandArguments []string
	manager.executeCommand = func(_ context.Context, path string, arguments ...string) ([]byte, error) {
		commandPath = path
		commandArguments = append([]string(nil), arguments...)
		return nil, nil
	}
	current := Session{ID: "3", User: "child", Type: "wayland", Class: "user", State: "active"}
	if err := manager.Logout(context.Background(), current); err != nil {
		t.Fatal(err)
	}
	expectedArguments := []string{
		"--user", "--machine=child@.host", "--wait", "--collect", "--pipe", "--quiet",
		"/usr/libexec/compasso-session-logout",
	}
	if commandPath != "/usr/bin/systemd-run" || !reflect.DeepEqual(commandArguments, expectedArguments) {
		t.Fatalf("command=%q arguments=%q", commandPath, commandArguments)
	}
	for _, argument := range commandArguments {
		switch argument {
		case "terminate-session", "org.kde.Shutdown", "org.gnome.SessionManager":
			t.Fatalf("root daemon should not select a desktop logout mechanism: %q", argument)
		}
	}
}

func TestLogoutNeverFallsBackToAbruptTermination(t *testing.T) {
	manager, err := newLogind("/usr/bin/loginctl", "testnamespace")
	if err != nil {
		t.Fatal(err)
	}
	manager.executeCommand = func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		return []byte("desktop service unavailable"), errors.New("exit status 1")
	}
	current := Session{ID: "9", User: "child", Type: "x11", Class: "user", State: "active"}
	if err := manager.Logout(context.Background(), current); err == nil {
		t.Fatal("missing desktop logout service was treated as success")
	}
}
