package link

import (
	"bot-jadwal/internal/academic"
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

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

type LinkManager struct {
	db           *sql.DB
	academicRepo *academic.Repository
}

// NewLinkManager menginisialisasi LinkManager yang bertumpu pada tabel target materials.
func NewLinkManager(db *sql.DB) (*LinkManager, error) {
	if db == nil {
		return nil, fmt.Errorf("koneksi database tidak boleh nil")
	}

	lm := &LinkManager{
		db:           db,
		academicRepo: academic.NewRepository(db),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	lm.backfillLegacyLinks(ctx)

	return lm, nil
}

// CategoryToMaterialType memetakan kategori lama ke material_type tabel materials.
func CategoryToMaterialType(cat string) string {
	switch strings.ToLower(strings.TrimSpace(cat)) {
	case "drive":
		return "DOCUMENT"
	case "meeting":
		return "MEETING"
	case "repo":
		return "REPOSITORY"
	case "portal":
		return "PORTAL"
	default:
		return "OTHER"
	}
}

// MaterialTypeToCategory memetakan material_type tabel materials ke kategori lama LinkItem.
func MaterialTypeToCategory(matType string) string {
	switch strings.ToUpper(strings.TrimSpace(matType)) {
	case "DOCUMENT":
		return "drive"
	case "MEETING":
		return "meeting"
	case "REPOSITORY":
		return "repo"
	case "PORTAL":
		return "portal"
	default:
		return "umum"
	}
}

func (lm *LinkManager) backfillLegacyLinks(ctx context.Context) {
	var tableName string
	err := lm.db.QueryRowContext(ctx, "SELECT name FROM sqlite_master WHERE type='table' AND name='class_links'").Scan(&tableName)
	if err != nil || tableName == "" {
		return
	}

	rows, err := lm.db.QueryContext(ctx, `
		SELECT scope_jid, is_group, title, url, category, description, created_by
		FROM class_links
	`)
	if err != nil {
		return
	}

	type legacyLink struct {
		scopeJID    string
		isGroup     bool
		title       string
		url         string
		category    string
		description string
		createdBy   string
	}
	var legacyItems []legacyLink
	for rows.Next() {
		var l legacyLink
		if err := rows.Scan(&l.scopeJID, &l.isGroup, &l.title, &l.url, &l.category, &l.description, &l.createdBy); err == nil {
			legacyItems = append(legacyItems, l)
		}
	}
	_ = rows.Close()

	for _, item := range legacyItems {
		classID, _, err := lm.academicRepo.ResolveClassIDFromScope(ctx, item.scopeJID)
		if err != nil {
			cls, ensureErr := lm.academicRepo.EnsureClass(ctx, item.scopeJID)
			if ensureErr != nil {
				continue
			}
			classID = cls.ID
		}

		userID, err := lm.academicRepo.EnsureUser(ctx, item.createdBy, item.createdBy)
		if err != nil {
			continue
		}

		matType := CategoryToMaterialType(item.category)
		visibility := "CLASS_ACCESS"
		if !item.isGroup {
			visibility = "WHATSAPP_ONLY"
		}

		var exists int
		_ = lm.db.QueryRowContext(ctx, `SELECT 1 FROM materials WHERE class_id = ? AND url = ? LIMIT 1`, classID, item.url).Scan(&exists)
		if exists == 0 {
			_, _ = lm.db.ExecContext(ctx, `
				INSERT INTO materials (class_id, title, material_type, url, description, visibility, status, created_by_user_id)
				VALUES (?, ?, ?, ?, ?, ?, 'ACTIVE', ?)
			`, classID, item.title, matType, item.url, item.description, visibility, userID)
		}
	}
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

// DetectLinkCategory menentukan kategori dari judul dan host URL.
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

// AddLink menambahkan tautan baru ke database target materials dengan normalisasi URL dan deteksi kategori cerdas
func (lm *LinkManager) AddLink(scopeJID string, isGroup bool, title, rawURL, desc, createdBy string) (int64, error) {
	title = strings.TrimSpace(title)
	rawURL = NormalizeURL(rawURL)
	desc = strings.TrimSpace(desc)
	createdBy = strings.TrimSpace(createdBy)

	if title == "" {
		return 0, fmt.Errorf("judul tautan tidak boleh kosong")
	}
	if rawURL == "" {
		return 0, fmt.Errorf("URL tautan tidak boleh kosong")
	}

	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Host == "" {
		return 0, fmt.Errorf("format URL '%s' tidak valid", rawURL)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	classID, _, err := lm.academicRepo.ResolveClassIDFromScope(ctx, scopeJID)
	if err != nil {
		cls, ensureErr := lm.academicRepo.EnsureClass(ctx, scopeJID)
		if ensureErr != nil {
			return 0, fmt.Errorf("gagal mengaitkan tautan ke kelas: %w", ensureErr)
		}
		classID = cls.ID
		if strings.HasSuffix(scopeJID, "@g.us") {
			_, _ = lm.db.ExecContext(ctx, `
				INSERT INTO whatsapp_channels (class_id, jid, channel_type, display_name, status)
				VALUES (?, ?, 'GROUP', ?, 'ACTIVE')
				ON CONFLICT(jid) DO NOTHING;
			`, cls.ID, scopeJID, cls.Code)
		} else {
			_, _ = lm.db.ExecContext(ctx, `
				INSERT INTO chat_class_contexts (chat_jid, class_id)
				VALUES (?, ?)
				ON CONFLICT(chat_jid) DO NOTHING;
			`, scopeJID, cls.ID)
		}
	}

	userID, err := lm.academicRepo.EnsureUser(ctx, createdBy, createdBy)
	if err != nil {
		return 0, fmt.Errorf("gagal memetakan user pembuat tautan: %w", err)
	}

	category := DetectLinkCategory(title, rawURL)
	matType := CategoryToMaterialType(category)

	visibility := "CLASS_ACCESS"
	if !isGroup {
		visibility = "WHATSAPP_ONLY"
	}

	query := `
		INSERT INTO materials (class_id, title, material_type, url, description, visibility, status, created_by_user_id)
		VALUES (?, ?, ?, ?, ?, ?, 'ACTIVE', ?);
	`
	res, err := lm.db.ExecContext(ctx, query, classID, title, matType, rawURL, desc, visibility, userID)
	if err != nil {
		return 0, fmt.Errorf("gagal menambahkan tautan ke materials: %w", err)
	}

	return res.LastInsertId()
}

func (lm *LinkManager) DeleteLink(scopeJID string, id int64) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	classID, _, err := lm.academicRepo.ResolveClassIDFromScope(ctx, scopeJID)
	if err != nil {
		cls, ensureErr := lm.academicRepo.GetClassByCode(ctx, scopeJID)
		if ensureErr != nil || cls == nil {
			return false, nil
		}
		classID = cls.ID
	}

	userID, _ := lm.academicRepo.EnsureUser(ctx, "system", "System")

	query := `
		UPDATE materials
		SET deleted_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now'),
		    deleted_by_user_id = ?,
		    status = 'INACTIVE'
		WHERE id = ? AND class_id = ? AND deleted_at IS NULL;
	`
	res, err := lm.db.ExecContext(ctx, query, userID, id, classID)
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
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return lm.queryLinks(ctx, scopeJID, "")
}

func (lm *LinkManager) GetLinksByCategory(scopeJID, category string) ([]LinkItem, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	matType := CategoryToMaterialType(category)
	return lm.queryLinks(ctx, scopeJID, "AND m.material_type = ?", matType)
}

func (lm *LinkManager) SearchLinks(scopeJID, keyword string) ([]LinkItem, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	pattern := "%" + strings.ToLower(strings.TrimSpace(keyword)) + "%"
	return lm.queryLinks(ctx, scopeJID, "AND (LOWER(m.title) LIKE ? OR LOWER(m.description) LIKE ? OR LOWER(m.material_type) LIKE ?)", pattern, pattern, pattern)
}

func (lm *LinkManager) queryLinks(ctx context.Context, scopeJID string, filterClause string, args ...any) ([]LinkItem, error) {
	classID, _, err := lm.academicRepo.ResolveClassIDFromScope(ctx, scopeJID)
	if err != nil {
		cls, errGet := lm.academicRepo.GetClassByCode(ctx, scopeJID)
		if errGet != nil || cls == nil {
			return nil, nil
		}
		classID = cls.ID
	}

	query := `
		SELECT m.id, m.title, m.url, m.material_type, COALESCE(m.description, ''), u.identity_key, m.created_at, m.visibility
		FROM materials m
		JOIN users u ON u.id = m.created_by_user_id
		WHERE m.class_id = ? AND m.deleted_at IS NULL AND m.status = 'ACTIVE' ` + filterClause + `
		ORDER BY
			CASE m.material_type
				WHEN 'DOCUMENT' THEN 1
				WHEN 'MEETING' THEN 2
				WHEN 'REPOSITORY' THEN 3
				WHEN 'PORTAL' THEN 4
				ELSE 5
			END,
			m.created_at ASC;
	`
	fullArgs := append([]any{classID}, args...)
	rows, err := lm.db.QueryContext(ctx, query, fullArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []LinkItem
	for rows.Next() {
		var item LinkItem
		var matType, rawCreatedAt, visibility string
		err := rows.Scan(
			&item.ID,
			&item.Title,
			&item.URL,
			&matType,
			&item.Description,
			&item.CreatedBy,
			&rawCreatedAt,
			&visibility,
		)
		if err != nil {
			return nil, err
		}
		item.ScopeJID = scopeJID
		item.IsGroup = (visibility == "CLASS_ACCESS")
		item.Category = MaterialTypeToCategory(matType)
		item.CreatedAt, _ = time.Parse(time.RFC3339, rawCreatedAt)
		if item.CreatedAt.IsZero() {
			item.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", rawCreatedAt)
		}
		links = append(links, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return links, nil
}

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

	parts := strings.Fields(lowerMsg)
	if len(parts) == 0 {
		return ""
	}
	rootCmd := strings.TrimPrefix(parts[0], "!")

	if rootCmd == "drive" || rootCmd == "gdrive" {
		links, err := lm.GetLinksByCategory(scopeJID, "drive")
		if err != nil {
			return fmt.Sprintf("❌ Gagal memuat tautan Drive: %v", err)
		}
		if len(links) == 0 {
			links, _ = lm.SearchLinks(scopeJID, "drive")
		}
		return lm.FormatDriveShortcut(links, isGroup)
	}

	if rootCmd == "zoom" || rootCmd == "gmeet" || rootCmd == "meet" {
		links, err := lm.GetLinksByCategory(scopeJID, "meeting")
		if err != nil {
			return fmt.Sprintf("❌ Gagal memuat tautan meeting: %v", err)
		}
		if len(links) == 0 {
			links, _ = lm.SearchLinks(scopeJID, "zoom")
			if len(links) == 0 {
				links, _ = lm.SearchLinks(scopeJID, "meet")
			}
		}
		return lm.FormatMeetingShortcut(links, isGroup)
	}

	if rootCmd == "link" || rootCmd == "tautan" {
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
			sb.WriteString("\n_Tautan ini dapat diakses kapan saja menggunakan perintah `!link`._")
			return sb.String()

		case "hapus", "delete", "rm":
			if isGroup && !isAdmin {
				return "❌ *Akses Ditolak: Hanya Admin Grup yang dapat menghapus tautan.*"
			}

			if len(parts) < 3 {
				return "⚠️ *ID Tautan Tidak Valid!*\nFormat: `!link hapus [ID]` (Contoh: `!link hapus 1`)\nKetik `!link` untuk melihat daftar ID tautan."
			}

			idStr := strings.TrimPrefix(parts[2], "#")
			id, err := strconv.ParseInt(idStr, 10, 64)
			if err != nil || id <= 0 {
				return "⚠️ *ID Tautan Harus Berupa Angka Positif!*\nContoh: `!link hapus 1`"
			}

			deleted, err := lm.DeleteLink(scopeJID, id)
			if err != nil {
				return fmt.Sprintf("❌ Terjadi kesalahan saat menghapus tautan: %v", err)
			}
			if !deleted {
				return fmt.Sprintf("⚠️ Tautan dengan ID #%d tidak ditemukan pada chat/grup ini.", id)
			}

			return fmt.Sprintf("🗑️ *TAUTAN BERHASIL DIHAPUS*\nTautan #%d telah dihapus dari daftar.", id)

		case "drive", "gdrive":
			links, err := lm.GetLinksByCategory(scopeJID, "drive")
			if err != nil {
				return fmt.Sprintf("❌ Gagal memuat tautan: %v", err)
			}
			return lm.FormatDriveShortcut(links, isGroup)

		case "zoom", "meet", "gmeet":
			links, err := lm.GetLinksByCategory(scopeJID, "meeting")
			if err != nil {
				return fmt.Sprintf("❌ Gagal memuat tautan: %v", err)
			}
			return lm.FormatMeetingShortcut(links, isGroup)

		case "repo", "github":
			links, err := lm.GetLinksByCategory(scopeJID, "repo")
			if err != nil {
				return fmt.Sprintf("❌ Gagal memuat tautan: %v", err)
			}
			return lm.FormatLinkList(links, isGroup, "🐙 *REPOSITORI & PROYEK (GITHUB / GITLAB)*")

		case "portal", "siakad":
			links, err := lm.GetLinksByCategory(scopeJID, "portal")
			if err != nil {
				return fmt.Sprintf("❌ Gagal memuat tautan: %v", err)
			}
			return lm.FormatLinkList(links, isGroup, "🌐 *PORTAL AKADEMIK (SIAKAD / LMS)*")

		default:
			keyword := subCmd
			if (subCmd == "cari" || subCmd == "search") && len(parts) > 2 {
				keyword = strings.Join(parts[2:], " ")
			} else if len(parts) > 1 {
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
