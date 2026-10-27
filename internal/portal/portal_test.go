package portal

import (
	"testing"
	"time"
)

func TestService_RateLimitCleanup_NoMapLeak(t *testing.T) {
	svc := NewService(nil)
	now := time.Now()

	classID := int64(10)
	source := "192.168.1.100"
	key := limiterKey(classID, source)

	// Record a failure
	svc.recordFailure(classID, source, now.Add(-20*time.Minute))

	svc.mu.Lock()
	if len(svc.failures[key]) != 1 {
		t.Fatalf("diharapkan ada 1 failure record")
	}
	svc.mu.Unlock()

	// Check rate limit 20 minutes later (attemptWindow is 15 minutes)
	err := svc.checkRateLimit(classID, source, now)
	if err != nil {
		t.Fatalf("checkRateLimit tidak boleh error: %v", err)
	}

	// Pastikan key dihapus dari map dan tidak bocor di memory
	svc.mu.Lock()
	if _, exists := svc.failures[key]; exists {
		t.Fatalf("key seharusnya dihapus dari svc.failures setelah semua attempt kedaluwarsa")
	}
	svc.mu.Unlock()
}

func TestService_BlockPeriodExpiration_DeletesBlockKey(t *testing.T) {
	svc := NewService(nil)
	now := time.Now()

	classID := int64(10)
	source := "192.168.1.101"
	key := limiterKey(classID, source)

	// Set block in the past
	svc.mu.Lock()
	svc.blocks[key] = now.Add(-1 * time.Minute)
	svc.mu.Unlock()

	// checkRateLimit should clear the expired block
	err := svc.checkRateLimit(classID, source, now)
	if err != nil {
		t.Fatalf("checkRateLimit seharusnya tidak mengembalikan error untuk block yang kedaluwarsa: %v", err)
	}

	svc.mu.Lock()
	if _, exists := svc.blocks[key]; exists {
		t.Fatalf("key seharusnya dihapus dari svc.blocks setelah block period kedaluwarsa")
	}
	svc.mu.Unlock()
}
