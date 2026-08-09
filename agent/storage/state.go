package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"time"
)

const localDateLayout = "2006-01-02"

// DailyUsage is the last durable usage checkpoint for a local calendar date.
type DailyUsage struct {
	LocalDate    string
	SecondsUsed  int64
	CheckpointAt time.Time
}

// Bonus is a durable local or remote increment that policy replacement cannot
// remove.
type Bonus struct {
	UUID      string
	LocalDate string
	Seconds   int64
	Origin    string
	CreatedAt time.Time
}

// PendingEvent is an idempotent operation waiting for server acknowledgement.
type PendingEvent struct {
	UUID        string
	Kind        string
	PayloadJSON string
	CreatedAt   time.Time
	RetryCount  int
}

// EnqueueEvent durably adds a generic idempotent event for a future heartbeat.
func (s *Store) EnqueueEvent(ctx context.Context, event PendingEvent) error {
	if err := validateEvent(event); err != nil {
		return err
	}
	if _, err := s.db.ExecContext(ctx, `
		INSERT OR IGNORE INTO pending_event(uuid, kind, payload_json, created_at, retry_count)
		VALUES (?, ?, ?, ?, ?)`,
		event.UUID, event.Kind, event.PayloadJSON, formatTime(event.CreatedAt), event.RetryCount,
	); err != nil {
		return fmt.Errorf("enqueue pending event: %w", err)
	}
	return nil
}

// CheckpointUsage saves an absolute consumed-seconds value. Values are
// monotonic for a given day so an old checkpoint cannot return time to a user.
func (s *Store) CheckpointUsage(ctx context.Context, usage DailyUsage) error {
	if err := validateLocalDate(usage.LocalDate); err != nil {
		return err
	}
	if usage.SecondsUsed < 0 || usage.CheckpointAt.IsZero() {
		return errors.New("usage must be non-negative and checkpoint time must be set")
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO daily_usage(local_date, seconds_used, checkpoint_at) VALUES (?, ?, ?)
		ON CONFLICT(local_date) DO UPDATE SET
			seconds_used = MAX(daily_usage.seconds_used, excluded.seconds_used),
			checkpoint_at = CASE
				WHEN excluded.seconds_used >= daily_usage.seconds_used THEN excluded.checkpoint_at
				ELSE daily_usage.checkpoint_at
			END`,
		usage.LocalDate, usage.SecondsUsed, formatTime(usage.CheckpointAt),
	)
	if err != nil {
		return fmt.Errorf("checkpoint daily usage: %w", err)
	}
	return nil
}

// LoadDailyUsage returns zero usage when the date has no checkpoint yet.
func (s *Store) LoadDailyUsage(ctx context.Context, localDate string) (DailyUsage, error) {
	if err := validateLocalDate(localDate); err != nil {
		return DailyUsage{}, err
	}
	var usage DailyUsage
	var checkpoint string
	err := s.db.QueryRowContext(ctx, `
		SELECT local_date, seconds_used, checkpoint_at FROM daily_usage WHERE local_date = ?`, localDate,
	).Scan(&usage.LocalDate, &usage.SecondsUsed, &checkpoint)
	if errors.Is(err, sql.ErrNoRows) {
		return DailyUsage{LocalDate: localDate}, nil
	}
	if err != nil {
		return DailyUsage{}, fmt.Errorf("load daily usage: %w", err)
	}
	usage.CheckpointAt, err = parseTime(checkpoint)
	if err != nil {
		return DailyUsage{}, err
	}
	return usage, nil
}

// AddBonusWithEvent atomically stores a bonus and its upload event. Repeating
// the same UUID is harmless and does not duplicate either record.
func (s *Store) AddBonusWithEvent(ctx context.Context, bonus Bonus, event PendingEvent) error {
	if err := validateBonus(bonus); err != nil {
		return err
	}
	if err := validateEvent(event); err != nil {
		return err
	}
	if bonus.UUID != event.UUID {
		return errors.New("bonus and event UUID must match")
	}
	s.sessionStateMu.Lock()
	defer s.sessionStateMu.Unlock()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin bonus transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	bonusResult, err := tx.ExecContext(ctx, `
		INSERT OR IGNORE INTO bonus(uuid, local_date, seconds, origin, created_at)
		VALUES (?, ?, ?, ?, ?)`,
		bonus.UUID, bonus.LocalDate, bonus.Seconds, bonus.Origin, formatTime(bonus.CreatedAt),
	)
	if err != nil {
		return fmt.Errorf("store bonus: %w", err)
	}
	bonusInserted, err := bonusResult.RowsAffected()
	if err != nil {
		return fmt.Errorf("inspect stored bonus: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT OR IGNORE INTO pending_event(uuid, kind, payload_json, created_at, retry_count)
		VALUES (?, ?, ?, ?, ?)`,
		event.UUID, event.Kind, event.PayloadJSON, formatTime(event.CreatedAt), event.RetryCount,
	); err != nil {
		return fmt.Errorf("store pending bonus event: %w", err)
	}
	if bonusInserted != 0 {
		if _, err := tx.ExecContext(ctx, `
			UPDATE confirmed_session_state
			SET remaining_seconds=remaining_seconds+?
			WHERE singleton_id=1 AND local_date=?`, bonus.Seconds, bonus.LocalDate); err != nil {
			return fmt.Errorf("add local bonus to confirmed balance: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit bonus transaction: %w", err)
	}
	if err := s.refreshConfirmedSessionStateLocked(ctx); err != nil {
		return err
	}
	return nil
}

// TotalBonusSeconds calculates all bonus seconds for one local date.
func (s *Store) TotalBonusSeconds(ctx context.Context, localDate string) (int64, error) {
	if err := validateLocalDate(localDate); err != nil {
		return 0, err
	}
	var total int64
	if err := s.db.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(seconds), 0) FROM bonus WHERE local_date = ?`, localDate,
	).Scan(&total); err != nil {
		return 0, fmt.Errorf("sum bonuses: %w", err)
	}
	return total, nil
}

// PendingEvents returns events in creation order for the next heartbeat.
func (s *Store) PendingEvents(ctx context.Context, limit int) ([]PendingEvent, error) {
	if limit <= 0 {
		return nil, errors.New("event limit must be positive")
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT uuid, kind, payload_json, created_at, retry_count
		FROM pending_event ORDER BY created_at, uuid LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("load pending events: %w", err)
	}
	defer rows.Close()
	var events []PendingEvent
	for rows.Next() {
		var event PendingEvent
		var createdAt string
		if err := rows.Scan(&event.UUID, &event.Kind, &event.PayloadJSON, &createdAt, &event.RetryCount); err != nil {
			return nil, fmt.Errorf("scan pending event: %w", err)
		}
		event.CreatedAt, err = parseTime(createdAt)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate pending events: %w", err)
	}
	return events, nil
}

// AcknowledgeEvent removes an event only after server confirmation.
func (s *Store) AcknowledgeEvent(ctx context.Context, uuid string) error {
	if uuid == "" {
		return errors.New("event UUID cannot be empty")
	}
	s.sessionStateMu.Lock()
	defer s.sessionStateMu.Unlock()
	if _, err := s.db.ExecContext(ctx, `DELETE FROM pending_event WHERE uuid = ?`, uuid); err != nil {
		return fmt.Errorf("acknowledge pending event: %w", err)
	}
	return nil
}

// IncrementEventRetry records an unsuccessful upload attempt without removing
// the event from the durable queue.
func (s *Store) IncrementEventRetry(ctx context.Context, uuid string) error {
	if uuid == "" {
		return errors.New("event UUID cannot be empty")
	}
	result, err := s.db.ExecContext(ctx,
		`UPDATE pending_event SET retry_count = retry_count + 1 WHERE uuid = ?`, uuid,
	)
	if err != nil {
		return fmt.Errorf("increment event retry: %w", err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read event retry result: %w", err)
	}
	if changed == 0 {
		return fmt.Errorf("pending event %s not found", uuid)
	}
	return nil
}

// UsageTracker bounds crash loss by checkpointing after at most checkpointEvery
// accumulated seconds. The caller supplies elapsed monotonic durations.
type UsageTracker struct {
	mu              sync.Mutex
	store           *Store
	localDate       string
	seconds         int64
	uncheckpointed  time.Duration
	checkpointEvery time.Duration
}

// NewUsageTracker reconstructs a day's counter from its last durable value.
func NewUsageTracker(ctx context.Context, store *Store, localDate string, checkpointEvery time.Duration) (*UsageTracker, error) {
	if checkpointEvery <= 0 {
		return nil, errors.New("checkpoint interval must be positive")
	}
	usage, err := store.LoadDailyUsage(ctx, localDate)
	if err != nil {
		return nil, err
	}
	return &UsageTracker{
		store: store, localDate: localDate, seconds: usage.SecondsUsed, checkpointEvery: checkpointEvery,
	}, nil
}

// Add records allowed-use elapsed time and checkpoints when the configured
// maximum unpersisted interval is reached.
func (t *UsageTracker) Add(ctx context.Context, elapsed time.Duration, now time.Time) error {
	if elapsed < 0 || elapsed%time.Second != 0 || now.IsZero() {
		return errors.New("elapsed must be non-negative whole seconds and now must be set")
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	t.seconds += int64(elapsed / time.Second)
	t.uncheckpointed += elapsed
	if t.uncheckpointed < t.checkpointEvery {
		return nil
	}
	if err := t.store.CheckpointUsage(ctx, DailyUsage{
		LocalDate: t.localDate, SecondsUsed: t.seconds, CheckpointAt: now,
	}); err != nil {
		return err
	}
	t.uncheckpointed = 0
	return nil
}

// Flush persists all currently accounted usage during an orderly shutdown.
func (t *UsageTracker) Flush(ctx context.Context, now time.Time) error {
	if now.IsZero() {
		return errors.New("flush time must be set")
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if err := t.store.CheckpointUsage(ctx, DailyUsage{
		LocalDate: t.localDate, SecondsUsed: t.seconds, CheckpointAt: now,
	}); err != nil {
		return err
	}
	t.uncheckpointed = 0
	return nil
}

// Seconds returns the in-memory total, including time not checkpointed yet.
func (t *UsageTracker) Seconds() int64 {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.seconds
}

// EnsureAtLeast reconciles a server-confirmed absolute counter after a local
// crash may have lost an uncheckpointed tail. It never reduces local usage.
func (t *UsageTracker) EnsureAtLeast(ctx context.Context, seconds int64, now time.Time) error {
	if seconds < 0 || now.IsZero() {
		return errors.New("minimum usage must be non-negative and time must be set")
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if seconds <= t.seconds {
		return nil
	}
	t.seconds = seconds
	t.uncheckpointed = 0
	return t.store.CheckpointUsage(ctx, DailyUsage{
		LocalDate: t.localDate, SecondsUsed: seconds, CheckpointAt: now,
	})
}

func validateLocalDate(localDate string) error {
	parsed, err := time.Parse(localDateLayout, localDate)
	if err != nil || parsed.Format(localDateLayout) != localDate {
		return fmt.Errorf("invalid local date %q", localDate)
	}
	return nil
}

func validateBonus(bonus Bonus) error {
	if bonus.UUID == "" || bonus.Seconds <= 0 || bonus.Origin == "" || bonus.CreatedAt.IsZero() {
		return errors.New("bonus requires UUID, positive seconds, origin and creation time")
	}
	return validateLocalDate(bonus.LocalDate)
}

func validateEvent(event PendingEvent) error {
	if event.UUID == "" || event.Kind == "" || event.PayloadJSON == "" || event.CreatedAt.IsZero() || event.RetryCount < 0 {
		return errors.New("event requires UUID, kind, payload, creation time and non-negative retry count")
	}
	return nil
}
