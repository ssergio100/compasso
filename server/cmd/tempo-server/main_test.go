package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadBootstrapAdministratorPasswordFromFile(t *testing.T) {
	passwordFilePath := filepath.Join(t.TempDir(), "admin-password")
	if err := os.WriteFile(passwordFilePath, []byte("domestic-secret\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	password, err := loadBootstrapAdministratorPassword("", passwordFilePath)
	if err != nil || password != "domestic-secret" {
		t.Fatalf("password file result length=%d err=%v", len(password), err)
	}
	if _, err := loadBootstrapAdministratorPassword("environment-secret", passwordFilePath); err == nil {
		t.Fatal("simultaneous password sources were accepted")
	}
}
