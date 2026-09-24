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

// NewRateLimiter membuat instance RateLimiter baru dengan cooldown yang ditentukan.
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

	// Bersihkan record usang secara berkala
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

// Reset menghapus riwayat cooldown untuk pengirim tertentu.
func (rl *RateLimiter) Reset(key string) {
	if rl == nil {
		return
	}
	rl.mu.Lock()
	defer rl.mu.Unlock()
	delete(rl.records, key)
}

// Clear menghapus seluruh riwayat cooldown.
func (rl *RateLimiter) Clear() {
	if rl == nil {
		return
	}
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.records = make(map[string]time.Time)
}

// cleanupLocked menghapus entri yang sudah melewati 2x masa cooldown untuk menghemat memori.
func (rl *RateLimiter) cleanupLocked(now time.Time) {
	cutoff := rl.cooldown * 2
	for k, t := range rl.records {
		if now.Sub(t) > cutoff {
			delete(rl.records, k)
		}
	}
	rl.lastCleanup = now
}

// IsCommandMessage memeriksa apakah pesan teks berpotensi sebagai perintah bot:
// - Di grup chat: Pesan WAJIB diawali simbol prefix (!, /, atau #).
// - Di pesan pribadi (DM): Setiap teks dianggap berpotensi perintah.
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
