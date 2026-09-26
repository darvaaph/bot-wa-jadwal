package link

import (
	"bot-jadwal/internal/academic"
	"bot-jadwal/internal/util"
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"strings"
	"time"
)

var ErrUnmappedScope = academic.ErrUnmappedScope

type BackfillReport struct {
	TotalLegacy int
	Migrated    int
	Skipped     int
	Errors      []string
}

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
	_, _ = lm.BackfillLegacyLinksContext(ctx)
}

// BackfillLegacyLinksContext melakukan migrasi data dari tabel legacy class_links ke materials dengan manifest error.
func (lm *LinkManager) BackfillLegacyLinksContext(ctx context.Context) (*BackfillReport, error) {
	report := &BackfillReport{}
	var tableName string
	err := lm.db.QueryRowContext(ctx, "SELECT name FROM sqlite_master WHERE type='table' AND name='class_links'").Scan(&tableName)
	if err != nil || tableName == "" {
		return report, nil
	}

	batchID, err := lm.academicRepo.EnsureMigrationBatch(ctx, "class_links")
	if err != nil {
		report.Errors = append(report.Errors, fmt.Sprintf("gagal memastikan migration batch: %v", err))
	}

	rows, err := lm.db.QueryContext(ctx, `
		SELECT scope_jid, is_group, title, url, category, description, created_by
		FROM class_links
	`)
	if err != nil {
		report.Errors = append(report.Errors, fmt.Sprintf("gagal query class_links: %v", err))
		if batchID > 0 {
			_ = lm.academicRepo.RecordImportError(ctx, batchID, "class_links", "table", "QUERY_FAILED", err.Error())
			_ = lm.academicRepo.UpdateImportBatchStats(ctx, batchID, 0, 0, 1)
		}
		return report, err
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

	report.TotalLegacy = len(legacyItems)
	for _, item := range legacyItems {
		classID, _, err := lm.academicRepo.ResolveClassIDFromScope(ctx, item.scopeJID)
		if err != nil {
			report.Skipped++
			errDetail := fmt.Sprintf("unmapped scope %s: %v", item.scopeJID, err)
			report.Errors = append(report.Errors, errDetail)
			if batchID > 0 {
				_ = lm.academicRepo.RecordImportError(ctx, batchID, "class_links", "scope_jid", "UNMAPPED_SCOPE", errDetail)
			}
			continue
		}

		userID, err := lm.academicRepo.EnsureUser(ctx, item.createdBy, item.createdBy)
		if err != nil {
			report.Skipped++
			errDetail := fmt.Sprintf("gagal memastikan user %s: %v", item.createdBy, err)
			report.Errors = append(report.Errors, errDetail)
			if batchID > 0 {
				_ = lm.academicRepo.RecordImportError(ctx, batchID, "class_links", "created_by", "USER_ERROR", errDetail)
			}
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
			var newID int64
			err := lm.db.QueryRowContext(ctx, `
				INSERT INTO materials (class_id, title, material_type, url, description, visibility, status, created_by_user_id)
				VALUES (?, ?, ?, ?, ?, ?, 'ACTIVE', ?)
				RETURNING id;
			`, classID, item.title, matType, item.url, item.description, visibility, userID).Scan(&newID)
			if err != nil {
				report.Skipped++
				errDetail := fmt.Sprintf("gagal insert materials (%s): %v", item.title, err)
				report.Errors = append(report.Errors, errDetail)
				if batchID > 0 {
					_ = lm.academicRepo.RecordImportError(ctx, batchID, "class_links", "materials", "INSERT_FAILED", errDetail)
				}
				continue
			}
			report.Migrated++
		} else {
			report.Migrated++
		}
	}

	if batchID > 0 {
		_ = lm.academicRepo.UpdateImportBatchStats(ctx, batchID, report.TotalLegacy, report.Migrated, len(report.Errors))
	}
	return report, nil
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

// AddLinkContext menambahkan tautan baru ke database target materials dengan context-awareness.
func (lm *LinkManager) AddLinkContext(ctx context.Context, scopeJID string, isGroup bool, title, rawURL, desc, createdBy string) (int64, error) {
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

	classID, _, err := lm.academicRepo.ResolveClassIDFromScope(ctx, scopeJID)
	if err != nil {
		return 0, fmt.Errorf("gagal mengaitkan tautan ke kelas: %w", err)
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
		VALUES (?, ?, ?, ?, ?, ?, 'ACTIVE', ?)
		RETURNING id;
	`
	var insertedID int64
	err = lm.db.QueryRowContext(ctx, query, classID, title, matType, rawURL, desc, visibility, userID).Scan(&insertedID)
	if err != nil {
		return 0, fmt.Errorf("gagal menambahkan tautan ke materials: %w", err)
	}

	return insertedID, nil
}

// AddLink adalah adapter kompatibilitas untuk AddLinkContext
func (lm *LinkManager) AddLink(scopeJID string, isGroup bool, title, rawURL, desc, createdBy string) (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return lm.AddLinkContext(ctx, scopeJID, isGroup, title, rawURL, desc, createdBy)
}

// DeleteLinkContext menghapus tautan dengan context-awareness
func (lm *LinkManager) DeleteLinkContext(ctx context.Context, scopeJID string, id int64) (bool, error) {
	classID, _, err := lm.academicRepo.ResolveClassIDFromScope(ctx, scopeJID)
	if err != nil {
		cls, ensureErr := lm.academicRepo.GetClassByCode(ctx, scopeJID)
		if ensureErr != nil || cls == nil {
			return false, fmt.Errorf("gagal memetakan kelas untuk scope %s: %w", scopeJID, err)
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

// DeleteLink adalah adapter kompatibilitas untuk DeleteLinkContext
func (lm *LinkManager) DeleteLink(scopeJID string, id int64) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return lm.DeleteLinkContext(ctx, scopeJID, id)
}

// GetLinksContext mengambil seluruh tautan aktif pada scope chat terurut berdasarkan kategori utama dengan context-awareness
func (lm *LinkManager) GetLinksContext(ctx context.Context, scopeJID string) ([]LinkItem, error) {
	return lm.queryLinks(ctx, scopeJID, "")
}

// GetLinks adalah adapter kompatibilitas untuk GetLinksContext
func (lm *LinkManager) GetLinks(scopeJID string) ([]LinkItem, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return lm.GetLinksContext(ctx, scopeJID)
}

// GetLinksByCategoryContext mengambil tautan berdasarkan kategori dengan context-awareness
func (lm *LinkManager) GetLinksByCategoryContext(ctx context.Context, scopeJID, category string) ([]LinkItem, error) {
	matType := CategoryToMaterialType(category)
	return lm.queryLinks(ctx, scopeJID, "AND m.material_type = ?", matType)
}

// GetLinksByCategory adalah adapter kompatibilitas untuk GetLinksByCategoryContext
func (lm *LinkManager) GetLinksByCategory(scopeJID, category string) ([]LinkItem, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return lm.GetLinksByCategoryContext(ctx, scopeJID, category)
}

// SearchLinksContext mencari tautan berdasarkan kata kunci dengan context-awareness
func (lm *LinkManager) SearchLinksContext(ctx context.Context, scopeJID, keyword string) ([]LinkItem, error) {
	pattern := "%" + strings.ToLower(strings.TrimSpace(keyword)) + "%"
	return lm.queryLinks(ctx, scopeJID, "AND (LOWER(m.title) LIKE ? OR LOWER(m.description) LIKE ? OR LOWER(m.material_type) LIKE ?)", pattern, pattern, pattern)
}

// SearchLinks adalah adapter kompatibilitas untuk SearchLinksContext
func (lm *LinkManager) SearchLinks(scopeJID, keyword string) ([]LinkItem, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return lm.SearchLinksContext(ctx, scopeJID, keyword)
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
			return util.DashboardRedirectNotice("tautan")

		case "hapus", "delete", "rm":
			return util.DashboardRedirectNotice("tautan")

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

	sb.WriteString("⚠️ *Penambahan, perubahan, dan pembatalan tautan kini hanya melalui Web Dashboard Pengelola:*\n")
	sb.WriteString("👉 http://localhost:8080/app.html (atau domain portal Anda)\n\n")

	sb.WriteString("──────────\n")
	sb.WriteString("_Tips: URL otomatis dinormalisasi menjadi HTTPS agar langsung bisa diklik di ponsel._")
	return sb.String()
}
