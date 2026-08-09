package localapi

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sergio/compasso/agent/localauth"
	"github.com/sergio/compasso/agent/storage"
)

func TestBonusAPIMapsPasswordErrorsAndReturnsEventUUID(t *testing.T) {
	ctx := context.Background()
	store, err := storage.Open(ctx, filepath.Join(t.TempDir(), "agent.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	params := localauth.Argon2Params{Memory: 8 * 1024, Iterations: 1, Parallelism: 1, SaltLength: 8, KeyLength: 16}
	verifier, err := localauth.HashPassword("secret", params)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, time.August, 10, 12, 0, 0, 0, time.Local)
	if err := store.ReplacePolicy(ctx, storage.PolicySnapshot{Revision: 1, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	service, _ := localauth.NewService(store)
	api, _ := newBonusAPI(service)
	api.now = func() time.Time { return now }
	if _, dbusErr := api.AddLocalBonus("anything", 900); dbusErr == nil || !strings.HasSuffix(dbusErr.Name, "PasswordNotConfigured") {
		t.Fatalf("missing password D-Bus error=%v", dbusErr)
	}
	if err := store.ReplacePolicy(ctx, storage.PolicySnapshot{
		Revision: 2, LocalPasswordVerifier: verifier, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	if _, dbusErr := api.AddLocalBonus("wrong", 900); dbusErr == nil || !strings.HasSuffix(dbusErr.Name, "InvalidPassword") {
		t.Fatalf("wrong password D-Bus error=%v", dbusErr)
	}
	api.now = func() time.Time { return now.Add(2 * time.Second) }
	uuid, dbusErr := api.AddLocalBonus("secret", 900)
	if dbusErr != nil || uuid == "" {
		t.Fatalf("correct password uuid=%q error=%v", uuid, dbusErr)
	}
}
