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
	ListenAddress   string
	DatabasePath    string
	AssetsDirectory string
	SecureCookies   bool
	SessionLifetime time.Duration
	OnlineTimeout   time.Duration
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
		AssetsDirectory: valueOrDefault(values["assets_directory"], "./server/web"),
		SessionLifetime: 8 * time.Hour, OnlineTimeout: 60 * time.Second,
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
	if err := configuration.Validate(); err != nil {
		return Config{}, err
	}
	return configuration, nil
}

func (c Config) Validate() error {
	if c.ListenAddress == "" || c.DatabasePath == "" || c.AssetsDirectory == "" {
		return errors.New("listen_address, database_path and assets_directory are required")
	}
	if c.SessionLifetime < time.Minute || c.SessionLifetime > 7*24*time.Hour {
		return errors.New("session_lifetime must be between 1 minute and 7 days")
	}
	if c.OnlineTimeout < 10*time.Second || c.OnlineTimeout > 10*time.Minute {
		return errors.New("online_timeout must be between 10 seconds and 10 minutes")
	}
	return nil
}

func valueOrDefault(value, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
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
