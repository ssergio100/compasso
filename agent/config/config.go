// Package config loads the small, installation-time configuration of tempo-agent.
package config

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config contains settings that belong to the local installation. Policies and
// other remotely managed values are deliberately not part of this file.
type Config struct {
	DatabasePath       string
	ControlledUser     string
	TickInterval       time.Duration
	CheckpointInterval time.Duration
	LoginctlPath       string
}

// Load reads the flat TOML subset used by the agent configuration. Keeping the
// parser deliberately small avoids adding a runtime dependency for five scalar
// installation settings.
func Load(path string) (Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return Config{}, fmt.Errorf("open configuration: %w", err)
	}
	defer file.Close()

	values := make(map[string]string)
	scanner := bufio.NewScanner(file)
	for lineNumber := 1; scanner.Scan(); lineNumber++ {
		line := strings.TrimSpace(stripComment(scanner.Text()))
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "[") {
			return Config{}, fmt.Errorf("configuration line %d: sections are not supported", lineNumber)
		}
		key, rawValue, found := strings.Cut(line, "=")
		if !found {
			return Config{}, fmt.Errorf("configuration line %d: expected key = value", lineNumber)
		}
		key = strings.TrimSpace(key)
		value, err := strconv.Unquote(strings.TrimSpace(rawValue))
		if err != nil {
			return Config{}, fmt.Errorf("configuration line %d: value must be a quoted string: %w", lineNumber, err)
		}
		values[key] = value
	}
	if err := scanner.Err(); err != nil {
		return Config{}, fmt.Errorf("read configuration: %w", err)
	}

	config := Config{
		DatabasePath:       values["database_path"],
		ControlledUser:     values["controlled_user"],
		TickInterval:       time.Second,
		CheckpointInterval: 5 * time.Second,
		LoginctlPath:       "/usr/bin/loginctl",
	}
	if value := values["tick_interval"]; value != "" {
		config.TickInterval, err = time.ParseDuration(value)
		if err != nil {
			return Config{}, fmt.Errorf("parse tick_interval: %w", err)
		}
	}
	if value := values["checkpoint_interval"]; value != "" {
		config.CheckpointInterval, err = time.ParseDuration(value)
		if err != nil {
			return Config{}, fmt.Errorf("parse checkpoint_interval: %w", err)
		}
	}
	if value := values["loginctl_path"]; value != "" {
		config.LoginctlPath = value
	}
	if err := config.Validate(); err != nil {
		return Config{}, err
	}
	return config, nil
}

// Validate rejects unsafe or incomplete local settings.
func (c Config) Validate() error {
	if c.DatabasePath == "" {
		return errors.New("database_path cannot be empty")
	}
	if c.ControlledUser == "" || strings.ContainsAny(c.ControlledUser, " \t\r\n/") {
		return errors.New("controlled_user must be a single Linux account name")
	}
	if c.TickInterval <= 0 {
		return errors.New("tick_interval must be positive")
	}
	if c.CheckpointInterval <= 0 {
		return errors.New("checkpoint_interval must be positive")
	}
	if c.LoginctlPath == "" {
		return errors.New("loginctl_path cannot be empty")
	}
	return nil
}

func stripComment(line string) string {
	inString := false
	escaped := false
	for index, character := range line {
		if escaped {
			escaped = false
			continue
		}
		if character == '\\' && inString {
			escaped = true
			continue
		}
		if character == '"' {
			inString = !inString
			continue
		}
		if character == '#' && !inString {
			return line[:index]
		}
	}
	return line
}
