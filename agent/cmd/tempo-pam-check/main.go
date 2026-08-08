package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/sergio/compasso/agent/config"
	"github.com/sergio/compasso/agent/pamgate"
	"github.com/sergio/compasso/agent/storage"
)

const (
	exitAllowed = 0
	exitDenied  = 1
)

func main() {
	os.Exit(run(os.Args[1:], os.Getenv, os.Stderr, time.Now()))
}

// run fails open on operational errors so a broken configuration cannot make
// the graphical login permanently unusable. A valid blocking policy is the
// only condition that returns exitDenied.
func run(args []string, getenv func(string) string, stderr io.Writer, now time.Time) int {
	flags := flag.NewFlagSet("tempo-pam-check", flag.ContinueOnError)
	flags.SetOutput(stderr)
	configPath := flags.String("config", "/etc/tempo-agent/config.toml", "agent configuration path")
	if err := flags.Parse(args); err != nil {
		return failOpen(stderr, err)
	}
	if pamType := getenv("PAM_TYPE"); pamType != "" && pamType != "account" {
		return exitAllowed
	}
	username := getenv("PAM_USER")
	if username == "" {
		return failOpen(stderr, fmt.Errorf("PAM_USER is empty"))
	}
	settings, err := config.Load(*configPath)
	if err != nil {
		return failOpen(stderr, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	store, err := storage.OpenReadOnly(ctx, settings.DatabasePath)
	if err != nil {
		return failOpen(stderr, err)
	}
	defer store.Close()
	result, err := pamgate.Check(ctx, store, settings.ControlledUser, username, now)
	if err != nil {
		return failOpen(stderr, err)
	}
	if !result.Allowed {
		return exitDenied
	}
	return exitAllowed
}

func failOpen(stderr io.Writer, err error) int {
	fmt.Fprintf(stderr, "tempo-pam-check: allowing login after internal error: %v\n", err)
	return exitAllowed
}
