package bot

import (
	"testing"
	"time"

	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

func TestHandleIncomingMessage_RateLimiter(t *testing.T) {
	// Pasang limiter khusus dengan cooldown 1 detik untuk pengujian
	testLimiter := NewRateLimiter(1 * time.Second)
	SetDefaultCommandLimiter(testLimiter)
	defer SetDefaultCommandLimiter(NewRateLimiter(2 * time.Second))

	senderJID := types.NewJID("62855554444", types.DefaultUserServer)
	chatJID := types.NewJID("123456-group@g.us", types.GroupServer)

	makeMsg := func(id string, text string) *events.Message {
		return &events.Message{
			Info: types.MessageInfo{
				ID:      types.MessageID(id),
				Sender:  senderJID,
				Chat:    chatJID,
				IsGroup: true,
			},
			Message: &waE2E.Message{
				Conversation: proto.String(text),
			},
		}
	}

	// 1. Pesan non-command di grup tidak memicu rate limiter
	nonCmd := makeMsg("MSG-1", "halo semuanya")
	HandleIncomingMessage(nil, nonCmd, nil, nil, nil, nil, nil, nil)
	if testLimiter.Remaining(senderJID.User) != 0 {
		t.Errorf("Expected non-command message in group to NOT trigger rate limiter")
	}

	// 2. Pesan command pertama di grup -> lolos rate limiter (masuk cooldown)
	cmd1 := makeMsg("MSG-2", "!jadwal")
	HandleIncomingMessage(nil, cmd1, nil, nil, nil, nil, nil, nil)
	if rem := testLimiter.Remaining(senderJID.User); rem <= 0 {
		t.Errorf("Expected command message to register cooldown, got %v", rem)
	}

	// 3. Pesan command kedua berturut-turut langsung ditolak oleh rate limiter
	cmd2 := makeMsg("MSG-3", "!tugas")
	HandleIncomingMessage(nil, cmd2, nil, nil, nil, nil, nil, nil)
	// Karena ditolak rate limiter, cooldown tetap aktif
	if testLimiter.Allow(senderJID.User) {
		t.Errorf("Expected immediate subsequent command to be blocked by rate limiter")
	}

	// 4. Setelah cooldown berlalu (reset untuk simulasi), command berikutnya diizinkan
	testLimiter.Reset(senderJID.User)
	if !testLimiter.Allow(senderJID.User) {
		t.Errorf("Expected command to be allowed after cooldown reset")
	}
}
