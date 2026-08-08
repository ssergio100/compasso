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
	"syscall"
	"time"

	"github.com/sergio/compasso/agent/localauth"
	"github.com/sergio/compasso/server/config"
	"github.com/sergio/compasso/server/storage"
	"github.com/sergio/compasso/server/web"
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
	bootstrapLogin, bootstrapPassword := os.Getenv("TEMPO_ADMIN_LOGIN"), os.Getenv("TEMPO_ADMIN_PASSWORD")
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
	application, err := web.New(store, settings.SecureCookies, settings.SessionLifetime, settings.OnlineTimeout)
	if err != nil {
		return err
	}
	server := &http.Server{
		Addr: settings.ListenAddress, Handler: application,
		ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second,
		WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second,
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
