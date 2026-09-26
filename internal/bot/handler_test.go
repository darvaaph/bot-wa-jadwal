package bot

import (
	"strings"
	"testing"
	"time"

	"bot-jadwal/internal/link"
	"bot-jadwal/internal/schedule"
	"bot-jadwal/internal/task"

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

func TestMutationRedirectEntity(t *testing.T) {
	cases := []struct {
		msg     string
		isGroup bool
		entity  string
	}{
		{"!tugas tambah SBD | Lapres | Jumat 23:59", true, "tugas"},
		{"!tugas hapus 1", true, "tugas"},
		{"!tugas selesai 1", true, "tugas"},
		{"!tugas edit 1 | besok", true, "tugas"},
		{"!tugas ganti 1 | besok", true, "tugas"},
		{"tugas tambah SBD | x | y", false, "tugas"},
		{"/tugas hapus 2", true, "tugas"},
		{"!link tambah Drive | https://s.id/x", true, "tautan"},
		{"!link hapus 1", true, "tautan"},
		{"!link edit 1", true, "tautan"},
		{"!tautan tambah X | https://s.id/x", true, "tautan"},
		{"!pindah aljabar | besok 13:00", true, "jadwal kuliah"},
		{"!ganti sbd | besok", true, "jadwal kuliah"},
		{"!kosong sbd | besok", true, "jadwal kuliah"},
		{"!libur besok | Libur", true, "jadwal kuliah"},
		{"!kuliahganti matdis | sabtu 09:00", true, "jadwal kuliah"},
		{"!tambahkelas matdis | sabtu 09:00", true, "jadwal kuliah"},
		{"!batalganti 1", true, "jadwal kuliah"},
		{"!batal foo", true, "jadwal kuliah"},
		{"pindah aljabar | besok", false, "jadwal kuliah"},
	}
	for _, tc := range cases {
		entity, ok := MutationRedirectEntity(tc.msg, tc.isGroup)
		if !ok || entity != tc.entity {
			t.Errorf("MutationRedirectEntity(%q, group=%v) = (%q, %v), want (%q, true)", tc.msg, tc.isGroup, entity, ok, tc.entity)
		}
		if !strings.Contains(RedirectMessage(entity), "PENGELOLAAN DATA TERPUSAT") ||
			!strings.Contains(RedirectMessage(entity), entity) ||
			!strings.Contains(RedirectMessage(entity), "http://localhost:8080/app.html") {
			t.Errorf("RedirectMessage(%q) missing spec format", entity)
		}
	}

	reads := []struct {
		msg     string
		isGroup bool
	}{
		{"!tugas", true},
		{"!tugas hari ini", true},
		{"!tugas sbd", true},
		{"!tugas riwayat", true},
		{"!tugas bantuan", true},
		{"tugas", false},
		{"!jadwal", true},
		{"!jadwal besok", true},
		{"!link", true},
		{"!link bantuan", true},
		{"!link aljabar", true},
		{"!drive", true},
		{"!zoom", true},
		{"!jadwalganti", true},
		{"!portal", true},
		{"!web", true},
		{"!menu", true},
		{"halo semuanya", true},
		{"tugas tambah tanpa prefix di grup", true},
		{"!tugaskemarin", true},
	}
	for _, tc := range reads {
		if entity, ok := MutationRedirectEntity(tc.msg, tc.isGroup); ok {
			t.Errorf("MutationRedirectEntity(%q, group=%v) = (%q, true), want no redirect", tc.msg, tc.isGroup, entity)
		}
	}
}

func TestPortalMessage(t *testing.T) {
	msg := PortalMessage()
	if !strings.Contains(msg, "http://localhost:8080/app.html") || !strings.Contains(msg, "http://localhost:8080/") {
		t.Errorf("PortalMessage missing dashboard/portal URLs, got:\n%s", msg)
	}
}

func TestManagerMutationRedirects(t *testing.T) {
	now := time.Date(2026, 9, 9, 10, 0, 0, 0, time.Local)
	tm := &task.TaskManager{}
	for _, cmd := range []string{
		"!tugas tambah SBD | Lapres | Jumat 23:59",
		"!tugas selesai 1",
		"!tugas hapus 1",
		"!tugas edit 1 | besok",
	} {
		resp := tm.HandleCommand("120363001@g.us", true, "628120001@s.whatsapp.net", true, cmd, nil, now)
		if !strings.Contains(resp, "PENGELOLAAN DATA TERPUSAT") || !strings.Contains(resp, "tugas") {
			t.Errorf("task %q not redirected, got:\n%s", cmd, resp)
		}
	}

	om := &schedule.OverrideManager{}
	for _, cmd := range []string{
		"!pindah aljabar | besok 13:00",
		"!kosong sbd | besok",
		"!libur besok | Libur",
		"!kuliahganti matdis | sabtu 09:00",
		"!batalganti 1",
	} {
		resp := om.HandleCommand("120363001@g.us", true, "628120001@s.whatsapp.net", true, cmd, nil, now)
		if !strings.Contains(resp, "PENGELOLAAN DATA TERPUSAT") || !strings.Contains(resp, "jadwal kuliah") {
			t.Errorf("override %q not redirected, got:\n%s", cmd, resp)
		}
	}

	lm := &link.LinkManager{}
	for _, cmd := range []string{
		"!link tambah Drive | https://s.id/x",
		"!link hapus 1",
	} {
		resp := lm.HandleCommand("120363001@g.us", true, "628120001@s.whatsapp.net", true, cmd)
		if !strings.Contains(resp, "PENGELOLAAN DATA TERPUSAT") || !strings.Contains(resp, "tautan") {
			t.Errorf("link %q not redirected, got:\n%s", cmd, resp)
		}
	}
}
