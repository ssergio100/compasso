// Package config loads installation settings for tempo-server.
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

type Config struct {
	ListenAddress     string
	DatabasePath      string
	AdminOrigin       string
	SecureCookies     bool
	SessionLifetime   time.Duration
	OnlineTimeout     time.Duration
	HeartbeatInterval time.Duration
}

func Load(path string) (Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return Config{}, fmt.Errorf("open server configuration: %w", err)
	}
	defer file.Close()
	values := make(map[string]string)
	scanner := bufio.NewScanner(file)
	for lineNumber := 1; scanner.Scan(); lineNumber++ {
		line := strings.TrimSpace(stripComment(scanner.Text()))
		if line == "" {
			continue
		}
		key, raw, found := strings.Cut(line, "=")
		if !found {
			return Config{}, fmt.Errorf("configuration line %d: expected key = value", lineNumber)
		}
		key, raw = strings.TrimSpace(key), strings.TrimSpace(raw)
		if raw == "true" || raw == "false" {
			values[key] = raw
			continue
		}
		value, err := strconv.Unquote(raw)
		if err != nil {
			return Config{}, fmt.Errorf("configuration line %d: invalid value: %w", lineNumber, err)
		}
		values[key] = value
	}
	if err := scanner.Err(); err != nil {
		return Config{}, fmt.Errorf("read server configuration: %w", err)
	}
	configuration := Config{
		ListenAddress: values["listen_address"], DatabasePath: values["database_path"],
		AdminOrigin: values["admin_origin"], SessionLifetime: 8 * time.Hour,
		OnlineTimeout: 60 * time.Second, HeartbeatInterval: 3 * time.Second,
	}
	if value := values["secure_cookies"]; value != "" {
		configuration.SecureCookies, err = strconv.ParseBool(value)
		if err != nil {
			return Config{}, fmt.Errorf("parse secure_cookies: %w", err)
		}
	}
	if value := values["session_lifetime"]; value != "" {
		configuration.SessionLifetime, err = time.ParseDuration(value)
		if err != nil {
			return Config{}, fmt.Errorf("parse session_lifetime: %w", err)
		}
	}
	if value := values["online_timeout"]; value != "" {
		configuration.OnlineTimeout, err = time.ParseDuration(value)
		if err != nil {
			return Config{}, fmt.Errorf("parse online_timeout: %w", err)
		}
	}
	if value := values["heartbeat_interval"]; value != "" {
		configuration.HeartbeatInterval, err = time.ParseDuration(value)
		if err != nil {
			return Config{}, fmt.Errorf("parse heartbeat_interval: %w", err)
		}
	}
	if err := configuration.Validate(); err != nil {
		return Config{}, err
	}
	return configuration, nil
}

// ApplyEnvironmentOverrides applies deployment settings that can vary without
// rebuilding the server image. An empty value preserves the file setting.
func ApplyEnvironmentOverrides(configuration Config, environmentValue func(string) string) (Config, error) {
	if environmentValue == nil {
		return configuration, errors.New("environment lookup is required")
	}
	if adminOrigin := strings.TrimSpace(environmentValue("TEMPO_ADMIN_ORIGIN")); adminOrigin != "" {
		configuration.AdminOrigin = adminOrigin
	}
	if secureCookies := strings.TrimSpace(environmentValue("TEMPO_SECURE_COOKIES")); secureCookies != "" {
		value, err := strconv.ParseBool(secureCookies)
		if err != nil {
			return Config{}, fmt.Errorf("parse TEMPO_SECURE_COOKIES: %w", err)
		}
		configuration.SecureCookies = value
	}
	if heartbeatInterval := strings.TrimSpace(environmentValue("TEMPO_HEARTBEAT_INTERVAL")); heartbeatInterval != "" {
		value, err := time.ParseDuration(heartbeatInterval)
		if err != nil {
			return Config{}, fmt.Errorf("parse TEMPO_HEARTBEAT_INTERVAL: %w", err)
		}
		configuration.HeartbeatInterval = value
	}
	if err := configuration.Validate(); err != nil {
		return Config{}, err
	}
	return configuration, nil
}

func (c Config) Validate() error {
	if c.ListenAddress == "" || c.DatabasePath == "" {
		return errors.New("listen_address and database_path are required")
	}
	if c.SessionLifetime < time.Minute || c.SessionLifetime > 7*24*time.Hour {
		return errors.New("session_lifetime must be between 1 minute and 7 days")
	}
	if c.OnlineTimeout < 10*time.Second || c.OnlineTimeout > 10*time.Minute {
		return errors.New("online_timeout must be between 10 seconds and 10 minutes")
	}
	if c.HeartbeatInterval < time.Second || c.HeartbeatInterval > 10*time.Minute || c.HeartbeatInterval%time.Second != 0 {
		return errors.New("heartbeat_interval must be a whole number of seconds between 1 second and 10 minutes")
	}
	return nil
}

func stripComment(line string) string {
	inString := false
	for index, character := range line {
		if character == '"' {
			inString = !inString
		}
		if character == '#' && !inString {
			return line[:index]
		}
	}
	return line
}
