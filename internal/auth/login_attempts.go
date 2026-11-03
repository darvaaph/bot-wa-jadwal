package auth

import (
	"context"
	"database/sql"
	"sync"
	"time"
)

var loginAttemptMu sync.Mutex

// LockLoginAttempts serializes check/password/record for the single-process runtime.
func LockLoginAttempts() func() { loginAttemptMu.Lock(); return loginAttemptMu.Unlock }

func LoginBlocked(ctx context.Context, db *sql.DB, identityHash, sourceHash string, now time.Time) (bool, time.Duration, error) {
	rows, err := db.QueryContext(ctx, `SELECT outcome,attempted_at FROM login_attempts WHERE identity_hash=? AND source_hash=? AND outcome IN ('FAILURE','SUCCESS') AND julianday(attempted_at)>=julianday(?) ORDER BY id DESC`, identityHash, sourceHash, formatTime(now.Add(-loginWindow-loginBlockPeriod)))
	if err != nil {
		return false, 0, err
	}
	defer rows.Close()
	var last time.Time
	failures := 0
	for rows.Next() {
		var outcome, raw string
		if err := rows.Scan(&outcome, &raw); err != nil {
			return false, 0, err
		}
		if outcome == "SUCCESS" {
			break
		}
		t, err := parseTime(raw)
		if err != nil {
			return false, 0, err
		}
		if last.IsZero() {
			last = t
		}
		if last.Sub(t) > loginWindow {
			break
		}
		failures++
	}
	if err := rows.Err(); err != nil {
		return false, 0, err
	}
	remaining := last.Add(loginBlockPeriod).Sub(now)
	return failures >= maxLoginFailures && remaining > 0, remaining, nil
}
