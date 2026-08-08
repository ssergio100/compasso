package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "server.toml")
	if err := os.WriteFile(path, []byte(`
listen_address = "127.0.0.1:8080"
database_path = "./var/server.db"
secure_cookies = false
session_lifetime = "2h"
`), 0o600); err != nil {
		t.Fatal(err)
	}
	configuration, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if configuration.ListenAddress != "127.0.0.1:8080" || configuration.SecureCookies || configuration.SessionLifetime != 2*time.Hour {
		t.Fatalf("unexpected config: %+v", configuration)
	}
}
