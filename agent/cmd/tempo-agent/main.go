package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"os/user"
	"path/filepath"
	"syscall"

	"github.com/sergio/compasso/agent/config"
	"github.com/sergio/compasso/agent/daemon"
	"github.com/sergio/compasso/agent/session"
	"github.com/sergio/compasso/agent/storage"
)

func main() {
	configPath := flag.String("config", "/etc/tempo-agent/config.toml", "path to the agent configuration")
	flag.Parse()

	logger := log.New(os.Stdout, "tempo-agent: ", log.LstdFlags|log.LUTC)
	if err := run(*configPath, logger); err != nil {
		logger.Printf("fatal: %v", err)
		os.Exit(1)
	}
}

func run(configPath string, logger *log.Logger) error {
	settings, err := config.Load(configPath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(settings.DatabasePath), 0o700); err != nil {
		return fmt.Errorf("create state directory: %w", err)
	}
	controlledAccount, err := user.Lookup(settings.ControlledUser)
	if err != nil {
		return fmt.Errorf("find controlled Linux account %q: %w", settings.ControlledUser, err)
	}
	if controlledAccount.Uid == "0" {
		return fmt.Errorf("controlled Linux account %q cannot be root", settings.ControlledUser)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	store, err := storage.Open(ctx, settings.DatabasePath)
	if err != nil {
		return err
	}
	defer store.Close()
	logind, err := session.NewLogind(settings.LoginctlPath)
	if err != nil {
		return err
	}
	agent, err := daemon.New(store, logind, settings.ControlledUser, settings.CheckpointInterval)
	if err != nil {
		return err
	}
	logger.Printf("starting controlled_user=%s database=%s", settings.ControlledUser, settings.DatabasePath)
	return agent.Run(ctx, settings.TickInterval, logger)
}
