package main

import (
	"database/sql"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// LinkItem merepresentasikan satu rekaman tautan kelas di database
type LinkItem struct {
	ID          int64     `json:"id"`
	ScopeJID    string    `json:"scope_jid"`
	IsGroup     bool      `json:"is_group"`
	Title       string    `json:"title"`
	URL         string    `json:"url"`
	Category    string    `json:"category"` // "drive", "meeting", "repo", "portal", "umum"
	Description string    `json:"description"`
	CreatedBy   string    `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
}

// LinkManager mengelola penyimpanan dan kueri tautan kelas berbasis SQLite
type LinkManager struct {
	db *sql.DB
}

// NewLinkManager menginisialisasi tabel class_links pada instance *sql.DB bersama
func NewLinkManager(db *sql.DB) (*LinkManager, error) {
	if db == nil {
		return nil, fmt.Errorf("koneksi database tidak boleh nil")
	}

	query := `
	CREATE TABLE IF NOT EXISTS class_links (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		scope_jid TEXT NOT NULL,
		is_group BOOLEAN NOT NULL,
		title TEXT NOT NULL,
		url TEXT NOT NULL,
		category TEXT NOT NULL DEFAULT 'umum',
		description TEXT DEFAULT '',
		created_by TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_links_scope ON class_links(scope_jid);
	`
	_, err := db.Exec(query)
	if err != nil {
		return nil, fmt.Errorf("gagal membuat tabel class_links: %w", err)
	}

	return &LinkManager{db: db}, nil
}

// NormalizeURL memastikan URL diawali http:// atau https:// agar otomatis clickable di WhatsApp
func NormalizeURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if !strings.HasPrefix(raw, "http://") && !strings.HasPrefix(raw, "https://") {
		return "https://" + raw
	}
	return raw
}

// DetectLinkCategory mendeteksi kategori tautan secara cerdas dari judul dan URL
func DetectLinkCategory(title, rawURL string) string {
	combined := strings.ToLower(title + " " + rawURL)
	if strings.Contains(combined, "drive.google.com") || strings.Contains(combined, "onedrive") || strings.Contains(combined, "dropbox") || strings.Contains(combined, "drive") || strings.Contains(combined, "materi") {
		return "drive"
	}
	if strings.Contains(combined, "zoom.us") || strings.Contains(combined, "meet.google.com") || strings.Contains(combined, "teams.microsoft.com") || strings.Contains(combined, "webex") || strings.Contains(combined, "zoom") || strings.Contains(combined, "gmeet") || strings.Contains(combined, "meet") || strings.Contains(combined, "kuliah online") || strings.Contains(combined, "kuliah daring") {
		return "meeting"
	}
	if strings.Contains(combined, "github.com") || strings.Contains(combined, "gitlab.com") || strings.Contains(combined, "bitbucket.org") || strings.Contains(combined, "repo") || strings.Contains(combined, "repositori") {
		return "repo"
	}
	if strings.Contains(combined, "siakad") || strings.Contains(combined, "elearning") || strings.Contains(combined, "moodle") || strings.Contains(combined, "portal") || strings.Contains(combined, "presensi") || strings.Contains(combined, "absensi") {
		return "portal"
	}
	return "umum"
}

// AddLink menambahkan tautan baru ke database dengan normalisasi URL dan deteksi kategori cerdas
func (lm *LinkManager) AddLink(scopeJID string, isGroup bool, title, rawURL, desc, createdBy string) (int64, error) {
	title = strings.TrimSpace(title)
	rawURL = NormalizeURL(rawURL)
	desc = strings.TrimSpace(desc)

	if title == "" {
		return 0, fmt.Errorf("judul tautan tidak boleh kosong")
	}
	if rawURL == "" {
		return 0, fmt.Errorf("URL tautan tidak boleh kosong")
	}

	// Validasi struktur URL
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Host == "" {
		return 0, fmt.Errorf("format URL '%s' tidak valid", rawURL)
	}

	category := DetectLinkCategory(title, rawURL)

	query := `
	INSERT INTO class_links (scope_jid, is_group, title, url, category, description, created_by, created_at)
	VALUES (?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP);
	`
	res, err := lm.db.Exec(query, scopeJID, isGroup, title, rawURL, category, desc, createdBy)
	if err != nil {
		return 0, fmt.Errorf("gagal menambahkan tautan ke database: %w", err)
	}

	return res.LastInsertId()
}

// DeleteLink menghapus tautan berdasarkan ID dan scope chat
func (lm *LinkManager) DeleteLink(scopeJID string, id int64) (bool, error) {
	query := `DELETE FROM class_links WHERE id = ? AND scope_jid = ?;`
	res, err := lm.db.Exec(query, id, scopeJID)
	if err != nil {
		return false, fmt.Errorf("gagal menghapus tautan: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}

// GetLinks mengambil seluruh tautan aktif pada scope chat terurut berdasarkan kategori utama
func (lm *LinkManager) GetLinks(scopeJID string) ([]LinkItem, error) {
	query := `
	SELECT id, scope_jid, is_group, title, url, category, description, created_by, created_at
	FROM class_links
	WHERE scope_jid = ?
	ORDER BY 
		CASE category
			WHEN 'drive' THEN 1
			WHEN 'meeting' THEN 2
			WHEN 'repo' THEN 3
			WHEN 'portal' THEN 4
			ELSE 5
		END,
		created_at ASC;
	`
	return lm.queryLinks(query, scopeJID)
}

// GetLinksByCategory mengambil tautan berdasarkan kategori tertentu (misal: "drive", "meeting")
func (lm *LinkManager) GetLinksByCategory(scopeJID, category string) ([]LinkItem, error) {
	query := `
	SELECT id, scope_jid, is_group, title, url, category, description, created_by, created_at
	FROM class_links
	WHERE scope_jid = ? AND category = ?
	ORDER BY created_at ASC;
	`
	return lm.queryLinks(query, scopeJID, category)
}

// SearchLinks mencari tautan berdasarkan kata kunci pada judul, deskripsi, atau kategori
func (lm *LinkManager) SearchLinks(scopeJID, keyword string) ([]LinkItem, error) {
	pattern := "%" + strings.ToLower(strings.TrimSpace(keyword)) + "%"
	query := `
	SELECT id, scope_jid, is_group, title, url, category, description, created_by, created_at
	FROM class_links
	WHERE scope_jid = ? AND (LOWER(title) LIKE ? OR LOWER(description) LIKE ? OR LOWER(category) LIKE ?)
	ORDER BY created_at ASC;
	`
	return lm.queryLinks(query, scopeJID, pattern, pattern, pattern)
}

func (lm *LinkManager) queryLinks(query string, args ...any) ([]LinkItem, error) {
	rows, err := lm.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []LinkItem
	for rows.Next() {
		var item LinkItem
		var rawCreatedAt string
		err := rows.Scan(
			&item.ID,
			&item.ScopeJID,
			&item.IsGroup,
			&item.Title,
			&item.URL,
			&item.Category,
			&item.Description,
			&item.CreatedBy,
			&rawCreatedAt,
		)
		if err != nil {
			return nil, err
		}
		item.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", rawCreatedAt)
		links = append(links, item)
	}
	return links, nil
}

// FormatLinkList menyusun daftar tautan ke format pesan WhatsApp yang rapi dan dikelompokkan
func (lm *LinkManager) FormatLinkList(links []LinkItem, isGroup bool, customHeader ...string) string {
	var sb strings.Builder

	header := "🔗 *DAFTAR TAUTAN PENTING KELAS*"
	if !isGroup {
		header = "🔗 *DAFTAR TAUTAN PRIBADI*"
	}
	if len(customHeader) > 0 && customHeader[0] != "" {
		header = customHeader[0]
	}

	sb.WriteString(fmt.Sprintf("%s\n", header))
	sb.WriteString("──────────\n\n")

	if len(links) == 0 {
		sb.WriteString("Belum ada tautan penting yang dicatat.\n\n")
		sb.WriteString("💡 *Format Menambah Tautan:*\n")
		sb.WriteString("👉 `!link tambah [Judul] | [URL] (| [Catatan])`\n")
		sb.WriteString("_Contoh:_ `!link tambah Drive Materi | https://s.id/drive-d4a`\n")
		return sb.String()
	}

	// Kelompokkan per kategori
	categoryMap := map[string][]LinkItem{
		"drive":   {},
		"meeting": {},
		"repo":    {},
		"portal":  {},
		"umum":    {},
	}

	for _, l := range links {
		cat := l.Category
		if _, ok := categoryMap[cat]; !ok {
			cat = "umum"
		}
		categoryMap[cat] = append(categoryMap[cat], l)
	}

	categoryMeta := []struct {
		Key   string
		Label string
	}{
		{"drive", "📁 PENYIMPANAN & MATERI (GOOGLE DRIVE)"},
		{"meeting", "📹 KULIAH DARING (ZOOM / GMEET)"},
		{"repo", "🐙 REPOSITORI & PROYEK (GITHUB / GITLAB)"},
		{"portal", "🌐 PORTAL AKADEMIK (SIAKAD / LMS)"},
		{"umum", "📌 TAUTAN PENTING LAINNYA"},
	}

	for _, meta := range categoryMeta {
		items := categoryMap[meta.Key]
		if len(items) == 0 {
			continue
		}

		sb.WriteString(fmt.Sprintf("*%s*\n", meta.Label))
		for _, item := range items {
			sb.WriteString(fmt.Sprintf("• *%s* (#%d)\n", item.Title, item.ID))
			sb.WriteString(fmt.Sprintf("  └ 🔗 %s\n", item.URL))
			if item.Description != "" {
				sb.WriteString(fmt.Sprintf("  └ 📝 _%s_\n", item.Description))
			}
		}
		sb.WriteString("\n")
	}

	sb.WriteString("──────────\n")
	sb.WriteString("_Tips: Ketik `!drive` untuk akses cepat Drive, atau `!zoom` untuk kuliah daring._\n")
	sb.WriteString("_Admin dapat menghapus dengan `!link hapus [ID]`._")

	return sb.String()
}

// FormatDriveShortcut menyusun tampilan akses cepat untuk shortcut !drive / !gdrive
func (lm *LinkManager) FormatDriveShortcut(links []LinkItem, isGroup bool) string {
	var sb strings.Builder

	title := "📁 *GOOGLE DRIVE KELAS*"
	if !isGroup {
		title = "📁 *GOOGLE DRIVE PRIBADI*"
	}

	sb.WriteString(fmt.Sprintf("%s\n", title))
	sb.WriteString("──────────\n\n")

	if len(links) == 0 {
		sb.WriteString("Belum ada tautan Google Drive yang didaftarkan.\n\n")
		sb.WriteString("💡 *Cara Mendaftarkan Tautan Drive:*\n")
		sb.WriteString("👉 `!link tambah Drive Materi | [URL_Drive] | [Keterangan]`\n")
		return sb.String()
	}

	for _, l := range links {
		sb.WriteString(fmt.Sprintf("• *%s* (#%d)\n", l.Title, l.ID))
		sb.WriteString(fmt.Sprintf("  └ 🔗 %s\n", l.URL))
		if l.Description != "" {
			sb.WriteString(fmt.Sprintf("  └ 📝 _%s_\n", l.Description))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("──────────\n")
	sb.WriteString("_Ketik `!link` untuk melihat seluruh tautan penting lainnya._")

	return sb.String()
}

// FormatMeetingShortcut menyusun tampilan akses cepat untuk shortcut !zoom / !gmeet / !meet
func (lm *LinkManager) FormatMeetingShortcut(links []LinkItem, isGroup bool) string {
	var sb strings.Builder

	title := "📹 *RUANG KULIAH DARING (ZOOM / GMEET)*"
	if !isGroup {
		title = "📹 *RUANG KULIAH DARING PRIBADI*"
	}

	sb.WriteString(fmt.Sprintf("%s\n", title))
	sb.WriteString("──────────\n\n")

	if len(links) == 0 {
		sb.WriteString("Belum ada tautan Zoom atau GMeet yang didaftarkan.\n\n")
		sb.WriteString("💡 *Cara Mendaftarkan Ruang Kuliah Daring:*\n")
		sb.WriteString("👉 `!link tambah Zoom [Nama_Matkul] | [URL_Meeting]`\n")
		return sb.String()
	}

	for _, l := range links {
		sb.WriteString(fmt.Sprintf("• *%s* (#%d)\n", l.Title, l.ID))
		sb.WriteString(fmt.Sprintf("  └ 🔗 %s\n", l.URL))
		if l.Description != "" {
			sb.WriteString(fmt.Sprintf("  └ 📝 _%s_\n", l.Description))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("──────────\n")
	sb.WriteString("_Ketik `!link` untuk melihat seluruh tautan penting lainnya._")

	return sb.String()
}

// HandleCommand memproses perintah !link, !drive, !zoom dan sub-perintah lainnya
func (lm *LinkManager) HandleCommand(
	scopeJID string,
	isGroup bool,
	senderJID string,
	isAdmin bool,
	rawMsg string,
) string {
	cleanMsg := strings.TrimSpace(rawMsg)
	lowerMsg := strings.ToLower(cleanMsg)

	// Pisahkan kata pertama sebagai command root
	parts := strings.Fields(lowerMsg)
	if len(parts) == 0 {
		return ""
	}
	rootCmd := strings.TrimPrefix(parts[0], "!")

	// 1. Shortcut !drive / !gdrive
	if rootCmd == "drive" || rootCmd == "gdrive" {
		links, err := lm.GetLinksByCategory(scopeJID, "drive")
		if err != nil {
			return fmt.Sprintf("❌ Gagal memuat tautan Drive: %v", err)
		}
		if len(links) == 0 {
			// Fallback: coba cari link yang judulnya mengandung kata "drive"
			links, _ = lm.SearchLinks(scopeJID, "drive")
		}
		return lm.FormatDriveShortcut(links, isGroup)
	}

	// 2. Shortcut !zoom / !gmeet / !meet
	if rootCmd == "zoom" || rootCmd == "gmeet" || rootCmd == "meet" {
		links, err := lm.GetLinksByCategory(scopeJID, "meeting")
		if err != nil {
			return fmt.Sprintf("❌ Gagal memuat tautan meeting: %v", err)
		}
		if len(links) == 0 {
			// Fallback: coba cari link yang judulnya mengandung zoom / meet
			links, _ = lm.SearchLinks(scopeJID, "zoom")
			if len(links) == 0 {
				links, _ = lm.SearchLinks(scopeJID, "meet")
			}
		}
		return lm.FormatMeetingShortcut(links, isGroup)
	}

	// 3. Perintah Utama !link / !tautan
	if rootCmd == "link" || rootCmd == "tautan" {
		// Jika hanya "!link" atau "!tautan" tanpa sub-perintah
		if len(parts) == 1 {
			links, err := lm.GetLinks(scopeJID)
			if err != nil {
				return fmt.Sprintf("❌ Gagal memuat daftar tautan: %v", err)
			}
			return lm.FormatLinkList(links, isGroup)
		}

		subCmd := parts[1]

		switch subCmd {
		case "bantuan", "help", "panduan":
			return lm.buildHelp(isGroup)

		case "tambah", "add":
			if isGroup && !isAdmin {
				return "❌ *Akses Ditolak: Hanya Admin Grup yang dapat menambahkan tautan penting kelas.*"
			}

			// Ambil sisa teks setelah "!link tambah"
			rawArgs := strings.TrimSpace(cleanMsg[len(parts[0]):])
			rawArgs = strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(rawArgs, subCmd), strings.ToUpper(subCmd)))

			segments := strings.Split(rawArgs, "|")
			if len(segments) < 2 {
				return "⚠️ *Format Penambahan Tautan Kurang Tepat!*\n──────────\n" +
					"Gunakan tanda pemisah pipa `|`:\n" +
					"`!link tambah [Judul] | [URL] (| [Catatan Opsional])`\n\n" +
					"*Contoh:*\n" +
					"• `!link tambah Drive Materi | https://s.id/drive-d4a`\n" +
					"• `!link tambah Zoom Aljabar | https://meet.google.com/abc-xyz | Dosen: Bu Retno`\n" +
					"• `!link tambah Repo Praktikum | https://github.com/kelas-sbd`"
			}

			title := strings.TrimSpace(segments[0])
			rawURL := strings.TrimSpace(segments[1])
			desc := ""
			if len(segments) >= 3 {
				desc = strings.TrimSpace(segments[2])
			}

			id, err := lm.AddLink(scopeJID, isGroup, title, rawURL, desc, senderJID)
			if err != nil {
				return fmt.Sprintf("❌ Gagal menyimpan tautan: %v", err)
			}

			cat := DetectLinkCategory(title, rawURL)
			catName := "Tautan Umum"
			switch cat {
			case "drive":
				catName = "Google Drive / Penyimpanan Materi"
			case "meeting":
				catName = "Kuliah Daring (Zoom / GMeet)"
			case "repo":
				catName = "Repositori / Proyek (GitHub / GitLab)"
			case "portal":
				catName = "Portal Akademik (SIAKAD / LMS)"
			}

			var sb strings.Builder
			sb.WriteString("✅ *TAUTAN BERHASIL DISIMPAN!*\n")
			sb.WriteString("──────────\n")
			sb.WriteString(fmt.Sprintf("• ID Tautan : #%d\n", id))
			sb.WriteString(fmt.Sprintf("• Judul     : %s\n", title))
			sb.WriteString(fmt.Sprintf("• Kategori  : %s\n", catName))
			sb.WriteString(fmt.Sprintf("• URL       : %s\n", NormalizeURL(rawURL)))
			if desc != "" {
				sb.WriteString(fmt.Sprintf("• Catatan   : %s\n", desc))
			}
			sb.WriteString("──────────\n")
			sb.WriteString("_Tautan kini dapat diakses oleh seluruh anggota kelas dengan mengetik `!link`._")
			return sb.String()

		case "hapus", "delete", "del", "remove":
			if isGroup && !isAdmin {
				return "❌ *Akses Ditolak: Hanya Admin Grup yang dapat menghapus tautan penting kelas.*"
			}
			if len(parts) < 3 {
				return "⚠️ *Harap tentukan ID tautan yang ingin dihapus!*\nContoh: `!link hapus 1`\n\nKetik `!link` untuk melihat daftar ID tautan."
			}

			rawID := strings.TrimPrefix(parts[2], "#")
			id, err := strconv.ParseInt(rawID, 10, 64)
			if err != nil || id <= 0 {
				return fmt.Sprintf("⚠️ ID tautan '%s' tidak valid. Gunakan angka, contoh: `!link hapus 2`.", parts[2])
			}

			deleted, err := lm.DeleteLink(scopeJID, id)
			if err != nil {
				return fmt.Sprintf("❌ Gagal menghapus tautan: %v", err)
			}
			if !deleted {
				return fmt.Sprintf("⚠️ Tautan dengan ID #%d tidak ditemukan pada chat ini.", id)
			}

			return fmt.Sprintf("🗑️ *TAUTAN BERHASIL DIHAPUS!*\nTautan dengan ID #%d telah dibersihkan dari sistem.", id)

		default:
			// Filter atau cari tautan berdasarkan kata kunci: !link [kata] atau !link cari [kata]
			keyword := subCmd
			if (subCmd == "cari" || subCmd == "search") && len(parts) > 2 {
				keyword = strings.Join(parts[2:], " ")
			} else if len(parts) > 1 {
				// Ambil sisa teks setelah !link
				keyword = strings.TrimSpace(cleanMsg[len(parts[0]):])
			}

			links, err := lm.SearchLinks(scopeJID, keyword)
			if err != nil {
				return fmt.Sprintf("❌ Gagal mencari tautan: %v", err)
			}

			header := fmt.Sprintf("🔍 *PENCARIAN TAUTAN: \"%s\"*", strings.ToUpper(keyword))
			if len(links) == 0 {
				return fmt.Sprintf("🔍 *PENCARIAN TAUTAN: \"%s\"*\n──────────\nTidak ada tautan yang cocok dengan kata kunci tersebut.\n\nKetik `!link` untuk melihat seluruh tautan yang tersedia.", strings.ToUpper(keyword))
			}
			return lm.FormatLinkList(links, isGroup, header)
		}
	}

	return ""
}

func (lm *LinkManager) buildHelp(isGroup bool) string {
	var sb strings.Builder
	sb.WriteString("📖 *PANDUAN MODUL TAUTAN PENTING KELAS*\n")
	sb.WriteString("──────────\n\n")
	sb.WriteString("Modul ini memudahkan anggota kelas mengakses link Drive, Zoom, GitHub, dan portal kampus tanpa tenggelam di grup.\n\n")

	sb.WriteString("*Perintah Membaca (Bebas Semua Anggota):*\n")
	sb.WriteString("• `!link` / `!tautan`\n  ➔ Melihat seluruh tautan penting kelas\n")
	sb.WriteString("• `!drive` / `!gdrive`\n  ➔ Shortcut instan link Google Drive materi\n")
	sb.WriteString("• `!zoom` / `!gmeet` / `!meet`\n  ➔ Shortcut instan link ruang kuliah daring\n")
	sb.WriteString("• `!link [kata kunci]`\n  ➔ Mencari tautan spesifik (Contoh: `!link alin`, `!link sbd`)\n\n")

	sb.WriteString("*Perintah Pengelolaan (Khusus Admin di Grup):*\n")
	sb.WriteString("• `!link tambah [Judul] | [URL] (| [Catatan])`\n  ➔ Menambahkan tautan baru\n")
	sb.WriteString("  _Contoh:_ `!link tambah Drive Materi | https://s.id/drive-d4a`\n")
	sb.WriteString("  _Contoh:_ `!link tambah Zoom Alin | https://meet.google.com/abc | Bu Retno`\n\n")
	sb.WriteString("• `!link hapus [ID]`\n  ➔ Menghapus tautan (Contoh: `!link hapus 1`)\n\n")

	sb.WriteString("──────────\n")
	sb.WriteString("_Tips: URL otomatis dinormalisasi menjadi HTTPS agar langsung bisa diklik di ponsel._")
	return sb.String()
}
