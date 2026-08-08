package session

import "testing"

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
