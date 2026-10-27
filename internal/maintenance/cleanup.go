package maintenance

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// PurgeStaleRecords membersihkan data usang dari database v1:
// 1. security_attempts lebih lama dari 30 hari
// 2. user_sessions yang telah dicabut atau kedaluwarsa absolut lebih lama dari 30 hari
// 3. portal_sessions yang telah dicabut atau kedaluwarsa lebih lama dari 7 hari
func PurgeStaleRecords(ctx context.Context, db *sql.DB) error {
	if db == nil {
		return nil
	}
	release, allowed := EnterMutation()
	if !allowed {
		return nil
	}
	defer release()

	// 1. Purge security_attempts (> 30 hari)
	cutoffAttempts := time.Now().UTC().Add(-30 * 24 * time.Hour).Format(time.RFC3339Nano)
	if _, err := db.ExecContext(ctx, `DELETE FROM security_attempts WHERE attempted_at < ?;`, cutoffAttempts); err != nil {
		return fmt.Errorf("purge security_attempts: %w", err)
	}

	// 2. Purge user_sessions (> 30 hari)
	if _, err := db.ExecContext(ctx, `
		DELETE FROM user_sessions
		WHERE (revoked_at IS NOT NULL AND datetime(revoked_at) < datetime('now', '-30 days'))
		   OR (datetime(absolute_expires_at) < datetime('now', '-30 days'));
	`); err != nil {
		return fmt.Errorf("purge user_sessions: %w", err)
	}

	// 3. Purge portal_sessions (> 7 hari)
	if _, err := db.ExecContext(ctx, `
		DELETE FROM portal_sessions
		WHERE (revoked_at IS NOT NULL AND datetime(revoked_at) < datetime('now', '-7 days'))
		   OR (datetime(expires_at) < datetime('now', '-7 days'));
	`); err != nil {
		return fmt.Errorf("purge portal_sessions: %w", err)
	}

	return nil
}

// StartMaintenanceWorker menjalankan pembersihan berkala di latar belakang setiap interval tertentu.
func StartMaintenanceWorker(ctx context.Context, db *sql.DB, interval time.Duration) func() {
	if interval <= 0 {
		interval = 24 * time.Hour
	}
	done := make(chan struct{})

	go func() {
		// Jalankan sekali saat startup
		_ = PurgeStaleRecords(ctx, db)

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-done:
				return
			case <-ctx.Done():
				return
			case <-ticker.C:
				_ = PurgeStaleRecords(ctx, db)
			}
		}
	}()

	return func() {
		close(done)
	}
}
