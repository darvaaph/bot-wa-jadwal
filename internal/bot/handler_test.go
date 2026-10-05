package bot

import (
	"strings"
	"testing"
	"time"

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
	HandleIncomingMessage(nil, nonCmd, nil, nil, nil, nil, nil, nil)
	if testLimiter.Remaining(senderJID.User) != 0 {
		t.Errorf("Expected non-command message in group to NOT trigger rate limiter")
	}

	cmd1 := makeMsg("MSG-2", "!jadwal")
	HandleIncomingMessage(nil, cmd1, nil, nil, nil, nil, nil, nil)
	if rem := testLimiter.Remaining(senderJID.User); rem <= 0 {
		t.Errorf("Expected command message to register cooldown, got %v", rem)
	}

	cmd2 := makeMsg("MSG-3", "!tugas")
	HandleIncomingMessage(nil, cmd2, nil, nil, nil, nil, nil, nil)
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

	HandleIncomingMessage(nil, msg, nil, nil, nil, nil, nil, nil, nil)
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
