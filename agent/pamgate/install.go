package pamgate

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	beginMarker = "# BEGIN COMPASSO PAM GATE"
	endMarker   = "# END COMPASSO PAM GATE"
)

// InstallOptions identifies the one graphical PAM service being protected.
type InstallOptions struct {
	PAMServicePath string
	HelperPath     string
	ConfigPath     string
	ControlledUser string
}

// Install backs up the original PAM service and appends a small account gate.
// The helper must be installed before this function is called.
func Install(options InstallOptions) error {
	if err := validateInstallOptions(options); err != nil {
		return err
	}
	if info, err := os.Stat(options.HelperPath); err != nil {
		return fmt.Errorf("inspect PAM helper: %w", err)
	} else if info.IsDir() || info.Mode().Perm()&0o111 == 0 {
		return fmt.Errorf("PAM helper %s is not executable", options.HelperPath)
	}
	original, err := os.ReadFile(options.PAMServicePath)
	if err != nil {
		return fmt.Errorf("read PAM service: %w", err)
	}
	if strings.Contains(string(original), beginMarker) || strings.Contains(string(original), endMarker) {
		return errors.New("Compasso PAM gate is already installed")
	}
	info, err := os.Stat(options.PAMServicePath)
	if err != nil {
		return fmt.Errorf("inspect PAM service: %w", err)
	}
	backupPath := BackupPath(options.PAMServicePath)
	if err := writeExclusive(backupPath, original, info.Mode().Perm()); err != nil {
		return fmt.Errorf("create PAM backup: %w", err)
	}

	updated := append(append([]byte(nil), original...), renderGate(options, len(original) > 0 && original[len(original)-1] != '\n')...)
	if err := replaceFile(options.PAMServicePath, updated, info.Mode().Perm()); err != nil {
		_ = os.Remove(backupPath)
		return fmt.Errorf("install PAM gate: %w", err)
	}
	return nil
}

// Uninstall restores the exact PAM service saved before installation.
func Uninstall(pamServicePath string) error {
	if !filepath.IsAbs(pamServicePath) {
		return errors.New("PAM service path must be absolute")
	}
	backupPath := BackupPath(pamServicePath)
	backup, err := os.ReadFile(backupPath)
	if err != nil {
		return fmt.Errorf("read PAM backup: %w", err)
	}
	info, err := os.Stat(backupPath)
	if err != nil {
		return fmt.Errorf("inspect PAM backup: %w", err)
	}
	if err := replaceFile(pamServicePath, backup, info.Mode().Perm()); err != nil {
		return fmt.Errorf("restore PAM service: %w", err)
	}
	if err := os.Remove(backupPath); err != nil {
		return fmt.Errorf("remove restored PAM backup: %w", err)
	}
	return nil
}

// BackupPath returns the deterministic backup used for safe removal.
func BackupPath(pamServicePath string) string {
	return pamServicePath + ".compasso.bak"
}

func renderGate(options InstallOptions, needsLeadingNewline bool) []byte {
	prefix := ""
	// Keep the original final line intact even when the distribution file does
	// not end with a newline.
	if needsLeadingNewline {
		prefix = "\n"
	}
	return []byte(fmt.Sprintf(
		"%s%s\naccount [success=1 default=ignore] pam_succeed_if.so quiet user != %s\n"+
			"account requisite pam_exec.so quiet type=account %s -config %s\n%s\n",
		prefix, beginMarker, options.ControlledUser, options.HelperPath, options.ConfigPath, endMarker,
	))
}

func validateInstallOptions(options InstallOptions) error {
	if !filepath.IsAbs(options.PAMServicePath) || !filepath.IsAbs(options.HelperPath) || !filepath.IsAbs(options.ConfigPath) {
		return errors.New("PAM service, helper and configuration paths must be absolute")
	}
	if strings.ContainsAny(options.HelperPath+options.ConfigPath, " \t\r\n") {
		return errors.New("helper and configuration paths cannot contain whitespace")
	}
	if !safePAMUser(options.ControlledUser) {
		return fmt.Errorf("controlled user %q is not safe for a PAM rule", options.ControlledUser)
	}
	return nil
}

func safePAMUser(username string) bool {
	if username == "" || username[0] == '-' {
		return false
	}
	for _, character := range username {
		if (character < 'a' || character > 'z') && (character < 'A' || character > 'Z') &&
			(character < '0' || character > '9') && character != '_' && character != '-' && character != '.' {
			return false
		}
	}
	return true
}

func writeExclusive(path string, contents []byte, mode os.FileMode) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return err
	}
	if _, err := file.Write(contents); err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		return err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		return err
	}
	return file.Close()
}

func replaceFile(path string, contents []byte, mode os.FileMode) error {
	directory := filepath.Dir(path)
	temporary, err := os.CreateTemp(directory, ".compasso-pam-*")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(mode); err != nil {
		_ = temporary.Close()
		return err
	}
	if _, err := temporary.Write(contents); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(temporaryPath, path)
}
