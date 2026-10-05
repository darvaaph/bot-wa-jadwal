package api

import (
	"net/http"
	"strings"
	"testing"
)

func TestRoomHistoryAndDelivery_ScopedToActiveRole(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()
	seedSecondClass(t, db)
	if _, err := db.Exec(`INSERT INTO room_confirmations(teaching_event_id,room_id,confirmation_status,recorded_by_user_id) VALUES(1,1,'PENDING',1);
		INSERT INTO notification_messages(class_id,whatsapp_channel_id,event_type,entity_type,entity_id,idempotency_key,payload_json,status,scheduled_at) VALUES(1,1,'TEACHING_EVENT_PUBLISHED','TEACHING_EVENT',1,'access-test','{}','PENDING',CURRENT_TIMESTAMP)`); err != nil {
		t.Fatal(err)
	}
	pj := helperLogin(t, s, "+6281298765432", "password123")
	room := helperDo(t, s, "GET", "/api/v1/rooms/confirmations", pj, nil)
	if room.Code != http.StatusOK || !strings.Contains(room.Body.String(), "teaching_event_id") {
		t.Fatalf("PJ own room history: %d %s", room.Code, room.Body.String())
	}
	delivery := helperDo(t, s, "GET", "/api/v1/publications/TEACHING_EVENT/1/delivery", pj, nil)
	if delivery.Code != http.StatusOK || !strings.Contains(delivery.Body.String(), "TEACHING_EVENT_PUBLISHED") || strings.Contains(delivery.Body.String(), "access-test") {
		t.Fatalf("PJ own delivery: %d %s", delivery.Code, delivery.Body.String())
	}
	pjOther := helperSwitchContext(t, s, pj, 5)
	room = helperDo(t, s, "GET", "/api/v1/rooms/confirmations", pjOther, nil)
	if room.Code != 200 || strings.Contains(room.Body.String(), "teaching_event_id") {
		t.Fatalf("PJ other room history leaked: %d %s", room.Code, room.Body.String())
	}
	delivery = helperDo(t, s, "GET", "/api/v1/publications/TEACHING_EVENT/1/delivery", pjOther, nil)
	if delivery.Code != 404 {
		t.Fatalf("PJ other delivery leaked: %d %s", delivery.Code, delivery.Body.String())
	}
}
