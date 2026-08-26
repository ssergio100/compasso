package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/ssergio100/compasso/agent/localauth"
	"github.com/ssergio100/compasso/server/config"
	"github.com/ssergio100/compasso/server/storage"
	"github.com/ssergio100/compasso/server/web"
)

func main() {
	configPath := flag.String("config", "server/config.toml", "server configuration path")
	flag.Parse()
	logger := log.New(os.Stdout, "tempo-server: ", log.LstdFlags|log.LUTC)
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
	settings, err = config.ApplyEnvironmentOverrides(settings, os.Getenv)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(settings.DatabasePath), 0o700); err != nil {
		return fmt.Errorf("create server state directory: %w", err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	store, err := storage.Open(ctx, settings.DatabasePath)
	if err != nil {
		return err
	}
	defer store.Close()
	bootstrapLogin := os.Getenv("TEMPO_ADMIN_LOGIN")
	bootstrapPassword, err := loadBootstrapAdministratorPassword(
		os.Getenv("TEMPO_ADMIN_PASSWORD"), os.Getenv("TEMPO_ADMIN_PASSWORD_FILE"),
	)
	if err != nil {
		return err
	}
	if bootstrapLogin != "" && bootstrapPassword != "" {
		hash, err := localauth.HashPassword(bootstrapPassword, localauth.DefaultArgon2Params)
		if err != nil {
			return fmt.Errorf("hash bootstrap administrator password: %w", err)
		}
		created, err := store.BootstrapAdmin(ctx, bootstrapLogin, hash, time.Now())
		if err != nil {
			return err
		}
		if created {
			logger.Printf("initial administrator created login=%s", bootstrapLogin)
		}
	}
	application, err := web.New(
		store, settings.SecureCookies, settings.SessionLifetime, settings.OnlineTimeout,
		settings.HeartbeatInterval, settings.AdminOrigin,
	)
	if err != nil {
		return err
	}
	application.StartOfflineDetector(ctx)
	server := &http.Server{
		Addr: settings.ListenAddress, Handler: application,
		ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second,
		// WriteTimeout stays disabled so authenticated SSE streams stay open;
		// every other endpoint is short-lived and bounded by the caller.
		WriteTimeout: 0, IdleTimeout: 60 * time.Second,
		MaxHeaderBytes: 32 << 10,
	}
	shutdownDone := make(chan error, 1)
	go func() {
		<-ctx.Done()
		shutdownContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		shutdownDone <- server.Shutdown(shutdownContext)
	}()
	logger.Printf("listening address=%s secure_cookies=%t", settings.ListenAddress, settings.SecureCookies)
	err = server.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return <-shutdownDone
}

func loadBootstrapAdministratorPassword(environmentPassword, passwordFilePath string) (string, error) {
	if environmentPassword != "" && passwordFilePath != "" {
		return "", errors.New("configure only one of TEMPO_ADMIN_PASSWORD or TEMPO_ADMIN_PASSWORD_FILE")
	}
	if passwordFilePath == "" {
		return environmentPassword, nil
	}
	fileInformation, err := os.Stat(passwordFilePath)
	if err != nil {
		return "", fmt.Errorf("inspect bootstrap administrator password file: %w", err)
	}
	if fileInformation.IsDir() || fileInformation.Size() > 4096 {
		return "", errors.New("bootstrap administrator password file must be a regular file of at most 4096 bytes")
	}
	contents, err := os.ReadFile(passwordFilePath)
	if err != nil {
		return "", fmt.Errorf("read bootstrap administrator password file: %w", err)
	}
	return strings.TrimRight(string(contents), "\r\n"), nil
}
