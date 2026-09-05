package main

import (
	"database/sql"
	"fmt"
	"sync"
)

// ChatSettingsManager mengelola konfigurasi dan preferensi per grup/chat (seperti pemilihan kelas)
type ChatSettingsManager struct {
	db    *sql.DB
	mu    sync.RWMutex
	cache map[string]string // key: scope_jid, value: class_id (uppercase)
}

// NewChatSettingsManager menginisialisasi tabel chat_settings pada SQLite dan memuat cache ke memori
func NewChatSettingsManager(db *sql.DB) (*ChatSettingsManager, error) {
	if db == nil {
		return nil, fmt.Errorf("koneksi database tidak boleh nil")
	}

	query := `
	CREATE TABLE IF NOT EXISTS chat_settings (
		scope_jid TEXT PRIMARY KEY,
		class_id TEXT NOT NULL,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	_, err := db.Exec(query)
	if err != nil {
		return nil, fmt.Errorf("gagal membuat tabel chat_settings: %w", err)
	}

	csm := &ChatSettingsManager{
		db:    db,
		cache: make(map[string]string),
	}

	// Warm-up cache dari database saat startup agar pembacaan saat pesan masuk berkecepatan nanosekon (O(1))
	if err := csm.loadCache(); err != nil {
		return nil, fmt.Errorf("gagal memuat data chat_settings: %w", err)
	}

	return csm, nil
}

// loadCache membaca seluruh setelan dari database ke memori
func (csm *ChatSettingsManager) loadCache() error {
	rows, err := csm.db.Query(`SELECT scope_jid, class_id FROM chat_settings`)
	if err != nil {
		return err
	}
	defer rows.Close()

	csm.mu.Lock()
	defer csm.mu.Unlock()

	for rows.Next() {
		var scopeJID, classID string
		if err := rows.Scan(&scopeJID, &classID); err == nil {
			csm.cache[scopeJID] = NormalizeClassID(classID)
		}
	}

	return rows.Err()
}

// GetClass mengambil ID kelas yang diatur untuk suatu chat/grup (mengembalikan string kosong jika belum diatur)
func (csm *ChatSettingsManager) GetClass(scopeJID string) string {
	csm.mu.RLock()
	defer csm.mu.RUnlock()

	return csm.cache[scopeJID]
}

// SetClass menyimpan atau memperbarui pilihan kelas untuk suatu chat/grup
func (csm *ChatSettingsManager) SetClass(scopeJID string, rawClassID string) error {
	classID := NormalizeClassID(rawClassID)
	if classID == "" {
		return fmt.Errorf("nama kelas tidak boleh kosong")
	}

	query := `
	INSERT OR REPLACE INTO chat_settings (scope_jid, class_id, updated_at)
	VALUES (?, ?, CURRENT_TIMESTAMP);`

	_, err := csm.db.Exec(query, scopeJID, classID)
	if err != nil {
		return fmt.Errorf("gagal menyimpan setelan kelas ke database: %w", err)
	}

	// Perbarui in-memory cache
	csm.mu.Lock()
	csm.cache[scopeJID] = classID
	csm.mu.Unlock()

	return nil
}

// DeleteClass menghapus pilihan kelas untuk chat tertentu (kembali ke default)
func (csm *ChatSettingsManager) DeleteClass(scopeJID string) error {
	_, err := csm.db.Exec(`DELETE FROM chat_settings WHERE scope_jid = ?`, scopeJID)
	if err != nil {
		return fmt.Errorf("gagal menghapus setelan kelas: %w", err)
	}

	csm.mu.Lock()
	delete(csm.cache, scopeJID)
	csm.mu.Unlock()

	return nil
}

// CountSettings mengembalikan jumlah grup/chat yang telah melakukan binding kelas
func (csm *ChatSettingsManager) CountSettings() int {
	csm.mu.RLock()
	defer csm.mu.RUnlock()

	return len(csm.cache)
}
