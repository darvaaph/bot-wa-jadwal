package bot

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"bot-jadwal/internal/chat"
	"bot-jadwal/internal/link"
	"bot-jadwal/internal/schedule"
	"bot-jadwal/internal/task"
	"bot-jadwal/internal/util"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

var defaultGroupAdminResolver = chat.NewGroupAdminResolver(3 * time.Minute)
var defaultCommandLimiter = NewRateLimiter(2 * time.Second)

// SetDefaultCommandLimiter mengganti instance default command rate limiter (berguna untuk testing/kustomisasi)
func SetDefaultCommandLimiter(limiter *RateLimiter) {
	defaultCommandLimiter = limiter
}

var tugasMutationWords = []string{"tambah", "add", "hapus", "delete", "rm", "selesai", "done", "edit", "update", "mundur", "perpanjang", "ganti"}

var linkMutationWords = []string{"tambah", "add", "hapus", "delete", "rm", "edit", "ubah"}

var scheduleMutationRoots = []string{"pindah", "ganti", "reschedule", "kosong", "batal", "cancel", "libur", "holiday", "kuliahganti", "tambahkelas", "extraclass", "batalganti", "hapusganti", "rmganti"}

// MutationRedirectEntity memeriksa apakah pesan adalah perintah mutasi yang
// sudah dipensiunkan. Mengembalikan label entitas untuk pesan pengalihan.
// Perintah baca (jadwal, daftar tugas, tautan, portal) tidak cocok.
// kanalReplyText menyusun teks balasan JID chat untuk perintah !kanal.
// Fungsi murni agar isi balasan dapat diuji tanpa klien WhatsApp.
func kanalReplyText(chatJID string, isGroup bool) string {
	if isGroup {
		return fmt.Sprintf("*ID GRUP WHATSAPP (JID)*\n──────────\nID grup ini: `%s`\n\nUntuk menghubungkan grup ini dengan jadwal kuliah:\n1. Salin ID di atas.\n2. Buka Dashboard (KM / Admin) -> menu *Kanal WhatsApp*.\n3. Tempel pada kolom *JID Grup WhatsApp* lalu simpan.", chatJID)
	}
	return fmt.Sprintf("*ID CHAT PRIBADI (JID)*\n──────────\nNomor/ID pribadi Anda: `%s`\n\nUntuk memilih kelas di chat pribadi, Anda cukup mengetik perintah:\n`!setkelas [nama_kelas]` (contoh: `!setkelas D4-TI-2025-A` atau `!setkelas 2a`).", chatJID)
}

// linkRestrictedDMMessage menyusun teks pemberitahuan pembatasan akses tautan di chat pribadi.
func linkRestrictedDMMessage(slug string) string {
	portalURL := util.PublicBaseURL() + "/"
	if slug != "" {
		portalURL = fmt.Sprintf("%s/c/%s", util.PublicBaseURL(), slug)
	}
	var sb strings.Builder
	sb.WriteString("*AKSES TAUTAN DIBATASI*\n")
	sb.WriteString("──────────\n")
	sb.WriteString("Untuk keamanan dan privasi perkuliahan (Google Meet, Zoom, atau Google Drive), daftar tautan kelas hanya dapat diakses melalui:\n")
	sb.WriteString("- Grup WhatsApp resmi kelas yang terdaftar\n")
	sb.WriteString(fmt.Sprintf("- Web Portal Kelas: %s", portalURL))
	return sb.String()
}

func MutationRedirectEntity(msgText string, isGroup bool) (string, bool) {
	clean := strings.TrimSpace(msgText)
	if clean == "" {
		return "", false
	}
	lower := strings.ToLower(clean)
	hasSymbol := strings.HasPrefix(lower, "!") || strings.HasPrefix(lower, "/") || strings.HasPrefix(lower, "#")
	if isGroup && !hasSymbol {
		return "", false
	}
	cmdName := lower
	if hasSymbol {
		cmdName = lower[1:]
	}
	fields := strings.Fields(cmdName)
	if len(fields) == 0 {
		return "", false
	}
	root := fields[0]

	switch root {
	case "tugas":
		if len(fields) > 1 && util.Contains(tugasMutationWords, fields[1]) {
			return "tugas", true
		}
		return "", false
	case "link", "tautan":
		if len(fields) > 1 && util.Contains(linkMutationWords, fields[1]) {
			return "tautan", true
		}
		return "", false
	default:
		if util.Contains(scheduleMutationRoots, root) {
			return "jadwal kuliah", true
		}
		return "", false
	}
}

// RedirectMessage menyusun pesan pengalihan standar ke Web Dashboard.
func RedirectMessage(entity string) string {
	return util.DashboardRedirectNotice(entity)
}

// PortalMessage menyusun balasan perintah !portal/!web.
func PortalMessage() string {
	var sb strings.Builder
	sb.WriteString("*PORTAL & DASHBOARD*\n")
	sb.WriteString("──────────\n")
	sb.WriteString("*Portal Kelas (mahasiswa):*\n")
	sb.WriteString(util.PublicBaseURL() + "/ (pilih kelas Anda)\n\n")
	sb.WriteString("*Dashboard Pengelola (PJ/KM):*\n")
	sb.WriteString(util.PublicBaseURL() + "/login.html\n")
	sb.WriteString("Kelola tugas, jadwal, dan materi dengan login pengurus.\n\n")
	return sb.String()
}

// ResolveSenderAdmin mengembalikan status hak akses admin (selalu true di DM pribadi, atau cek admin grup di grup WA)
func ResolveSenderAdmin(ctx context.Context, client *whatsmeow.Client, isGroup bool, groupJID, senderJID, senderAltJID types.JID) bool {
	return defaultGroupAdminResolver.ResolveSenderAdmin(ctx, client, isGroup, groupJID, senderJID, senderAltJID)
}

func InvalidateGroupAdminCache(groupJID types.JID) {
	defaultGroupAdminResolver.Invalidate(groupJID)
}

// HandleIncomingMessage memproses setiap pesan masuk secara asinkron di dalam goroutine independen
func HandleIncomingMessage(
	client *whatsmeow.Client,
	v *events.Message,
	classManager *schedule.ClassManager,
	chatSettingsManager *chat.ChatSettingsManager,
	taskManager *task.TaskManager,
	overrideManager *schedule.OverrideManager,
	linkManager *link.LinkManager,
	v1DB ...*sql.DB,
) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("⚠️ [Panic Recovery] Terjadi kesalahan tidak terduga saat memproses pesan dari %s: %v\n", v.Info.Sender.User, r)
		}
	}()

	var msgText string
	if v.Message.GetConversation() != "" {
		msgText = v.Message.GetConversation()
	} else if v.Message.GetExtendedTextMessage() != nil && v.Message.GetExtendedTextMessage().GetText() != "" {
		msgText = v.Message.GetExtendedTextMessage().GetText()
	}

	msgText = strings.TrimSpace(msgText)
	if msgText == "" {
		return
	}

	// Batasi command per pengirim agar bot tidak memicu proteksi spam WhatsApp.
	if IsCommandMessage(msgText, v.Info.IsGroup) {
		senderKey := v.Info.Sender.ToNonAD().User
		if senderKey == "" {
			senderKey = v.Info.Sender.String()
		}
		if defaultCommandLimiter != nil && !defaultCommandLimiter.Allow(senderKey) {
			fmt.Printf("⏳ [Rate Limit] Perintah dari %s diabaikan (cooldown aktif)\n", senderKey)
			return
		}
	}

	fmt.Printf("[Pesan Masuk dari %s]: %s\n", v.Info.Sender.User, msgText)

	reply := func(replyText, emoji string, typingDuration time.Duration, actionName string) {
		ReplyWithTyping(
			context.Background(),
			client,
			v.Info.Chat,
			v.Info.Sender,
			v.Info.ID,
			v.Message,
			v.Info.IsGroup,
			replyText,
			emoji,
			typingDuration,
			actionName,
		)
	}

	if classManager == nil {
		return
	}
	if chatSettingsManager != nil {
		seenName := ""
		if !v.Info.IsGroup {
			seenName = v.Info.Sender.ToNonAD().User
		}
		chatSettingsManager.NoteSeenChat(v.Info.Chat.String(), seenName, v.Info.IsGroup)
	}

	// Antarmuka WhatsApp murni read-only: perintah mutasi dialihkan ke dashboard.
	if entity, ok := MutationRedirectEntity(msgText, v.Info.IsGroup); ok {
		reply(RedirectMessage(entity), "⚠️", 600*time.Millisecond, "pengalihan dashboard")
		return
	}

	if util.MatchCommandPrefix(msgText, v.Info.IsGroup, "portal", "web") {
		reply(PortalMessage(), "🌐", 600*time.Millisecond, "perintah portal")
		return
	}

	// BE-015: balas JID chat saat ini agar admin tinggal salin ke form
	// Kelola Kanal di dashboard. Baca saja, aman untuk semua pengirim.
	if util.MatchCommandPrefix(msgText, v.Info.IsGroup, "kanal", "channel", "jid") {
		reply(kanalReplyText(v.Info.Chat.String(), v.Info.IsGroup), "📡", 600*time.Millisecond, "perintah kanal")
		return
	}

	var activeClassID string
	if chatSettingsManager != nil {
		activeClassID = chatSettingsManager.GetClass(v.Info.Chat.String())
	}
	activeJadwal := classManager.GetClassOrDefault(activeClassID)

	var v1Service *V1BotService
	var v1Class *V1ClassInfo
	if len(v1DB) > 0 && v1DB[0] != nil {
		v1Service = NewV1BotService(v1DB[0])
		if activeClassID != "" {
			v1Class, _ = v1Service.FindClass(context.Background(), activeClassID)
		}
	}

	// 1. Handler Khusus Perintah Pengaturan Kelas (!kelas / !daftarkelas / !setkelas / !pilihkelas / !resetkelas)
	if chatSettingsManager != nil && util.MatchCommandPrefix(msgText, v.Info.IsGroup, "kelas", "daftarkelas", "setkelas", "pilihkelas", "resetkelas") {
		isAdmin := ResolveSenderAdmin(context.Background(), client, v.Info.IsGroup, v.Info.Chat, v.Info.Sender, v.Info.SenderAlt)
		classReply := chatSettingsManager.HandleCommand(v.Info.Chat.String(), v.Info.IsGroup, v.Info.Sender.String(), isAdmin, msgText, classManager)
		if classReply != "" {
			// Sinkronisasi otomatis ke whatsapp_channels jika target kelas adalah kelas pilot v1.
			// Hanya grup (JID grup valid sebagai kanal); DM adalah preferensi pribadi,
			// bukan kanal kelas. Tak pernah pindah kelas (BindChannel menolak).
			if v.Info.IsGroup && v1Service != nil {
				newClassID := chatSettingsManager.GetClass(v.Info.Chat.String())
				if matchedV1, err := v1Service.FindClass(context.Background(), newClassID); err == nil && matchedV1 != nil {
					groupName := "Grup Chat"
					if v.Info.IsGroup && client != nil {
						info, err := defaultGroupAdminResolver.GetGroupInfo(context.Background(), client, v.Info.Chat)
						if err == nil && info != nil && info.Name != "" {
							groupName = info.Name
						}
					}
					_ = v1Service.BindChannel(context.Background(), matchedV1.ID, v.Info.Chat.String(), groupName)
				}
			}
			reply(classReply, "🏫", 600*time.Millisecond, "perintah kelas")
			return
		}
	}

	if util.MatchCommandPrefix(msgText, v.Info.IsGroup, "reload") {
		count, errs := classManager.ReloadAll()
		var reloadReply string
		if len(errs) > 0 {
			reloadReply = fmt.Sprintf("⚠️ Berhasil memuat ulang %d kelas, namun terdapat error: %v", count, errs)
		} else {
			reloadReply = fmt.Sprintf("🔄 *BERHASIL MEMUAT ULANG JADWAL!*\n──────────\nSeluruh konfigurasi jadwal (%d kelas) berhasil disegarkan dari disk ke memori.", count)
		}
		reply(reloadReply, "🔄", 600*time.Millisecond, "perintah reload")
		return
	}

	if util.MatchCommandPrefix(msgText, v.Info.IsGroup, "reminder", "pengingat") {
		reply("⏰ Pengingat kelas diatur melalui Dashboard Pengelola → Pengaturan Pengingat. Tautkan kanal WhatsApp kelas melalui Dashboard System Admin agar pesan terjadwal dapat terkirim.\n"+util.PublicBaseURL()+"/login.html", "⏰", 600*time.Millisecond, "pengaturan reminder")
		return
	}

	// 4. Handler Khusus Perintah Tugas (!tugas)
	if util.MatchCommandPrefix(msgText, v.Info.IsGroup, "tugas") {
		if activeClassID == "" && chatSettingsManager != nil {
			reply(chatSettingsManager.GetOnboardingPrompt(v.Info.IsGroup), "👋", 600*time.Millisecond, "onboarding tugas")
			return
		}

		// Jika chat ini terikat pada kelas target pilot v1, layani dari database v1
		if v1Class != nil && v1Service != nil {
			parts := strings.Fields(msgText)
			subCmd := ""
			if len(parts) > 1 {
				subCmd = strings.ToLower(parts[1])
			}

			if subCmd == "selesai" || subCmd == "done" {
				if len(parts) < 3 {
					reply("⚠️ Format salah! Gunakan: `!tugas selesai <ID>`\nContoh: `!tugas selesai 1`", "⚠️", 600*time.Millisecond, "tugas selesai v1")
					return
				}
				idStr := parts[2]
				taskID, err := strconv.ParseInt(idStr, 10, 64)
				if err != nil || taskID <= 0 {
					reply("⚠️ ID tugas harus berupa angka valid!", "⚠️", 600*time.Millisecond, "tugas selesai v1")
					return
				}
				resp, err := v1Service.CompleteTask(context.Background(), v1Class.ID, taskID)
				if err != nil {
					fmt.Printf("⚠️ [CompleteTask Error] gagal menyelesaikan tugas %d untuk kelas %d: %v\n", taskID, v1Class.ID, err)
					reply("⚠️ Tugas tidak ditemukan atau gagal diperbarui. Pastikan nomor ID tugas sudah benar.", "⚠️", 600*time.Millisecond, "tugas selesai v1")
					return
				}
				reply(resp, "✅", 600*time.Millisecond, "tugas selesai v1")
				return
			}

			// Tampilkan daftar tugas aktif dari v1 database
			resp, err := v1Service.GetTasks(context.Background(), v1Class.ID)
			if err != nil {
				fmt.Printf("⚠️ [GetTasks Error] gagal memuat tugas untuk kelas %d: %v\n", v1Class.ID, err)
				reply("⚠️ Terjadi kendala saat memuat daftar tugas. Silakan coba beberapa saat lagi.", "⚠️", 600*time.Millisecond, "daftar tugas v1")
				return
			}
			reply(resp, "📝", 600*time.Millisecond, "daftar tugas v1")
			return
		}

		// Fallback ke legacy taskManager untuk kelas non-pilot
		if taskManager != nil {
			isAdmin := ResolveSenderAdmin(context.Background(), client, v.Info.IsGroup, v.Info.Chat, v.Info.Sender, v.Info.SenderAlt)
			tugasReply := taskManager.HandleCommand(v.Info.Chat.String(), v.Info.IsGroup, v.Info.Sender.String(), isAdmin, msgText, activeJadwal, time.Now(), activeClassID)
			reply(tugasReply, "📝", 600*time.Millisecond, "perintah tugas")
			return
		}
	}

	if overrideManager != nil && util.MatchCommandPrefix(msgText, v.Info.IsGroup, "pindah", "ganti", "kosong", "libur", "kuliahganti", "tambahkelas", "jadwalganti", "overrides", "batalganti") {
		if activeClassID == "" && chatSettingsManager != nil {
			reply(chatSettingsManager.GetOnboardingPrompt(v.Info.IsGroup), "👋", 600*time.Millisecond, "onboarding override")
			return
		}
		isAdmin := ResolveSenderAdmin(context.Background(), client, v.Info.IsGroup, v.Info.Chat, v.Info.Sender, v.Info.SenderAlt)
		overrideReply := overrideManager.HandleCommand(v.Info.Chat.String(), v.Info.IsGroup, v.Info.Sender.String(), isAdmin, msgText, activeJadwal, time.Now())
		reply(overrideReply, "🔄", 600*time.Millisecond, "perintah override")
		return
	}

	// 6. Handler Khusus Perintah Tautan & Materi Penting Kelas (!link, !tautan, !materi, !drive, !gdrive, !zoom, !gmeet, !meet)
	if util.MatchCommandPrefix(msgText, v.Info.IsGroup, "link", "tautan", "materi", "drive", "gdrive", "zoom", "gmeet", "meet") {
		if activeClassID == "" && chatSettingsManager != nil {
			reply(chatSettingsManager.GetOnboardingPrompt(v.Info.IsGroup), "👋", 600*time.Millisecond, "onboarding link")
			return
		}

		// Keamanan Tautan: Di chat pribadi (DM), akses tautan dibatasi dan dialihkan ke Web Portal / Grup Resmi.
		if !v.Info.IsGroup {
			slug := ""
			if v1Class != nil {
				slug = v1Class.Slug
			} else if activeClassID != "" {
				slug = strings.ToLower(activeClassID)
			}
			reply(linkRestrictedDMMessage(slug), "🔒", 600*time.Millisecond, "tautan dibatasi di DM")
			return
		}

		if v1Class != nil && v1Service != nil {
			resp, err := v1Service.GetLinks(context.Background(), v1Class.ID)
			if err != nil {
				fmt.Printf("⚠️ [GetLinks Error] gagal memuat tautan untuk kelas %d: %v\n", v1Class.ID, err)
				reply("⚠️ Terjadi kendala saat memuat tautan kelas. Silakan coba beberapa saat lagi.", "⚠️", 600*time.Millisecond, "daftar tautan v1")
				return
			}
			reply(resp, "🔗", 600*time.Millisecond, "daftar tautan v1")
			return
		}

		if linkManager != nil {
			isAdmin := ResolveSenderAdmin(context.Background(), client, v.Info.IsGroup, v.Info.Chat, v.Info.Sender, v.Info.SenderAlt)
			linkReply := linkManager.HandleCommand(v.Info.Chat.String(), v.Info.IsGroup, v.Info.Sender.String(), isAdmin, msgText)
			reply(linkReply, "🔗", 600*time.Millisecond, "perintah tautan")
			return
		}
	}

	// 7. Jika kelas target adalah kelas pilot v1, layani perintah jadwal, menu, dan pencarian dari database v1
	if v1Class != nil && v1Service != nil {
		if util.MatchCommandPrefix(msgText, v.Info.IsGroup, "jadwal", "next", "sekarang", "kuliah", "ongoing", "senin", "selasa", "rabu", "kamis", "jumat", "jum'at", "sabtu", "minggu", "hari ini", "hariini", "today", "besok", "tomorrow", "seminggu", "semua", "all") {
			schedReply, err := v1Service.HandleScheduleCommand(context.Background(), v1Class.ID, msgText, time.Now())
			if err == nil && schedReply != "" {
				reply(schedReply, "📅", 700*time.Millisecond, "jadwal v1")
				return
			}
		}

		if util.MatchCommandPrefix(msgText, v.Info.IsGroup, "keyword", "keywords", "katakunci") {
			kwReply, err := v1Service.GetKeywords(context.Background(), v1Class.ID)
			if err == nil && kwReply != "" {
				reply(kwReply, "📖", 600*time.Millisecond, "keywords v1")
				return
			}
		}

		if util.MatchCommandPrefix(msgText, v.Info.IsGroup, "menu", "help", "bantuan", "panduan") {
			menuReply, err := v1Service.GetMenu(context.Background(), v1Class.ID)
			if err == nil && menuReply != "" {
				reply(menuReply, "📋", 600*time.Millisecond, "menu v1")
				return
			}
		}

		if util.MatchCommandPrefix(msgText, v.Info.IsGroup, "matkul", "matakuliah", "mata kuliah", "daftarmatkul") {
			courseReply, err := v1Service.GetCourses(context.Background(), v1Class.ID)
			if err == nil && courseReply != "" {
				reply(courseReply, "📚", 600*time.Millisecond, "matkul v1")
				return
			}
		}

		if util.MatchCommandPrefix(msgText, v.Info.IsGroup, "dosen") {
			parts := strings.Fields(msgText)
			query := ""
			if len(parts) > 1 {
				query = strings.Join(parts[1:], " ")
			}
			dosenReply, err := v1Service.SearchDosen(context.Background(), v1Class.ID, query)
			if err == nil && dosenReply != "" {
				reply(dosenReply, "👨‍🏫", 600*time.Millisecond, "dosen v1")
				return
			}
		}

		if util.MatchCommandPrefix(msgText, v.Info.IsGroup, "ruang", "ruangan", "lab") {
			parts := strings.Fields(msgText)
			query := ""
			if len(parts) > 1 {
				query = strings.Join(parts[1:], " ")
			}
			ruangReply, err := v1Service.SearchRuang(context.Background(), v1Class.ID, query)
			if err == nil && ruangReply != "" {
				reply(ruangReply, "🏫", 600*time.Millisecond, "ruang v1")
				return
			}
		}

		if util.MatchCommandPrefix(msgText, v.Info.IsGroup, "cari", "search") {
			parts := strings.Fields(msgText)
			query := ""
			if len(parts) > 1 {
				query = strings.Join(parts[1:], " ")
			}
			cariReply, err := v1Service.SearchGlobal(context.Background(), v1Class.ID, query)
			if err == nil && cariReply != "" {
				reply(cariReply, "🔍", 600*time.Millisecond, "cari v1")
				return
			}
		}
	}

	// 8. Proses pesan masuk dengan parser perintah jadwal legacy (menerapkan aturan Hybrid & Override)
	replyText := activeJadwal.ProcessMessage(msgText, v.Info.IsGroup, v.Info.Chat.String())

	if replyText != "" {
		if activeClassID == "" && chatSettingsManager != nil {
			if util.IsMenuOrHelpCommand(msgText, v.Info.IsGroup) {
				menuReply := chatSettingsManager.BuildUnconfiguredMenu(v.Info.IsGroup)
				reply(menuReply, "📅", 700*time.Millisecond, "menu unconfigured")
				return
			}
			if strings.Contains(replyText, "tidak dikenali") {
				reply(replyText, "⚠️", 700*time.Millisecond, fmt.Sprintf("perintah '%s'", msgText))
				return
			}
			// Jika chat belum memilih kelas, berikan panduan onboarding alih-alih menampilkan kelas default
			reply(chatSettingsManager.GetOnboardingPrompt(v.Info.IsGroup), "👋", 700*time.Millisecond, "onboarding jadwal")
			return
		}

		reply(replyText, "📅", 700*time.Millisecond, fmt.Sprintf("perintah '%s'", msgText))
	}
}
