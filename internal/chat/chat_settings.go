package chat

import (
	"bot-jadwal/internal/academic"
	"bot-jadwal/internal/schedule"
	"bot-jadwal/internal/util"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type ChatSettingsManager struct {
	db           *sql.DB
	academicRepo *academic.Repository
	mu           sync.RWMutex
	cache        map[string]string // key: scope_jid, value: class_id (canonical code)
}

// NewChatSettingsManager menginisialisasi ChatSettingsManager menggunakan model target whatsapp_channels dan chat_class_contexts.
func NewChatSettingsManager(db *sql.DB) (*ChatSettingsManager, error) {
	if db == nil {
		return nil, fmt.Errorf("koneksi database tidak boleh nil")
	}

	csm := &ChatSettingsManager{
		db:           db,
		academicRepo: academic.NewRepository(db),
		cache:        make(map[string]string),
	}

	if err := csm.loadCache(); err != nil {
		return nil, fmt.Errorf("gagal memuat data chat settings dari model target: %w", err)
	}

	return csm, nil
}

func (csm *ChatSettingsManager) loadCache() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 1. Backfill otomatis data lama jika tabel legacy chat_settings masih ada
	csm.backfillLegacyChatSettings(ctx)

	csm.mu.Lock()
	defer csm.mu.Unlock()

	// 2. Baca dari whatsapp_channels untuk kanal grup aktif
	rowsChannels, err := csm.db.QueryContext(ctx, `
		SELECT wc.jid, c.code
		FROM whatsapp_channels wc
		JOIN classes c ON c.id = wc.class_id
		WHERE wc.status = 'ACTIVE'
	`)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("gagal membaca whatsapp_channels: %w", err)
	}
	if err == nil {
		defer rowsChannels.Close()
		for rowsChannels.Next() {
			var jid, code string
			if err := rowsChannels.Scan(&jid, &code); err == nil {
				csm.cache[jid] = schedule.NormalizeClassID(code)
			}
		}
		if err := rowsChannels.Err(); err != nil {
			return err
		}
	}

	// 3. Baca dari chat_class_contexts untuk preferensi chat pribadi
	rowsContexts, err := csm.db.QueryContext(ctx, `
		SELECT ccc.chat_jid, c.code
		FROM chat_class_contexts ccc
		JOIN classes c ON c.id = ccc.class_id
	`)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("gagal membaca chat_class_contexts: %w", err)
	}
	if err == nil {
		defer rowsContexts.Close()
		for rowsContexts.Next() {
			var jid, code string
			if err := rowsContexts.Scan(&jid, &code); err == nil {
				csm.cache[jid] = schedule.NormalizeClassID(code)
			}
		}
		if err := rowsContexts.Err(); err != nil {
			return err
		}
	}

	return nil
}

func (csm *ChatSettingsManager) backfillLegacyChatSettings(ctx context.Context) {
	var tableName string
	err := csm.db.QueryRowContext(ctx, "SELECT name FROM sqlite_master WHERE type='table' AND name='chat_settings'").Scan(&tableName)
	if err != nil || tableName == "" {
		return
	}

	rows, err := csm.db.QueryContext(ctx, "SELECT scope_jid, class_id FROM chat_settings")
	if err != nil {
		return
	}

	type legacyEntry struct {
		scopeJID string
		classID  string
	}
	var entries []legacyEntry
	for rows.Next() {
		var scopeJID, classID string
		if err := rows.Scan(&scopeJID, &classID); err == nil {
			entries = append(entries, legacyEntry{
				scopeJID: strings.TrimSpace(scopeJID),
				classID:  strings.TrimSpace(classID),
			})
		}
	}
	_ = rows.Close()

	for _, entry := range entries {
		if entry.scopeJID == "" || entry.classID == "" {
			continue
		}

		cls, err := csm.academicRepo.EnsureClass(ctx, schedule.NormalizeClassID(entry.classID))
		if err != nil || cls == nil {
			continue
		}

		if strings.HasSuffix(entry.scopeJID, "@g.us") {
			_, _ = csm.db.ExecContext(ctx, `
				INSERT INTO whatsapp_channels (class_id, jid, channel_type, display_name, status)
				VALUES (?, ?, 'GROUP', ?, 'ACTIVE')
				ON CONFLICT(jid) DO UPDATE SET class_id = excluded.class_id, status = 'ACTIVE'
			`, cls.ID, entry.scopeJID, cls.Code)
		} else {
			_, _ = csm.db.ExecContext(ctx, `
				INSERT INTO chat_class_contexts (chat_jid, class_id)
				VALUES (?, ?)
				ON CONFLICT(chat_jid) DO UPDATE SET class_id = excluded.class_id
			`, entry.scopeJID, cls.ID)
		}
	}
}

// GetClass mengambil ID kelas yang diatur untuk suatu chat/grup (mengembalikan string kosong jika belum diatur)
func (csm *ChatSettingsManager) GetClass(scopeJID string) string {
	csm.mu.RLock()
	defer csm.mu.RUnlock()

	return csm.cache[scopeJID]
}

func (csm *ChatSettingsManager) SetClass(scopeJID string, rawClassID string) error {
	classID := schedule.NormalizeClassID(rawClassID)
	if classID == "" {
		return fmt.Errorf("nama kelas tidak boleh kosong")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cls, err := csm.academicRepo.EnsureClass(ctx, classID)
	if err != nil {
		return fmt.Errorf("gagal memastikan data kelas %s: %w", classID, err)
	}

	if strings.HasSuffix(scopeJID, "@g.us") {
		query := `
			INSERT INTO whatsapp_channels (class_id, jid, channel_type, display_name, status, updated_at)
			VALUES (?, ?, 'GROUP', ?, 'ACTIVE', strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
			ON CONFLICT(jid) DO UPDATE SET
				class_id = excluded.class_id,
				display_name = excluded.display_name,
				status = 'ACTIVE',
				updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now');
		`
		if _, err := csm.db.ExecContext(ctx, query, cls.ID, scopeJID, cls.Code); err != nil {
			return fmt.Errorf("gagal menyimpan kanal grup whatsapp: %w", err)
		}
	} else {
		query := `
			INSERT INTO chat_class_contexts (chat_jid, class_id, updated_at)
			VALUES (?, ?, strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
			ON CONFLICT(chat_jid) DO UPDATE SET
				class_id = excluded.class_id,
				updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now');
		`
		if _, err := csm.db.ExecContext(ctx, query, scopeJID, cls.ID); err != nil {
			return fmt.Errorf("gagal menyimpan konteks kelas chat: %w", err)
		}
	}

	csm.mu.Lock()
	csm.cache[scopeJID] = classID
	csm.mu.Unlock()

	return nil
}

func (csm *ChatSettingsManager) DeleteClass(scopeJID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if strings.HasSuffix(scopeJID, "@g.us") {
		_, err := csm.db.ExecContext(ctx, `DELETE FROM whatsapp_channels WHERE jid = ?`, scopeJID)
		if err != nil {
			return fmt.Errorf("gagal menghapus kanal whatsapp: %w", err)
		}
	} else {
		_, err := csm.db.ExecContext(ctx, `DELETE FROM chat_class_contexts WHERE chat_jid = ?`, scopeJID)
		if err != nil {
			return fmt.Errorf("gagal menghapus konteks kelas chat: %w", err)
		}
	}

	csm.mu.Lock()
	delete(csm.cache, scopeJID)
	csm.mu.Unlock()

	return nil
}

func (csm *ChatSettingsManager) CountSettings() int {
	csm.mu.RLock()
	defer csm.mu.RUnlock()

	return len(csm.cache)
}

// SyncWithClassManager menyelaraskan dan meng-upgrade setelan kelas lama (misal: "3A" -> "D4-TI-SMT3-A")
func (csm *ChatSettingsManager) SyncWithClassManager(classMgr *schedule.ClassManager) int {
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
	sb.WriteString("• `!link` ➔ Tautan penting kelas (Drive/Zoom)\n")
	sb.WriteString("• `!reminder on` ➔ Pengingat pagi otomatis (06:00 WIB)\n")
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
func (csm *ChatSettingsManager) FormatClassListMessage(classMgr *schedule.ClassManager, chatJID string, isGroup bool) string {
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
	classMgr *schedule.ClassManager,
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

	cleanCmd := util.CleanCommandPrefix(cmd)

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

		cfg, exists := classMgr.GetClass(canonicalClass)
		if !exists || canonicalClass == "" {
			return fmt.Sprintf("⚠️ *Kelas '%s' Tidak Ditemukan!*\n──────────\nPastikan format penulisan benar, contoh: `!setkelas D4-TI-SMT3-A` (atau `!setkelas 2A`).\n\nKetik `!daftarkelas` untuk melihat seluruh pilihan kelas per semester.", targetRaw)
		}

		if err := csm.SetClass(chatJID, canonicalClass); err != nil {
			return fmt.Sprintf("⚠️ Gagal menyimpan pengaturan kelas: %v", err)
		}

		_, sem, ting, letter := schedule.ParseClassMetadata(cfg.Kampus, cfg.FilePath)
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
