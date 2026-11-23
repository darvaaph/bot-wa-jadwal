package bot

import (
	"context"
	"strings"
	"testing"
	"time"

	"bot-jadwal/internal/chat"

	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

func TestHandleIncomingMessage_RateLimiter(t *testing.T) {
	testLimiter := NewRateLimiter(1 * time.Second)
	SetDefaultCommandLimiter(testLimiter)
	defer SetDefaultCommandLimiter(NewRateLimiter(2 * time.Second))

	senderJID := types.NewJID("62855554444", types.DefaultUserServer)
	chatJID := types.NewJID("123456-group@g.us", types.GroupServer)

	makeMsg := func(id string, text string) *events.Message {
		return &events.Message{
			Info: types.MessageInfo{
				MessageSource: types.MessageSource{
					Chat:    chatJID,
					Sender:  senderJID,
					IsGroup: true,
				},
				ID: types.MessageID(id),
			},
			Message: &waE2E.Message{
				Conversation: proto.String(text),
			},
		}
	}

	nonCmd := makeMsg("MSG-1", "halo semuanya")
	HandleIncomingMessage(nil, nonCmd, nil, nil, nil, nil, nil)
	if testLimiter.Remaining(senderJID.User) != 0 {
		t.Errorf("Expected non-command message in group to NOT trigger rate limiter")
	}

	cmd1 := makeMsg("MSG-2", "!jadwal")
	HandleIncomingMessage(nil, cmd1, nil, nil, nil, nil, nil)
	if rem := testLimiter.Remaining(senderJID.User); rem <= 0 {
		t.Errorf("Expected command message to register cooldown, got %v", rem)
	}

	cmd2 := makeMsg("MSG-3", "!tugas")
	HandleIncomingMessage(nil, cmd2, nil, nil, nil, nil, nil)
	if testLimiter.Allow(senderJID.User) {
		t.Errorf("Expected immediate subsequent command to be blocked by rate limiter")
	}

	testLimiter.Reset(senderJID.User)
	if !testLimiter.Allow(senderJID.User) {
		t.Errorf("Expected command to be allowed after cooldown reset")
	}
}

func TestHandleIncomingMessage_WithV1Cutover(t *testing.T) {
	senderJID := types.NewJID("62855554444", types.DefaultUserServer)
	chatJID := types.NewJID("120363001234567890@g.us", types.GroupServer)

	msg := &events.Message{
		Info: types.MessageInfo{
			ID:      types.MessageID("MSG-TEST-V1"),
			Sender:  senderJID,
			Chat:    chatJID,
			IsGroup: true,
		},
		Message: &waE2E.Message{
			Conversation: proto.String("!jadwal"),
		},
	}

	// Memastikan HandleIncomingMessage berjalan aman dengan v1DB nil / tanpa error recovery
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("HandleIncomingMessage panic: %v", r)
		}
	}()

	HandleIncomingMessage(nil, msg, nil, nil, nil, nil, nil, nil)
}

func TestKanalCommand_StaysReadable(t *testing.T) {
	// BE-015: !kanal adalah perintah baca (balas JID), bukan mutasi —
	// tidak boleh dialihkan ke dashboard oleh MutationRedirectEntity.
	for _, text := range []string{"!kanal", "!channel", "!jid", "/kanal"} {
		if entity, ok := MutationRedirectEntity(text, true); ok {
			t.Fatalf("%s dialihkan sebagai %s, seharusnya tidak", text, entity)
		}
		if entity, ok := MutationRedirectEntity(text, false); ok {
			t.Fatalf("%s (DM) dialihkan sebagai %s, seharusnya tidak", text, entity)
		}
	}
	got := kanalReplyText("120363000000000001@g.us", true)
	if !strings.Contains(got, "120363000000000001@g.us") || !strings.Contains(got, "grup") {
		t.Fatalf("balasan grup harus memuat JID + jenis: %s", got)
	}
	got = kanalReplyText("62812@s.whatsapp.net", false)
	if !strings.Contains(got, "62812@s.whatsapp.net") || !strings.Contains(got, "pribadi") {
		t.Fatalf("balasan DM harus memuat JID + jenis: %s", got)
	}
}

func TestPJCommand_StaysReadable(t *testing.T) {
	for _, text := range []string{"!pj", "/pj", "#pj", "!daftarpj", "!listpj", "!pj basis data"} {
		if entity, ok := MutationRedirectEntity(text, true); ok {
			t.Fatalf("%s (grup) dialihkan sebagai %s, seharusnya tidak", text, entity)
		}
		if entity, ok := MutationRedirectEntity(text, false); ok {
			t.Fatalf("%s (DM) dialihkan sebagai %s, seharusnya tidak", text, entity)
		}
	}
	for _, text := range []string{"pj", "daftarpj", "listpj", "pj ti2101"} {
		if entity, ok := MutationRedirectEntity(text, false); ok {
			t.Fatalf("%s (DM tanpa prefix) dialihkan sebagai %s, seharusnya tidak", text, entity)
		}
	}
}

func TestHandlePJCommand_Routing(t *testing.T) {
	svc := setupTestV1DB(t)
	ctx := context.Background()
	if _, err := svc.db.ExecContext(ctx, `
		INSERT INTO users (id, identity_key, display_name, password_hash, status)
		VALUES (2, 'private:pj-2', 'Budi Santoso', 'disabled', 'ACTIVE');
		INSERT INTO role_assignments (user_id, role, scope_type, class_id, semester_id, course_offering_id)
		VALUES (2, 'PJ', 'COURSE_OFFERING', 1, 1, 1);
	`); err != nil {
		t.Fatalf("seed PJ gagal: %v", err)
	}

	csm, err := chat.NewChatSettingsManager(svc.db)
	if err != nil {
		t.Fatalf("NewChatSettingsManager gagal: %v", err)
	}
	v1Class, err := svc.FindClass(ctx, "d4-ti-2024-a")
	if err != nil || v1Class == nil {
		t.Fatalf("FindClass gagal: %v", err)
	}

	// 1. Chat belum terikat kelas -> balas onboarding, dan BuildUnconfiguredMenu memuat !pj
	reply, _, handled := handlePJCommand(ctx, "!pj", true, "", csm, svc, nil)
	if !handled || !strings.Contains(reply, "KELAS BELUM DIATUR") {
		t.Fatalf("chat belum terikat kelas seharusnya balas onboarding, got handled=%v reply=%s", handled, reply)
	}
	if menu := csm.BuildUnconfiguredMenu(true); !strings.Contains(menu, "!pj") {
		t.Fatalf("BuildUnconfiguredMenu(true) seharusnya memuat !pj: %s", menu)
	}
	if menu := csm.BuildUnconfiguredMenu(false); !strings.Contains(menu, "!pj") {
		t.Fatalf("BuildUnconfiguredMenu(false) seharusnya memuat !pj: %s", menu)
	}

	// 2. Kelas legacy (non-v1) -> pesan informatif v1-only
	reply, _, handled = handlePJCommand(ctx, "!pj", true, "D4-TI-LEGACY", csm, svc, nil)
	if !handled || !strings.Contains(reply, "sistem v1") {
		t.Fatalf("kelas non-v1 seharusnya mendapat pesan v1-only, got handled=%v reply=%s", handled, reply)
	}

	// 3. Grup dengan prefix !, /, # dan alias + filter argumen
	for _, cmd := range []string{"!pj", "/pj", "#pj", "!daftarpj", "!listpj"} {
		reply, _, handled = handlePJCommand(ctx, cmd, true, "D4-TI-2024-A", csm, svc, v1Class)
		if !handled || !strings.Contains(reply, "Budi Santoso") {
			t.Fatalf("perintah grup %q gagal: handled=%v reply=%s", cmd, handled, reply)
		}
	}
	reply, _, handled = handlePJCommand(ctx, "!pj Struktur", true, "D4-TI-2024-A", csm, svc, v1Class)
	if !handled || !strings.Contains(reply, "Struktur Data") || strings.Contains(reply, "Algoritma") {
		t.Fatalf("filter !pj Struktur gagal: handled=%v reply=%s", handled, reply)
	}

	// 4. DM tanpa prefix maupun dengan prefix
	for _, cmd := range []string{"pj", "!pj", "/pj", "pj TI2101"} {
		reply, _, handled = handlePJCommand(ctx, cmd, false, "D4-TI-2024-A", csm, svc, v1Class)
		if !handled || !strings.Contains(reply, "Budi Santoso") {
			t.Fatalf("perintah DM %q gagal: handled=%v reply=%s", cmd, handled, reply)
		}
	}

	// 5. Perintah lain tidak ditangkap
	if _, _, handled = handlePJCommand(ctx, "!jadwal", true, "D4-TI-2024-A", csm, svc, v1Class); handled {
		t.Fatalf("!jadwal tidak boleh ditangani oleh handlePJCommand")
	}
}

func TestLinkRestrictedDMMessage(t *testing.T) {
	withSlug := linkRestrictedDMMessage("d4-ti-2025-a")
	if !strings.Contains(withSlug, "AKSES TAUTAN DIBATASI") {
		t.Fatalf("harus memuat judul pembatasan, got: %s", withSlug)
	}
	if !strings.Contains(withSlug, "/c/d4-ti-2025-a") {
		t.Fatalf("harus memuat slug portal kelas, got: %s", withSlug)
	}
	if !strings.Contains(withSlug, "Grup WhatsApp resmi") {
		t.Fatalf("harus menyebutkan alternatif grup resmi, got: %s", withSlug)
	}

	withoutSlug := linkRestrictedDMMessage("")
	if !strings.Contains(withoutSlug, "Web Jadwal & Tugas") {
		t.Fatalf("harus memuat web jadwal & tugas meski tanpa slug, got: %s", withoutSlug)
	}
	if !strings.Contains(withoutSlug, "/c/") {
		t.Fatalf("harus memuat /c/ meski tanpa slug, got: %s", withoutSlug)
	}
}

func TestPortalMessage(t *testing.T) {
	withSlug := PortalMessage("d4-ti-2024-a", "D4-TI 2A")
	if !strings.Contains(withSlug, "/c/d4-ti-2024-a") {
		t.Fatalf("harus memuat slug kelas, got: %s", withSlug)
	}
	if !strings.Contains(withSlug, "D4-TI 2A") {
		t.Fatalf("harus memuat nama kelas, got: %s", withSlug)
	}

	withoutSlug := PortalMessage("", "")
	if !strings.Contains(withoutSlug, "/c/") {
		t.Fatalf("harus memuat /c/ jika tanpa slug, got: %s", withoutSlug)
	}
	if strings.Contains(withoutSlug, "8080/ (") {
		t.Fatalf("tidak boleh memuat root / polos, got: %s", withoutSlug)
	}
}
