package pamgate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallAndUninstallRestoreExactPAMService(t *testing.T) {
	directory := t.TempDir()
	pamPath := filepath.Join(directory, "gdm-password")
	helperPath := filepath.Join(directory, "tempo-pam-check")
	configPath := filepath.Join(directory, "config.toml")
	original := []byte("#%PAM-1.0\n@include common-auth\n@include common-account\n")
	if err := os.WriteFile(pamPath, original, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(helperPath, []byte("helper"), 0o755); err != nil {
		t.Fatal(err)
	}
	options := InstallOptions{
		PAMServicePath: pamPath, HelperPath: helperPath,
		ConfigPath: configPath, ControlledUser: "child",
	}
	if err := Install(options); err != nil {
		t.Fatal(err)
	}
	installed, err := os.ReadFile(pamPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{
		beginMarker,
		"pam_succeed_if.so quiet user != child",
		"pam_exec.so quiet type=account " + helperPath + " -config " + configPath,
		endMarker,
	} {
		if !strings.Contains(string(installed), expected) {
			t.Fatalf("installed PAM service does not contain %q:\n%s", expected, installed)
		}
	}
	if _, err := os.Stat(BackupPath(pamPath)); err != nil {
		t.Fatalf("backup not created: %v", err)
	}
	if err := Install(options); err == nil {
		t.Fatal("duplicate installation should fail")
	}
	if err := Uninstall(pamPath); err != nil {
		t.Fatal(err)
	}
	restored, err := os.ReadFile(pamPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(restored) != string(original) {
		t.Fatalf("restored PAM service differs\ngot:  %q\nwant: %q", restored, original)
	}
	if _, err := os.Stat(BackupPath(pamPath)); !os.IsNotExist(err) {
		t.Fatalf("backup still exists after restoration: %v", err)
	}
}

func TestInstallRejectsUnsafeUserBeforeWritingBackup(t *testing.T) {
	directory := t.TempDir()
	pamPath := filepath.Join(directory, "gdm-password")
	helperPath := filepath.Join(directory, "tempo-pam-check")
	if err := os.WriteFile(pamPath, []byte("#%PAM-1.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(helperPath, []byte("helper"), 0o755); err != nil {
		t.Fatal(err)
	}
	err := Install(InstallOptions{
		PAMServicePath: pamPath, HelperPath: helperPath,
		ConfigPath: filepath.Join(directory, "config.toml"), ControlledUser: "child\naccount sufficient pam_permit.so",
	})
	if err == nil {
		t.Fatal("unsafe controlled user should fail")
	}
	if _, err := os.Stat(BackupPath(pamPath)); !os.IsNotExist(err) {
		t.Fatalf("backup created for invalid installation: %v", err)
	}
}
