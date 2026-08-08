package localauth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/sergio/compasso/agent/storage"
)

var (
	ErrInvalidPassword = errors.New("invalid local password")
	ErrRateLimited     = errors.New("too many password attempts")
)

// GrantResult identifies the durable bonus created by one successful request.
type GrantResult struct {
	UUID         string
	BonusSeconds int64
	TotalSeconds int64
}

// Service serializes password attempts and bonus writes for the local agent.
type Service struct {
	mu            sync.Mutex
	store         *storage.Store
	failed        int
	nextAttemptAt time.Time
}

func NewService(store *storage.Store) (*Service, error) {
	if store == nil {
		return nil, errors.New("store is required")
	}
	return &Service{store: store}, nil
}

// Grant verifies the current policy verifier and atomically stores a local
// bonus with its future synchronization event.
func (s *Service) Grant(ctx context.Context, password string, seconds int64, now time.Time) (GrantResult, error) {
	if now.IsZero() {
		return GrantResult{}, errors.New("current time is required")
	}
	if seconds < 60 || seconds > int64((12*time.Hour)/time.Second) {
		return GrantResult{}, errors.New("bonus must be between 1 minute and 12 hours")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if now.Before(s.nextAttemptAt) {
		return GrantResult{}, fmt.Errorf("%w: retry after %s", ErrRateLimited, s.nextAttemptAt.Format(time.RFC3339))
	}

	policy, err := s.store.LoadPolicy(ctx)
	if err != nil {
		return GrantResult{}, fmt.Errorf("load local password verifier: %w", err)
	}
	if policy.LocalPasswordVerifier == "" {
		return GrantResult{}, errors.New("local password is not configured")
	}
	valid, err := VerifyPassword(password, policy.LocalPasswordVerifier)
	if err != nil {
		return GrantResult{}, fmt.Errorf("verify local password: %w", err)
	}
	if !valid {
		s.failed++
		s.nextAttemptAt = now.Add(backoffForFailure(s.failed))
		return GrantResult{}, ErrInvalidPassword
	}
	s.failed = 0
	s.nextAttemptAt = time.Time{}

	uuid, err := newUUID()
	if err != nil {
		return GrantResult{}, err
	}
	localDate := now.Format("2006-01-02")
	payload, err := json.Marshal(struct {
		LocalDate string `json:"local_date"`
		Seconds   int64  `json:"seconds"`
		Origin    string `json:"origin"`
	}{LocalDate: localDate, Seconds: seconds, Origin: "local"})
	if err != nil {
		return GrantResult{}, fmt.Errorf("encode bonus event: %w", err)
	}
	bonus := storage.Bonus{
		UUID: uuid, LocalDate: localDate, Seconds: seconds, Origin: "local", CreatedAt: now,
	}
	event := storage.PendingEvent{
		UUID: uuid, Kind: "bonus_added", PayloadJSON: string(payload), CreatedAt: now,
	}
	if err := s.store.AddBonusWithEvent(ctx, bonus, event); err != nil {
		return GrantResult{}, fmt.Errorf("store local bonus: %w", err)
	}
	total, err := s.store.TotalBonusSeconds(ctx, localDate)
	if err != nil {
		return GrantResult{}, fmt.Errorf("read total local bonus: %w", err)
	}
	return GrantResult{UUID: uuid, BonusSeconds: seconds, TotalSeconds: total}, nil
}

func (s *Service) FailedAttempts() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.failed
}

func backoffForFailure(failures int) time.Duration {
	delays := [...]time.Duration{2 * time.Second, 5 * time.Second, 15 * time.Second, time.Minute, 5 * time.Minute}
	if failures <= 0 {
		return 0
	}
	if failures > len(delays) {
		return delays[len(delays)-1]
	}
	return delays[failures-1]
}

func newUUID() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generate bonus UUID: %w", err)
	}
	bytes[6] = (bytes[6] & 0x0f) | 0x40
	bytes[8] = (bytes[8] & 0x3f) | 0x80
	encoded := hex.EncodeToString(bytes)
	return encoded[0:8] + "-" + encoded[8:12] + "-" + encoded[12:16] + "-" + encoded[16:20] + "-" + encoded[20:32], nil
}
