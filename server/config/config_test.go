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
admin_origin = "https://admin.example"
secure_cookies = false
session_lifetime = "2h"
`), 0o600); err != nil {
		t.Fatal(err)
	}
	configuration, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if configuration.ListenAddress != "127.0.0.1:8080" || configuration.AdminOrigin != "https://admin.example" || configuration.SecureCookies || configuration.SessionLifetime != 2*time.Hour {
		t.Fatalf("unexpected config: %+v", configuration)
	}
}

func TestApplyEnvironmentOverrides(t *testing.T) {
	configuration := Config{
		ListenAddress: "0.0.0.0:8080", DatabasePath: "/var/lib/tempo-server/server.db",
		AdminOrigin: "same-host", SessionLifetime: time.Hour, OnlineTimeout: time.Minute,
	}
	environment := map[string]string{
		"TEMPO_ADMIN_ORIGIN":   "https://admin.example",
		"TEMPO_SECURE_COOKIES": "true",
	}
	updated, err := ApplyEnvironmentOverrides(configuration, func(name string) string { return environment[name] })
	if err != nil {
		t.Fatal(err)
	}
	if updated.AdminOrigin != "https://admin.example" || !updated.SecureCookies {
		t.Fatalf("environment overrides not applied: %+v", updated)
	}
	environment["TEMPO_SECURE_COOKIES"] = "not-a-boolean"
	if _, err := ApplyEnvironmentOverrides(configuration, func(name string) string { return environment[name] }); err == nil {
		t.Fatal("invalid secure cookie override accepted")
	}
}
