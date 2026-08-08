package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sergio/compasso/agent/storage"
)

func TestRunFailsOpenWithoutConfiguration(t *testing.T) {
	var stderr bytes.Buffer
	getenv := func(key string) string {
		if key == "PAM_USER" {
			return "child"
		}
		return ""
	}
	code := run([]string{"-config", filepath.Join(t.TempDir(), "missing.toml")}, getenv, &stderr, time.Now())
	if code != exitAllowed || stderr.Len() == 0 {
		t.Fatalf("code=%d stderr=%q", code, stderr.String())
	}
}

func TestRunIgnoresNonAccountPAMCalls(t *testing.T) {
	getenv := func(key string) string {
		if key == "PAM_TYPE" {
			return "auth"
		}
		return ""
	}
	if code := run(nil, getenv, os.Stderr, time.Now()); code != exitAllowed {
		t.Fatalf("code=%d, want allowed", code)
	}
}

func TestRunReturnsDeniedForBlockedControlledUser(t *testing.T) {
	directory := t.TempDir()
	databasePath := filepath.Join(directory, "agent.db")
	configPath := filepath.Join(directory, "config.toml")
	now := time.Date(2026, time.August, 10, 14, 0, 0, 0, time.Local)
	store, err := storage.Open(context.Background(), databasePath)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.ReplacePolicy(context.Background(), storage.PolicySnapshot{
		Revision: 1, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	configuration := fmt.Sprintf("database_path = %q\ncontrolled_user = %q\n", databasePath, "child")
	if err := os.WriteFile(configPath, []byte(configuration), 0o600); err != nil {
		t.Fatal(err)
	}
	getenv := func(key string) string {
		switch key {
		case "PAM_USER":
			return "child"
		case "PAM_TYPE":
			return "account"
		default:
			return ""
		}
	}
	var stderr bytes.Buffer
	if code := run([]string{"-config", configPath}, getenv, &stderr, now); code != exitDenied {
		t.Fatalf("code=%d stderr=%q, want denied", code, stderr.String())
	}
}

func TestEmergencyBypassAllowsBlockedControlledUser(t *testing.T) {
	bypassFilePath := filepath.Join(t.TempDir(), "compasso-pam-bypass")
	if err := os.WriteFile(bypassFilePath, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	getenv := func(key string) string {
		if key == "PAM_USER" {
			return "child"
		}
		return ""
	}
	missingConfigurationPath := filepath.Join(t.TempDir(), "missing.toml")
	if code := runWithEmergencyBypass(
		[]string{"-config", missingConfigurationPath}, getenv, os.Stderr, time.Now(), bypassFilePath,
	); code != exitAllowed {
		t.Fatalf("code=%d, want emergency login allowed", code)
	}
}
