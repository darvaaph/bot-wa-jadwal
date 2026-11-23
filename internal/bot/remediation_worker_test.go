package bot

import (
	"context"
	"strings"
	"testing"
	"time"

	"bot-jadwal/internal/notify"
)

func TestRemediation_WorkerRetainsConfiguredDailyBuilder(t *testing.T) {
	botService := setupTestV1DB(t)
	service := notify.NewService(botService.db)
	service.SetDailyTextBuilder(func(context.Context, int64, time.Time) (string, error) {
		return "Configured builder: Dosen Budi; ruang R1", nil
	})
	channel, err := service.EnsureChannel(context.Background(), 1, "fixture@g.us", "Test")
	if err != nil {
		t.Fatal(err)
	}
	loc, _ := time.LoadLocation("Asia/Jakarta")
	key := "daily:1:" + time.Now().In(loc).Format("2006-01-02")
	if _, _, err := service.Enqueue(context.Background(), 1, channel, "DAILY_SUMMARY", "CLASS", 1, key, "stale queued text", "", time.Now().Add(-time.Minute), nil); err != nil {
		t.Fatal(err)
	}
	sender := &mockWhatsAppSender{connected: true}
	worker := NewNotificationWorker(botService.db, sender)
	worker.SetScheduleService(service)
	n, err := worker.ProcessPending(context.Background())
	if err != nil || n != 1 {
		t.Fatal(n, err)
	}
	if !strings.Contains(sender.sentMessage.GetConversation(), "Configured builder: Dosen Budi") {
		t.Fatal(sender.sentMessage.GetConversation())
	}
}
