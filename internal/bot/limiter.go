package bot

import (
	"strings"
	"sync"
	"time"
)

// RateLimiter mengelola batas frekuensi pengiriman perintah per pengirim untuk mencegah spam / flood.
type RateLimiter struct {
	mu              sync.Mutex
	records         map[string]time.Time
	cooldown        time.Duration
	cleanupInterval time.Duration
	lastCleanup     time.Time
}

func NewRateLimiter(cooldown time.Duration) *RateLimiter {
	if cooldown <= 0 {
		cooldown = 2 * time.Second
	}
	return &RateLimiter{
		records:         make(map[string]time.Time),
		cooldown:        cooldown,
		cleanupInterval: 5 * time.Minute,
		lastCleanup:     time.Now(),
	}
}

// Allow memeriksa apakah pengirim (key) diizinkan mengeksekusi perintah pada waktu saat ini.
// Mengembalikan true jika diizinkan (cooldown telah lewat), atau false jika masih dalam cooldown.
func (rl *RateLimiter) Allow(key string) bool {
	return rl.AllowAt(key, time.Now())
}

// AllowAt memeriksa izin pada waktu tertentu (mempermudah unit testing deterministik).
func (rl *RateLimiter) AllowAt(key string, now time.Time) bool {
	if rl == nil || rl.cooldown <= 0 {
		return true
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	if now.Sub(rl.lastCleanup) >= rl.cleanupInterval {
		rl.cleanupLocked(now)
	}

	lastTime, exists := rl.records[key]
	if exists && now.Sub(lastTime) < rl.cooldown {
		return false
	}

	rl.records[key] = now
	return true
}

// Remaining mengembalikan sisa durasi cooldown untuk pengirim tertentu.
// Mengembalikan 0 jika pengirim tidak sedang dalam masa cooldown.
func (rl *RateLimiter) Remaining(key string) time.Duration {
	return rl.RemainingAt(key, time.Now())
}

// RemainingAt mengembalikan sisa durasi cooldown pada waktu tertentu.
func (rl *RateLimiter) RemainingAt(key string, now time.Time) time.Duration {
	if rl == nil || rl.cooldown <= 0 {
		return 0
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	lastTime, exists := rl.records[key]
	if !exists {
		return 0
	}

	elapsed := now.Sub(lastTime)
	if elapsed < rl.cooldown {
		return rl.cooldown - elapsed
	}
	return 0
}

func (rl *RateLimiter) Reset(key string) {
	if rl == nil {
		return
	}
	rl.mu.Lock()
	defer rl.mu.Unlock()
	delete(rl.records, key)
}

func (rl *RateLimiter) Clear() {
	if rl == nil {
		return
	}
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.records = make(map[string]time.Time)
}

// cleanupLocked membuang entri setelah dua kali durasi cooldown untuk membatasi penggunaan memori.
func (rl *RateLimiter) cleanupLocked(now time.Time) {
	cutoff := rl.cooldown * 2
	for k, t := range rl.records {
		if now.Sub(t) > cutoff {
			delete(rl.records, k)
		}
	}
	rl.lastCleanup = now
}

// IsCommandMessage mewajibkan prefix !, /, atau # di grup; semua teks DM dianggap command.
func IsCommandMessage(msgText string, isGroup bool) bool {
	clean := strings.TrimSpace(msgText)
	if clean == "" {
		return false
	}
	if !isGroup {
		return true
	}
	return strings.HasPrefix(clean, "!") || strings.HasPrefix(clean, "/") || strings.HasPrefix(clean, "#")
}
