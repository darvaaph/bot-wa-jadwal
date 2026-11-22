package auth

import (
	"context"
	"testing"
	"time"
)

func TestRemediation_LoginBlockExpiresFromLastFailureNotFirst(t *testing.T) {
	_, db, clock := newTestService(t)
	for i := 0; i < 5; i++ {
		at := clock.now.Add(time.Duration(i-4) * 3 * time.Minute)
		if _, err := db.Exec(`INSERT INTO login_attempts(identity_hash,source_hash,outcome,attempted_at) VALUES('identity','source','FAILURE',?)`, formatTime(at)); err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < 250; i++ {
		if _, err := db.Exec(`INSERT INTO login_attempts(identity_hash,source_hash,outcome,attempted_at) VALUES('identity','source','BLOCKED',?)`, formatTime(clock.now)); err != nil {
			t.Fatal(err)
		}
	}
	blocked, _, err := LoginBlocked(context.Background(), db, "identity", "source", clock.now.Add(14*time.Minute))
	if err != nil || !blocked {
		t.Fatal("block ended early", blocked, err)
	}
	blocked, _, err = LoginBlocked(context.Background(), db, "identity", "source", clock.now.Add(15*time.Minute))
	if err != nil || blocked {
		t.Fatal("block did not expire", blocked, err)
	}
	if _, err := db.Exec(`INSERT INTO login_attempts(identity_hash,source_hash,outcome,attempted_at) VALUES('identity','source','SUCCESS',?)`, formatTime(clock.now)); err != nil {
		t.Fatal(err)
	}
	blocked, _, err = LoginBlocked(context.Background(), db, "identity", "source", clock.now)
	if err != nil || blocked {
		t.Fatal("success did not reset", blocked, err)
	}
}
