package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"os/user"
	"path/filepath"
	"syscall"

	"github.com/sergio/compasso/agent/config"
	"github.com/sergio/compasso/agent/daemon"
	"github.com/sergio/compasso/agent/localapi"
	"github.com/sergio/compasso/agent/localauth"
	"github.com/sergio/compasso/agent/session"
	"github.com/sergio/compasso/agent/storage"
	"github.com/sergio/compasso/agent/syncclient"
)

func main() {
	configPath := flag.String("config", "/etc/tempo-agent/config.toml", "path to the agent configuration")
	validateConfigurationOnly := flag.Bool("check-config", false, "validate configuration and exit")
	flag.Parse()

	logger := log.New(os.Stdout, "tempo-agent: ", log.LstdFlags|log.LUTC)
	if *validateConfigurationOnly {
		if _, err := config.Load(*configPath); err != nil {
			logger.Printf("configuration invalid: %v", err)
			os.Exit(1)
		}
		logger.Printf("configuration valid")
		return
	}
	if err := run(*configPath, logger); err != nil {
		logger.Printf("fatal: %v", err)
		os.Exit(1)
	}
}

func run(configPath string, logger *log.Logger) error {
	// Protect databases, WAL files and any future state created by the daemon,
	// including when it is started manually instead of through systemd.
	syscall.Umask(0o077)
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
	localBonus, err := localauth.NewService(store)
	if err != nil {
		return err
	}
	localServer, err := localapi.ExportSystem(localBonus)
	if err != nil {
		logger.Printf("local D-Bus API unavailable: %v", err)
	} else {
		defer localServer.Close()
		logger.Printf("local D-Bus API ready name=%s", localapi.BusName)
	}
	logger.Printf("starting controlled_user=%s database=%s", settings.ControlledUser, settings.DatabasePath)
	if settings.SyncEnabled() {
		synchronizer, err := syncclient.New(store, &http.Client{Timeout: settings.HTTPTimeout}, syncclient.Config{
			ServerURL: settings.ServerURL, DeviceID: settings.DeviceID,
			DeviceToken: settings.DeviceToken, HeartbeatInterval: settings.HeartbeatInterval,
		})
		if err != nil {
			return err
		}
		go func() {
			if err := synchronizer.Run(ctx, logger); err != nil {
				logger.Printf("synchronization stopped: %v", err)
			}
		}()
		logger.Printf("synchronization enabled server=%s device_id=%s", settings.ServerURL, settings.DeviceID)
	} else {
		logger.Printf("synchronization disabled; local policy remains available offline")
	}
	return agent.Run(ctx, settings.TickInterval, logger)
}
