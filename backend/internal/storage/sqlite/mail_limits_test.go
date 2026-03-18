package sqlite

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"linuxdospace/backend/internal/storage"
	"linuxdospace/backend/internal/timeutil"
)

// TestEnqueueMailDeliveryBatchConcurrentRespectsForwardDailyLimit verifies that
// two concurrent queue writes cannot both consume the final remaining unit of
// the hidden per-user daily forwarding allowance.
func TestEnqueueMailDeliveryBatchConcurrentRespectsForwardDailyLimit(t *testing.T) {
	ctx := context.Background()
	databasePath := filepath.Join(t.TempDir(), "linuxdospace-mail-forward-limit.sqlite")

	firstStore := newTestStoreAtPath(t, databasePath)
	secondStore := newTestStoreAtPath(t, databasePath)
	for _, store := range []*Store{firstStore, secondStore} {
		if _, err := store.db.ExecContext(ctx, `PRAGMA busy_timeout = 5000`); err != nil {
			t.Fatalf("set sqlite busy_timeout: %v", err)
		}
	}

	user := newTestUser(t, ctx, firstStore, "mail-forward-limit")
	now := time.Now().UTC()
	usageDate := timeutil.ShanghaiDayKey(now)
	if _, err := firstStore.db.ExecContext(ctx, `
INSERT INTO mail_forward_daily_usage (
    user_id,
    usage_date,
    used_count,
    created_at,
    updated_at
) VALUES (?, ?, ?, ?, ?)
`,
		user.ID,
		usageDate,
		999,
		formatTime(now),
		formatTime(now),
	); err != nil {
		t.Fatalf("seed existing forward daily usage: %v", err)
	}

	enqueueInput := EnqueueMailDeliveryBatchInput{
		OriginalEnvelopeFrom: "sender@example.com",
		RawMessage:           []byte("Subject: Test\r\n\r\nbody"),
		MaxAttempts:          1,
		QueuedAt:             now,
		Groups: []storage.EnqueueMailDeliveryGroupInput{{
			OriginalRecipients: []string{"alias@linuxdo.space"},
			TargetRecipients:   []string{"target@example.com"},
			OwnerUserIDs:       []int64{user.ID},
		}},
	}

	type enqueueResult struct {
		err error
	}

	start := make(chan struct{})
	results := make(chan enqueueResult, 2)
	enqueue := func(store *Store) {
		<-start
		_, err := store.EnqueueMailDeliveryBatch(ctx, enqueueInput)
		results <- enqueueResult{err: err}
	}

	go enqueue(firstStore)
	go enqueue(secondStore)
	close(start)

	successCount := 0
	rejectionCount := 0
	for resultIndex := 0; resultIndex < 2; resultIndex++ {
		result := <-results
		switch {
		case result.err == nil:
			successCount++
		case errors.Is(result.err, storage.ErrMailForwardDailyLimitExceeded), isSQLiteBusyError(result.err):
			rejectionCount++
		default:
			t.Fatalf("expected one success and one safe rejection, got %v", result.err)
		}
	}

	if successCount != 1 {
		t.Fatalf("expected exactly one successful enqueue, got %d", successCount)
	}
	if rejectionCount != 1 {
		t.Fatalf("expected exactly one forward-daily-limit rejection, got %d", rejectionCount)
	}

	row := firstStore.db.QueryRowContext(ctx, `
SELECT used_count
FROM mail_forward_daily_usage
WHERE user_id = ? AND usage_date = ?
`, user.ID, usageDate)

	var usedCount int64
	if err := row.Scan(&usedCount); err != nil {
		t.Fatalf("reload forward daily usage: %v", err)
	}
	if usedCount != 1000 {
		t.Fatalf("expected forward daily usage to stop at 1000, got %d", usedCount)
	}
}

// TestConsumeEmailCatchAllConcurrentRespectsDailyLimit verifies that two
// concurrent catch-all reservations cannot both consume the only remaining unit
// under the configured daily cap.
func TestConsumeEmailCatchAllConcurrentRespectsDailyLimit(t *testing.T) {
	ctx := context.Background()
	databasePath := filepath.Join(t.TempDir(), "linuxdospace-catch-all-limit.sqlite")

	firstStore := newTestStoreAtPath(t, databasePath)
	secondStore := newTestStoreAtPath(t, databasePath)
	for _, store := range []*Store{firstStore, secondStore} {
		if _, err := store.db.ExecContext(ctx, `PRAGMA busy_timeout = 5000`); err != nil {
			t.Fatalf("set sqlite busy_timeout: %v", err)
		}
	}

	user := newTestUser(t, ctx, firstStore, "catch-all-limit")
	if _, err := firstStore.UpsertEmailCatchAllAccess(ctx, UpsertEmailCatchAllAccessInput{
		UserID:         user.ID,
		RemainingCount: 10,
	}); err != nil {
		t.Fatalf("seed catch-all access: %v", err)
	}

	now := time.Now().UTC()
	consumeInput := ConsumeEmailCatchAllInput{
		UserID:            user.ID,
		Count:             1,
		DefaultDailyLimit: 1,
		Now:               now,
	}

	type consumeResult struct {
		err error
	}

	start := make(chan struct{})
	results := make(chan consumeResult, 2)
	consume := func(store *Store) {
		<-start
		_, err := store.ConsumeEmailCatchAll(ctx, consumeInput)
		results <- consumeResult{err: err}
	}

	go consume(firstStore)
	go consume(secondStore)
	close(start)

	successCount := 0
	rejectionCount := 0
	for resultIndex := 0; resultIndex < 2; resultIndex++ {
		result := <-results
		switch {
		case result.err == nil:
			successCount++
		case errors.Is(result.err, storage.ErrEmailCatchAllDailyLimitExceeded), isSQLiteBusyError(result.err):
			rejectionCount++
		default:
			t.Fatalf("expected one success and one safe catch-all rejection, got %v", result.err)
		}
	}

	if successCount != 1 {
		t.Fatalf("expected exactly one successful catch-all consume, got %d", successCount)
	}
	if rejectionCount != 1 {
		t.Fatalf("expected exactly one catch-all daily-limit rejection, got %d", rejectionCount)
	}

	usageDate := timeutil.ShanghaiDayKey(now)
	usage, err := firstStore.GetEmailCatchAllDailyUsage(ctx, user.ID, usageDate)
	if err != nil {
		t.Fatalf("reload catch-all daily usage: %v", err)
	}
	if usage.UsedCount != 1 {
		t.Fatalf("expected catch-all daily usage to stop at 1, got %d", usage.UsedCount)
	}

	access, err := firstStore.GetEmailCatchAllAccessByUser(ctx, user.ID)
	if err != nil {
		t.Fatalf("reload catch-all access: %v", err)
	}
	if access.RemainingCount != 9 {
		t.Fatalf("expected remaining catch-all count to drop once to 9, got %d", access.RemainingCount)
	}
}

// isSQLiteBusyError recognizes SQLite's file-lock contention error. The
// PostgreSQL production path does not use this branch, but the SQLite fallback
// may reject the losing writer with SQLITE_BUSY while still preserving the
// stronger invariant this test cares about: no quota oversell.
func isSQLiteBusyError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "database is locked")
}
