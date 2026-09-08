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

// ResolveSenderAdmin mengembalikan status hak akses admin (selalu true di DM pribadi, atau cek admin grup di grup WA)
func ResolveSenderAdmin(ctx context.Context, client *whatsmeow.Client, isGroup bool, groupJID, senderJID, senderAltJID types.JID) bool {
	return defaultGroupAdminResolver.ResolveSenderAdmin(ctx, client, isGroup, groupJID, senderJID, senderAltJID)
}

// InvalidateGroupAdminCache menghapus cache info grup saat terjadi perubahan admin / grup
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

	// Ekstraksi teks pesan dari tipe Conversation atau ExtendedTextMessage
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

	// Log pesan yang diterima di konsol
	fmt.Printf("[Pesan Masuk dari %s]: %s\n", v.Info.Sender.User, msgText)

	lowerMsg := strings.ToLower(msgText)

	// Helper terpusat untuk membalas pesan pengguna dengan Quoted Reply
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

	// Tentukan jadwal kelas aktif untuk chat/grup ini secara dinamis (Multi-Tenant)
	var activeClassID string
	if chatSettingsManager != nil {
		activeClassID = chatSettingsManager.GetClass(v.Info.Chat.String())
	}
	activeJadwal := classManager.GetClassOrDefault(activeClassID)

	// 1. Handler Khusus Perintah Pengaturan Kelas (!kelas / !daftarkelas / !setkelas / !pilihkelas / !resetkelas)
	if chatSettingsManager != nil && util.MatchCommandPrefix(msgText, v.Info.IsGroup, "kelas", "daftarkelas", "setkelas", "pilihkelas", "resetkelas") {
		isAdmin := ResolveSenderAdmin(context.Background(), client, v.Info.IsGroup, v.Info.Chat, v.Info.Sender, v.Info.SenderAlt)
		classReply := chatSettingsManager.HandleCommand(v.Info.Chat.String(), v.Info.IsGroup, v.Info.Sender.String(), isAdmin, msgText, classManager)
		if classReply != "" {
			reply(classReply, "🏫", 600*time.Millisecond, "perintah kelas")
			return
		}
	}

	// 2. Handler Khusus Perintah Reload Jadwal Seluruh Kelas (!reload)
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

	// 3. Handler Khusus Perintah Pengingat Otomatis (!reminder / !pengingat)
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

	// 4. Handler Khusus Perintah Tugas (!tugas)
	if taskManager != nil && util.MatchCommandPrefix(msgText, v.Info.IsGroup, "tugas") {
		if activeClassID == "" && chatSettingsManager != nil {
			reply(chatSettingsManager.GetOnboardingPrompt(v.Info.IsGroup), "👋", 600*time.Millisecond, "onboarding tugas")
			return
		}
		isAdmin := ResolveSenderAdmin(context.Background(), client, v.Info.IsGroup, v.Info.Chat, v.Info.Sender, v.Info.SenderAlt)
		tugasReply := taskManager.HandleCommand(v.Info.Chat.String(), v.Info.IsGroup, v.Info.Sender.String(), isAdmin, msgText, activeJadwal, time.Now())
		reply(tugasReply, "📝", 600*time.Millisecond, "perintah tugas")
		return
	}

	// 5. Handler Khusus Perintah Jadwal Pengganti / Override (!pindah, !kosong, !kuliahganti, !jadwalganti, !batalganti)
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

	// 6. Handler Khusus Perintah Tautan Penting Kelas (!link, !tautan, !drive, !gdrive, !zoom, !gmeet, !meet)
	if linkManager != nil && util.MatchCommandPrefix(msgText, v.Info.IsGroup, "link", "tautan", "drive", "gdrive", "zoom", "gmeet", "meet") {
		isAdmin := ResolveSenderAdmin(context.Background(), client, v.Info.IsGroup, v.Info.Chat, v.Info.Sender, v.Info.SenderAlt)
		linkReply := linkManager.HandleCommand(v.Info.Chat.String(), v.Info.IsGroup, v.Info.Sender.String(), isAdmin, msgText)
		reply(linkReply, "🔗", 600*time.Millisecond, "perintah tautan")
		return
	}

	// 7. Proses pesan masuk dengan parser perintah jadwal (menerapkan aturan Hybrid & Override)
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
