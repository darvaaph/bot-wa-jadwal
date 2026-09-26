package bot

import (
	"context"
	"fmt"
	"strings"
	"time"

	"bot-jadwal/internal/chat"
	"bot-jadwal/internal/link"
	"bot-jadwal/internal/reminder"
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

// DashboardBaseURL adalah basis URL Web Dashboard. Ganti dengan domain
// produksi saat deploy (contoh: https://jadwal.kelas.ac.id).
const DashboardBaseURL = "http://localhost:8080"

// PortalBaseURL adalah basis URL Portal Kelas Mahasiswa.
const PortalBaseURL = "http://localhost:8080"

var tugasMutationWords = []string{"tambah", "add", "hapus", "delete", "rm", "selesai", "done", "edit", "update", "mundur", "perpanjang", "ganti"}

var linkMutationWords = []string{"tambah", "add", "hapus", "delete", "rm", "edit", "ubah"}

var scheduleMutationRoots = []string{"pindah", "ganti", "reschedule", "kosong", "batal", "cancel", "libur", "holiday", "kuliahganti", "tambahkelas", "extraclass", "batalganti", "hapusganti", "rmganti"}

// MutationRedirectEntity memeriksa apakah pesan adalah perintah mutasi yang
// sudah dipensiunkan. Mengembalikan label entitas untuk pesan pengalihan.
// Perintah baca (jadwal, daftar tugas, tautan, portal) tidak cocok.
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
	sb.WriteString("🌐 *PORTAL & DASHBOARD*\n")
	sb.WriteString("──────────\n")
	sb.WriteString("📖 *Portal Kelas (mahasiswa):*\n")
	sb.WriteString(PortalBaseURL + "/ (pilih kelas Anda)\n\n")
	sb.WriteString("🛠️ *Dashboard Pengelola (PJ/KM):*\n")
	sb.WriteString(DashboardBaseURL + "/app.html\n")
	sb.WriteString("Kelola tugas, jadwal, dan materi dengan login pengurus.\n\n")
	sb.WriteString("_(Ganti localhost:8080 dengan domain portal Anda di produksi.)_")
	return sb.String()
}

func GetDefaultCommandLimiter() *RateLimiter {
	return defaultCommandLimiter
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
	reminderManager *reminder.ReminderManager,
	taskManager *task.TaskManager,
	overrideManager *schedule.OverrideManager,
	linkManager *link.LinkManager,
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

	lowerMsg := strings.ToLower(msgText)

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

	var activeClassID string
	if chatSettingsManager != nil {
		activeClassID = chatSettingsManager.GetClass(v.Info.Chat.String())
	}
	activeJadwal := classManager.GetClassOrDefault(activeClassID)

	if chatSettingsManager != nil && util.MatchCommandPrefix(msgText, v.Info.IsGroup, "kelas", "daftarkelas", "setkelas", "pilihkelas", "resetkelas") {
		isAdmin := ResolveSenderAdmin(context.Background(), client, v.Info.IsGroup, v.Info.Chat, v.Info.Sender, v.Info.SenderAlt)
		classReply := chatSettingsManager.HandleCommand(v.Info.Chat.String(), v.Info.IsGroup, v.Info.Sender.String(), isAdmin, msgText, classManager)
		if classReply != "" {
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
		parts := strings.Fields(lowerMsg)
		subCmd := ""
		if len(parts) > 1 {
			subCmd = parts[1]
		}

		var reminderReply string
		switch subCmd {
		case "on", "aktif", "start", "enable":
			if activeClassID == "" {
				reminderReply = "⚠️ *PENGINGAT TIDAK DAPAT DIAKTIFKAN*\n──────────\nChat ini belum menentukan kelas perkuliahan.\nSilakan atur kelas terlebih dahulu dengan perintah:\n👉 `!setkelas [nama_kelas]` (Contoh: `!setkelas D4-TI-1A`)\n\nKetik `!daftarkelas` untuk melihat 19 pilihan kelas yang tersedia."
			} else {
				chatJID := v.Info.Chat.String()
				groupName := "Grup Chat"
				if v.Info.IsGroup {
					info, err := defaultGroupAdminResolver.GetGroupInfo(context.Background(), client, v.Info.Chat)
					if err == nil && info != nil && info.Name != "" {
						groupName = info.Name
					}
				}
				_, reminderReply = reminderManager.AddGroup(chatJID, groupName)
			}

		case "off", "nonaktif", "stop", "disable", "matikan":
			chatJID := v.Info.Chat.String()
			_, reminderReply = reminderManager.RemoveGroup(chatJID)

		case "test", "tes", "try":
			if activeClassID == "" && chatSettingsManager != nil {
				reminderReply = chatSettingsManager.GetOnboardingPrompt(v.Info.IsGroup)
			} else {
				reminderReply = fmt.Sprintf("🧪 *[SIMULASI PENGINGAT PAGI]*\n\n%s", reminder.BuildMorningReminder(v.Info.Chat.String(), activeJadwal, taskManager, time.Now(), linkManager))
			}

		default:
			reminderReply = reminderManager.Status(v.Info.Chat.String())
		}

		reply(reminderReply, "⏰", 600*time.Millisecond, "perintah reminder")
		return
	}

	if taskManager != nil && util.MatchCommandPrefix(msgText, v.Info.IsGroup, "tugas") {
		if activeClassID == "" && chatSettingsManager != nil {
			reply(chatSettingsManager.GetOnboardingPrompt(v.Info.IsGroup), "👋", 600*time.Millisecond, "onboarding tugas")
			return
		}
		isAdmin := ResolveSenderAdmin(context.Background(), client, v.Info.IsGroup, v.Info.Chat, v.Info.Sender, v.Info.SenderAlt)
		tugasReply := taskManager.HandleCommand(v.Info.Chat.String(), v.Info.IsGroup, v.Info.Sender.String(), isAdmin, msgText, activeJadwal, time.Now(), activeClassID)
		reply(tugasReply, "📝", 600*time.Millisecond, "perintah tugas")
		return
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

	if linkManager != nil && util.MatchCommandPrefix(msgText, v.Info.IsGroup, "link", "tautan", "drive", "gdrive", "zoom", "gmeet", "meet") {
		isAdmin := ResolveSenderAdmin(context.Background(), client, v.Info.IsGroup, v.Info.Chat, v.Info.Sender, v.Info.SenderAlt)
		linkReply := linkManager.HandleCommand(v.Info.Chat.String(), v.Info.IsGroup, v.Info.Sender.String(), isAdmin, msgText)
		reply(linkReply, "🔗", 600*time.Millisecond, "perintah tautan")
		return
	}

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
