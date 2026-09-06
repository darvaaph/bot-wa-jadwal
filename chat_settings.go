package main

import (
	"database/sql"
	"fmt"
	"sort"
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

// SyncWithClassManager menyelaraskan dan meng-upgrade setelan kelas lama (misal: "3A" -> "D4-TI-SMT3-A")
func (csm *ChatSettingsManager) SyncWithClassManager(classMgr *ClassManager) int {
	if classMgr == nil {
		return 0
	}

	csm.mu.RLock()
	toUpdate := make(map[string]string)
	for scopeJID, currentClass := range csm.cache {
		canonical := classMgr.ResolveClassID(currentClass)
		if canonical != "" && canonical != currentClass {
			toUpdate[scopeJID] = canonical
		}
	}
	csm.mu.RUnlock()

	updatedCount := 0
	for scopeJID, canonical := range toUpdate {
		if err := csm.SetClass(scopeJID, canonical); err == nil {
			updatedCount++
		}
	}

	return updatedCount
}

// GetOnboardingPrompt mengembalikan pesan panduan onboarding ketika chat belum memilih kelas
func (csm *ChatSettingsManager) GetOnboardingPrompt(isGroup bool) string {
	var sb strings.Builder
	sb.WriteString("👋 *HALO! KELAS BELUM DIATUR*\n")
	sb.WriteString("──────────\n")
	if isGroup {
		sb.WriteString("Grup ini belum terhubung ke jadwal kelas mana pun.\n")
		sb.WriteString("Silakan tentukan kelas terlebih dahulu agar bot dapat menampilkan jadwal kuliah, tugas, dan pengingat harian yang sesuai.\n\n")
		sb.WriteString("👉 *Cara Memilih Kelas (Admin Grup):*\n")
		sb.WriteString("Ketik: `!setkelas [nama_kelas]`\n")
		sb.WriteString("Contoh: `!setkelas D4-TI-SMT3-A` atau `!setkelas smt 3 a`\n\n")
		sb.WriteString("💡 Ketik `!daftarkelas` untuk melihat 19 pilihan kelas yang tersedia (dikelompokkan per semester).\n")
		sb.WriteString("──────────\n")
		sb.WriteString("⚠️ _Catatan: Di grup WhatsApp, hanya Admin Grup yang berhak mengatur kelas._")
	} else {
		sb.WriteString("Chat pribadi ini belum terhubung ke jadwal kelas mana pun.\n")
		sb.WriteString("Silakan tentukan kelas Anda terlebih dahulu agar bot dapat menampilkan jadwal kuliah, tugas, dan pengingat harian Anda.\n\n")
		sb.WriteString("👉 *Cara Memilih Kelas:*\n")
		sb.WriteString("Ketik: `!setkelas [nama_kelas]`\n")
		sb.WriteString("Contoh: `!setkelas D4-TI-SMT3-A` atau `!setkelas smt 3 a`\n\n")
		sb.WriteString("💡 Ketik `!daftarkelas` untuk melihat 19 pilihan kelas yang tersedia (dikelompokkan per semester).\n")
		sb.WriteString("──────────\n")
		sb.WriteString("💡 _Di chat pribadi, Anda bebas mengganti kelas kapan saja sesuai kebutuhan._")
	}
	return sb.String()
}

// BuildUnconfiguredMenu membuat menu panduan utama saat chat belum memilih kelas
func (csm *ChatSettingsManager) BuildUnconfiguredMenu(isGroup bool) string {
	var sb strings.Builder
	sb.WriteString("*JADWAL KULIAH MAHASISWA*\n")
	sb.WriteString("Politeknik Negeri Cilacap (PNC)\n")
	sb.WriteString("──────────\n")
	sb.WriteString("⚠️ *STATUS: KELAS BELUM DIATUR*\n")
	sb.WriteString("Chat ini belum memilih kelas perkuliahan aktif.\n\n")

	sb.WriteString("🚀 *LANGKAH AWAL (ONBOARDING):*\n")
	sb.WriteString("1. Ketik `!daftarkelas` ➔ Melihat 19 pilihan kelas D3 & D4 (dikelompokkan per semester)\n")
	sb.WriteString("2. Ketik `!setkelas [nama_kelas]` ➔ Mengaktifkan kelas untuk chat ini\n")
	sb.WriteString("   _Contoh: `!setkelas D4-TI-SMT3-A` atau `!setkelas smt 3 a`_\n\n")

	sb.WriteString("──────────\n")
	sb.WriteString("📋 *DAFTAR FITUR & PERINTAH (Setelah Kelas Aktif):*\n")
	sb.WriteString("• `!next` ➔ Kuliah sedang/berikutnya\n")
	sb.WriteString("• `!hari ini` ➔ Jadwal kuliah hari ini\n")
	sb.WriteString("• `!besok` ➔ Jadwal kuliah besok\n")
	sb.WriteString("• `!seminggu` ➔ Jadwal Senin - Jumat\n")
	sb.WriteString("• `!matkul` ➔ Daftar mata kuliah & dosen\n")
	sb.WriteString("• `!tugas` ➔ Pengingat tugas & deadline\n")
	sb.WriteString("• `!reminder on` ➔ Pengingat pagi otomatis (06:30 WIB)\n")
	sb.WriteString("• `!dosen [nama/kode]` ➔ Cari jadwal dosen\n")
	sb.WriteString("• `!ruang [nama]` ➔ Cari jadwal ruangan\n")
	sb.WriteString("• `!cari [kata]` ➔ Pencarian global\n\n")

	if isGroup {
		sb.WriteString("💡 _Catatan: Di grup WhatsApp, hanya Admin Grup yang dapat menyetel kelas (`!setkelas`)._")
	} else {
		sb.WriteString("💡 _Di chat pribadi (DM), Anda bebas menyetel kelas sesuai perkuliahan Anda._")
	}
	return sb.String()
}

// FormatClassListMessage membangun pesan daftar kelas yang terstruktur dan dikelompokkan per semester
func (csm *ChatSettingsManager) FormatClassListMessage(classMgr *ClassManager, chatJID string, isGroup bool) string {
	active := csm.GetClass(chatJID)
	canonicalActive := ""
	if active != "" {
		canonicalActive = classMgr.ResolveClassID(active)
		if canonicalActive == "" {
			canonicalActive = active
		}
	}

	var statusStr string
	if active == "" {
		statusStr = "⚠️ *Belum Diatur* _(Silakan pilih kelas terlebih dahulu)_"
	} else {
		cfg, _ := classMgr.GetClass(canonicalActive)
		kampus := ""
		if cfg != nil && cfg.Kampus != "" {
			kampus = fmt.Sprintf(" — _%s_", cfg.Kampus)
		}
		statusStr = fmt.Sprintf("*%s*%s", canonicalActive, kampus)
	}

	classes := classMgr.ListClasses()

	type classItem struct {
		id   string
		desc string
	}

	type semesterBucket struct {
		semester int
		items    []classItem
	}

	type programBucket struct {
		name      string
		semesters []*semesterBucket
	}

	progMap := make(map[string]*programBucket)
	progOrder := []string{"D4 TEKNIK INFORMATIKA", "D3 TEKNIK INFORMATIKA", "LAINNYA"}

	for _, pName := range progOrder {
		progMap[pName] = &programBucket{name: pName}
	}

	for _, c := range classes {
		cfg, _ := classMgr.GetClass(c)
		desc := ""
		if cfg != nil {
			desc = cfg.Kampus
		}

		pName := "LAINNYA"
		if strings.HasPrefix(c, "D4-TI-") || strings.HasPrefix(c, "D4-") {
			pName = "D4 TEKNIK INFORMATIKA"
		} else if strings.HasPrefix(c, "D3-TI-") || strings.HasPrefix(c, "D3-") {
			pName = "D3 TEKNIK INFORMATIKA"
		}

		// Ekstrak digit semester dari string seperti "D4-TI-SMT3-A"
		sem := 0
		if idx := strings.Index(c, "SMT"); idx != -1 && len(c) > idx+3 {
			digit := c[idx+3]
			if digit >= '1' && digit <= '8' {
				sem = int(digit - '0')
			}
		}

		pBucket := progMap[pName]
		var sBucket *semesterBucket
		for _, sb := range pBucket.semesters {
			if sb.semester == sem {
				sBucket = sb
				break
			}
		}
		if sBucket == nil {
			sBucket = &semesterBucket{semester: sem}
			pBucket.semesters = append(pBucket.semesters, sBucket)
		}
		sBucket.items = append(sBucket.items, classItem{id: c, desc: desc})
	}

	// Urutkan semester di setiap bucket
	for _, pb := range progMap {
		sort.Slice(pb.semesters, func(i, j int) bool {
			return pb.semesters[i].semester < pb.semesters[j].semester
		})
	}

	var sb strings.Builder
	sb.WriteString("🏫 *DAFTAR KELAS PERKULIAHAN*\n")
	sb.WriteString("──────────\n")
	sb.WriteString(fmt.Sprintf("📌 *Kelas Aktif di Chat Ini:* %s\n\n", statusStr))
	sb.WriteString("Pilihan kelas resmi (dikelompokkan per semester):\n\n")

	for _, pName := range progOrder {
		pb := progMap[pName]
		if len(pb.semesters) == 0 {
			continue
		}

		sb.WriteString(fmt.Sprintf("📚 *PROGRAM STUDI %s*\n", pb.name))
		for _, sem := range pb.semesters {
			semLabel := fmt.Sprintf("Semester %d", sem.semester)
			if sem.semester == 0 {
				semLabel = "Umum / Lintas Semester"
			}
			sb.WriteString(fmt.Sprintf("  • *%s:*\n", semLabel))
			for _, item := range sem.items {
				isActive := canonicalActive != "" && (item.id == canonicalActive || item.id == active)
				tingkat := (sem.semester + 1) / 2
				aliasHint := ""
				parts := strings.Split(item.id, "-")
				if len(parts) >= 1 {
					letter := parts[len(parts)-1]
					if tingkat > 0 && len(letter) == 1 {
						aliasHint = fmt.Sprintf(" _(Kelas %d%s)_", tingkat, letter)
					}
				}
				if isActive {
					sb.WriteString(fmt.Sprintf("    - ✅ *`%s`*%s *(Aktif)*\n", item.id, aliasHint))
				} else {
					sb.WriteString(fmt.Sprintf("    - 🔘 `%s`%s\n", item.id, aliasHint))
				}
			}
		}
		sb.WriteString("\n")
	}

	sb.WriteString("──────────\n")
	sb.WriteString("💡 *Cara Memilih / Mengatur Kelas:*\n")
	sb.WriteString("Ketik: `!setkelas [nama_kelas]`\n")
	sb.WriteString("Contoh: `!setkelas D4-TI-SMT3-A`\n")
	sb.WriteString("_Tips: Anda juga dapat mengetik santai seperti `!setkelas smt 3 a` atau `!setkelas d4 3 a`._\n\n")
	if isGroup {
		sb.WriteString("⚠️ _Catatan: Di grup WhatsApp, hanya Admin Grup yang berhak menyetel kelas._")
	} else {
		sb.WriteString("💡 _Catatan: Di chat pribadi (DM), Anda bebas menyetel kelas sesuai kebutuhan Anda._")
	}

	return sb.String()
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
	hasSymbol := strings.HasPrefix(cmd, "!") || strings.HasPrefix(cmd, "/") || strings.HasPrefix(cmd, "#")
	if isGroup && !hasSymbol {
		return ""
	}

	cleanCmd := cleanCommandPrefix(cmd)

	switch cleanCmd {
	case "daftarkelas", "kelas":
		return csm.FormatClassListMessage(classMgr, chatJID, isGroup)

	case "setkelas", "pilihkelas":
		if len(fields) < 2 {
			return "ℹ️ *Panduan Penggunaan !setkelas:*\n──────────\nKetik: `!setkelas [nama_kelas]`\nContoh: `!setkelas D4-TI-SMT3-A` (atau cukup `!setkelas smt 3 a`)\n\nKetik `!daftarkelas` untuk melihat seluruh pilihan kelas per semester."
		}

		targetRaw := strings.TrimSpace(rawMsg[len(fields[0]):])
		canonicalClass := classMgr.ResolveClassID(targetRaw)

		// Otorisasi: Di grup hanya admin yang boleh menyetel
		if isGroup && !isAdmin {
			return "⛔ *AKSES DITOLAK*\nMaaf, hanya *Admin Grup* yang berhak mengubah pengaturan kelas untuk grup ini."
		}

		// Validasi keberadaan kelas
		cfg, exists := classMgr.GetClass(canonicalClass)
		if !exists || canonicalClass == "" {
			return fmt.Sprintf("⚠️ *Kelas '%s' Tidak Ditemukan!*\n──────────\nPastikan format penulisan benar, contoh: `!setkelas D4-TI-SMT3-A` (atau `!setkelas 2A`).\n\nKetik `!daftarkelas` untuk melihat seluruh pilihan kelas per semester.", targetRaw)
		}

		// Simpan canonicalClass ke database dan cache
		if err := csm.SetClass(chatJID, canonicalClass); err != nil {
			return fmt.Sprintf("⚠️ Gagal menyimpan pengaturan kelas: %v", err)
		}

		_, sem, ting, letter := parseClassMetadata(cfg.Kampus, cfg.FilePath)
		classLabel := canonicalClass
		if sem > 0 && letter != "" {
			classLabel = fmt.Sprintf("%s (Kelas %d%s, Tingkat %d)", canonicalClass, ting, letter, ting)
		}

		return fmt.Sprintf("✅ *KELAS BERHASIL DIATUR!*\n──────────\nChat/Grup ini sekarang terhubung ke:\n📌 *Kelas %s*\n🏛️ _%s_\n\nSeluruh jadwal perkuliahan (`!jadwal`, `!hari ini`, `!besok`) dan pengingat harian otomatis mengikuti kelas ini. ✨", classLabel, cfg.Kampus)

	case "resetkelas", "hapuskelas":
		if isGroup && !isAdmin {
			return "⛔ *AKSES DITOLAK*\nMaaf, hanya *Admin Grup* yang berhak mereset pengaturan kelas untuk grup ini."
		}

		_ = csm.DeleteClass(chatJID)
		return "🔄 *PENGATURAN KELAS DIRESET*\n──────────\nPengaturan kelas untuk chat ini telah dihapus (status: *Belum Diatur*).\n\nSilakan gunakan perintah `!setkelas [nama_kelas]` untuk memilih kelas kembali."

	default:
		return ""
	}
}
