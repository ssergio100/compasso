package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "agent.toml")
	contents := `
# Local installation settings.
database_path = "/var/lib/tempo-agent/agent.db"
controlled_user = "child"
tick_interval = "2s"
checkpoint_interval = "7s" # bounds crash loss
loginctl_path = "/bin/loginctl"
`
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.DatabasePath != "/var/lib/tempo-agent/agent.db" || got.ControlledUser != "child" ||
		got.TickInterval != 2*time.Second || got.CheckpointInterval != 7*time.Second ||
		got.LoginctlPath != "/bin/loginctl" {
		t.Fatalf("unexpected configuration: %+v", got)
	}
}

func TestLoadRequiresControlledUser(t *testing.T) {
	path := filepath.Join(t.TempDir(), "agent.toml")
	if err := os.WriteFile(path, []byte(`database_path = "/tmp/agent.db"`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("expected missing controlled_user to fail")
	}
}
