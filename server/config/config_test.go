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
heartbeat_interval = "7s"
`), 0o600); err != nil {
		t.Fatal(err)
	}
	configuration, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if configuration.ListenAddress != "127.0.0.1:8080" || configuration.AdminOrigin != "https://admin.example" || configuration.SecureCookies ||
		configuration.SessionLifetime != 2*time.Hour || configuration.HeartbeatInterval != 7*time.Second {
		t.Fatalf("unexpected config: %+v", configuration)
	}
}

func TestLoadDefaultsHeartbeatIntervalForExistingConfiguration(t *testing.T) {
	path := filepath.Join(t.TempDir(), "server.toml")
	if err := os.WriteFile(path, []byte(`
listen_address = "127.0.0.1:8080"
database_path = "./var/server.db"
`), 0o600); err != nil {
		t.Fatal(err)
	}
	configuration, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if configuration.HeartbeatInterval != 3*time.Second {
		t.Fatalf("default heartbeat interval=%s", configuration.HeartbeatInterval)
	}
}

func TestApplyEnvironmentOverrides(t *testing.T) {
	configuration := Config{
		ListenAddress: "0.0.0.0:8080", DatabasePath: "/var/lib/tempo-server/server.db",
		AdminOrigin: "same-host", SessionLifetime: time.Hour, OnlineTimeout: time.Minute,
		HeartbeatInterval: 3 * time.Second,
	}
	environment := map[string]string{
		"TEMPO_ADMIN_ORIGIN":       "https://admin.example",
		"TEMPO_SECURE_COOKIES":     "true",
		"TEMPO_HEARTBEAT_INTERVAL": "9s",
	}
	updated, err := ApplyEnvironmentOverrides(configuration, func(name string) string { return environment[name] })
	if err != nil {
		t.Fatal(err)
	}
	if updated.AdminOrigin != "https://admin.example" || !updated.SecureCookies || updated.HeartbeatInterval != 9*time.Second {
		t.Fatalf("environment overrides not applied: %+v", updated)
	}
	environment["TEMPO_SECURE_COOKIES"] = "not-a-boolean"
	if _, err := ApplyEnvironmentOverrides(configuration, func(name string) string { return environment[name] }); err == nil {
		t.Fatal("invalid secure cookie override accepted")
	}
	environment["TEMPO_SECURE_COOKIES"] = "true"
	environment["TEMPO_HEARTBEAT_INTERVAL"] = "500ms"
	if _, err := ApplyEnvironmentOverrides(configuration, func(name string) string { return environment[name] }); err == nil {
		t.Fatal("unsafe heartbeat interval override accepted")
	}
}
