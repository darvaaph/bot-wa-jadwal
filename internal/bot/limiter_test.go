package bot

import (
	"sync"
	"testing"
	"time"
)

func TestRateLimiter_Allow(t *testing.T) {
	cooldown := 2 * time.Second
	rl := NewRateLimiter(cooldown)
	baseTime := time.Date(2026, 9, 19, 10, 0, 0, 0, time.UTC)

	userA := "62812345678"
	userB := "62898765432"

	if !rl.AllowAt(userA, baseTime) {
		t.Errorf("Expected first call for userA to be allowed")
	}

	if rl.AllowAt(userA, baseTime.Add(500*time.Millisecond)) {
		t.Errorf("Expected second call within cooldown for userA to be blocked")
	}

	if !rl.AllowAt(userB, baseTime.Add(500*time.Millisecond)) {
		t.Errorf("Expected call for userB to be allowed independently")
	}

	if !rl.AllowAt(userA, baseTime.Add(2100*time.Millisecond)) {
		t.Errorf("Expected call for userA after cooldown to be allowed")
	}

	if rl.AllowAt(userA, baseTime.Add(2200*time.Millisecond)) {
		t.Errorf("Expected call for userA shortly after new window to be blocked")
	}
}

func TestRateLimiter_Remaining(t *testing.T) {
	cooldown := 3 * time.Second
	rl := NewRateLimiter(cooldown)
	baseTime := time.Date(2026, 9, 19, 10, 0, 0, 0, time.UTC)
	user := "62811111111"

	if rem := rl.RemainingAt(user, baseTime); rem != 0 {
		t.Errorf("Expected remaining 0 for unseen user, got %v", rem)
	}

	rl.AllowAt(user, baseTime)

	rem := rl.RemainingAt(user, baseTime.Add(1*time.Second))
	if rem != 2*time.Second {
		t.Errorf("Expected remaining 2s, got %v", rem)
	}

	remExpired := rl.RemainingAt(user, baseTime.Add(3*time.Second))
	if remExpired != 0 {
		t.Errorf("Expected remaining 0 after cooldown expired, got %v", remExpired)
	}
}

func TestRateLimiter_ResetAndClear(t *testing.T) {
	cooldown := 5 * time.Second
	rl := NewRateLimiter(cooldown)
	baseTime := time.Date(2026, 9, 19, 10, 0, 0, 0, time.UTC)

	userA := "user_a"
	userB := "user_b"

	rl.AllowAt(userA, baseTime)
	rl.AllowAt(userB, baseTime)

	rl.Reset(userA)

	if !rl.AllowAt(userA, baseTime.Add(500*time.Millisecond)) {
		t.Errorf("Expected userA to be allowed immediately after Reset")
	}
	if rl.AllowAt(userB, baseTime.Add(500*time.Millisecond)) {
		t.Errorf("Expected userB to still be blocked")
	}

	rl.Clear()
	if !rl.AllowAt(userB, baseTime.Add(600*time.Millisecond)) {
		t.Errorf("Expected userB to be allowed after Clear")
	}
}

func TestRateLimiter_Cleanup(t *testing.T) {
	cooldown := 2 * time.Second
	rl := NewRateLimiter(cooldown)
	rl.cleanupInterval = 1 * time.Second // Percepat interval cleanup untuk testing

	baseTime := time.Date(2026, 9, 19, 10, 0, 0, 0, time.UTC)
	rl.lastCleanup = baseTime
	rl.AllowAt("old_user", baseTime)

	// Setelah 10 detik, memanggil AllowAt untuk pengguna baru akan memicu cleanupLocked
	now := baseTime.Add(10 * time.Second)
	rl.AllowAt("new_user", now)

	rl.mu.Lock()
	_, oldExists := rl.records["old_user"]
	_, newExists := rl.records["new_user"]
	rl.mu.Unlock()

	if oldExists {
		t.Errorf("Expected old_user to be deleted during cleanup")
	}
	if !newExists {
		t.Errorf("Expected new_user to remain in records")
	}
}

func TestRateLimiter_Concurrency(t *testing.T) {
	rl := NewRateLimiter(500 * time.Millisecond)
	var wg sync.WaitGroup

	numGoroutines := 50
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			user := "concurrency_user"
			if id%2 == 0 {
				user = "alternate_user"
			}
			rl.Allow(user)
			rl.Remaining(user)
		}(i)
	}

	wg.Wait()
}

func TestIsCommandMessage(t *testing.T) {
	tests := []struct {
		name     string
		msg      string
		isGroup  bool
		expected bool
	}{
		{"Empty string group", "", true, false},
		{"Empty string DM", "", false, false},
		{"Whitespace group", "   ", true, false},
		{"Whitespace DM", "   \n\t", false, false},

		{"Group command with !", "!jadwal", true, true},
		{"Group command with /", "/tugas", true, true},
		{"Group command with #", "#pindah", true, true},
		{"Group normal chat", "halo kawan-kawan", true, false},
		{"Group message mentioning keyword", "jadwal besok apa ya?", true, false},

		{"DM command with !", "!jadwal", false, true},
		{"DM command without prefix", "jadwal", false, true},
		{"DM single word", "senin", false, true},
		{"DM conversational", "pagi min", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsCommandMessage(tt.msg, tt.isGroup)
			if result != tt.expected {
				t.Errorf("IsCommandMessage(%q, %v) = %v; want %v", tt.msg, tt.isGroup, result, tt.expected)
			}
		})
	}
}
