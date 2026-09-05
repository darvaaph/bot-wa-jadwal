package main

import (
	"database/sql"
	"fmt"
	"strings"
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

// HandleCommand memproses perintah pengaturan kelas (!setkelas, !pilihkelas, !daftarkelas, !kelas, !resetkelas)
func (csm *ChatSettingsManager) HandleCommand(
	chatJID string,
	isGroup bool,
	senderJID string,
	isAdmin bool,
	rawMsg string,
	classMgr *ClassManager,
) string {
	if classMgr == nil {
		return "⚠️ Pengelola jadwal kelas belum siap. Silakan coba sesaat lagi."
	}

	fields := strings.Fields(strings.TrimSpace(rawMsg))
	if len(fields) == 0 {
		return ""
	}

	cmd := strings.ToLower(fields[0])

	// 1. Perintah Melihat Daftar Kelas (!daftarkelas / !kelas)
	if cmd == "!daftarkelas" || cmd == "/daftarkelas" || cmd == "#daftarkelas" ||
		cmd == "!kelas" || cmd == "/kelas" || cmd == "#kelas" ||
		(!isGroup && (cmd == "daftarkelas" || cmd == "kelas")) {

		active := csm.GetClass(chatJID)
		var statusStr string
		if active == "" {
			statusStr = fmt.Sprintf("*%s* _(Kelas Bawaan Default)_", classMgr.GetDefaultClassID())
		} else {
			cfg, _ := classMgr.GetClass(active)
			kampus := ""
			if cfg != nil {
				kampus = fmt.Sprintf(" - _%s_", cfg.Kampus)
			}
			statusStr = fmt.Sprintf("*%s*%s", active, kampus)
		}

		classes := classMgr.ListClasses()
		var sb strings.Builder
		sb.WriteString("🏫 *DAFTAR KELAS PERKULIAHAN*\n")
		sb.WriteString("──────────\n")
		sb.WriteString(fmt.Sprintf("📌 *Kelas Aktif di Chat Ini:* %s\n\n", statusStr))
		sb.WriteString("Pilihan kelas yang tersedia di bot:\n")

		for _, c := range classes {
			cfg, _ := classMgr.GetClass(c)
			desc := ""
			if cfg != nil && cfg.Kampus != "" {
				desc = fmt.Sprintf(" — _%s_", cfg.Kampus)
			}
			if c == active || (active == "" && c == classMgr.GetDefaultClassID()) {
				sb.WriteString(fmt.Sprintf("• ✅ *%s*%s *(Aktif)*\n", c, desc))
			} else {
				sb.WriteString(fmt.Sprintf("• 🔘 *%s*%s\n", c, desc))
			}
		}

		sb.WriteString("\n──────────\n")
		sb.WriteString("💡 *Cara Mengganti Kelas:*\n")
		sb.WriteString("Ketik: `!setkelas [nama_kelas]`\n")
		sb.WriteString("Contoh: `!setkelas 3B`\n")
		if isGroup {
			sb.WriteString("_(Khusus Admin Grup)_")
		} else {
			sb.WriteString("_(Bebas diatur untuk catatan pribadi)_")
		}

		return sb.String()
	}

	// 2. Perintah Mengatur Kelas (!setkelas / !pilihkelas)
	if cmd == "!setkelas" || cmd == "/setkelas" || cmd == "#setkelas" ||
		cmd == "!pilihkelas" || cmd == "/pilihkelas" || cmd == "#pilihkelas" ||
		(!isGroup && (cmd == "setkelas" || cmd == "pilihkelas")) {

		if len(fields) < 2 {
			available := strings.Join(classMgr.ListClasses(), ", ")
			return fmt.Sprintf("ℹ️ *Panduan Penggunaan !setkelas:*\n──────────\nKetik: `!setkelas [nama_kelas]`\nContoh: `!setkelas 3A`\n\nPilihan kelas yang tersedia:\n👉 *%s*\n\nKetik `!daftarkelas` untuk melihat rincian setiap kelas.", available)
		}

		targetRaw := strings.TrimSpace(rawMsg[len(fields[0]):])
		normClass := NormalizeClassID(targetRaw)

		// Otorisasi: Di grup hanya admin yang boleh menyetel
		if isGroup && !isAdmin {
			return "⛔ *AKSES DITOLAK*\nMaaf, hanya *Admin Grup* yang berhak mengubah pengaturan kelas untuk grup ini."
		}

		// Validasi keberadaan kelas
		cfg, exists := classMgr.GetClass(normClass)
		if !exists {
			available := strings.Join(classMgr.ListClasses(), ", ")
			return fmt.Sprintf("⚠️ *Kelas '%s' Tidak Ditemukan!*\n──────────\nPilihan kelas yang tersedia di bot:\n👉 *%s*\n\nKetik `!daftarkelas` untuk melihat informasi lengkap.", targetRaw, available)
		}

		// Simpan ke database dan cache
		if err := csm.SetClass(chatJID, normClass); err != nil {
			return fmt.Sprintf("⚠️ Gagal menyimpan pengaturan kelas: %v", err)
		}

		return fmt.Sprintf("✅ *KELAS BERHASIL DIATUR!*\n──────────\nChat/Grup ini sekarang terhubung ke:\n📌 *Kelas %s*\n🏛️ _%s_\n\nSeluruh jadwal perkuliahan (`!jadwal`, `!hari ini`, `!besok`) dan pengingat harian otomatis mengikuti kelas ini. ✨", normClass, cfg.Kampus)
	}

	// 3. Perintah Reset Kelas (!resetkelas)
	if cmd == "!resetkelas" || cmd == "/resetkelas" || cmd == "#resetkelas" ||
		(!isGroup && cmd == "resetkelas") {

		if isGroup && !isAdmin {
			return "⛔ *AKSES DITOLAK*\nMaaf, hanya *Admin Grup* yang berhak mereset pengaturan kelas untuk grup ini."
		}

		_ = csm.DeleteClass(chatJID)
		return fmt.Sprintf("🔄 *PENGATURAN KELAS DIRESET*\n──────────\nChat/Grup ini telah dikembalikan ke kelas bawaan default (*%s*).", classMgr.GetDefaultClassID())
	}

	return ""
}

