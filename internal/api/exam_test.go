package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestExamAPI_GranularLifecycleAndScope(t *testing.T) {
	for _, kind := range []string{"EXAM_UTS", "EXAM_UAS"} {
		t.Run(kind, func(t *testing.T) {
			db, s := setupV1TestEnv(t)
			_, err := db.Exec(`UPDATE semesters SET starts_on='2026-09-01',ends_on='2027-01-31';
   UPDATE schedule_patterns SET effective_from='2026-09-01';
   INSERT INTO course_offerings(id,semester_id,course_id,display_name,activity_type) VALUES(2,1,1,'Praktikum','PRACTICE');
   INSERT INTO schedule_patterns(id,course_offering_id,day_of_week,start_time,end_time,effective_from) VALUES(2,2,1,'11:00','12:00','2026-09-01');`)
			if err != nil {
				t.Fatal(err)
			}
			km := helperLogin(t, s, "+6281234567890", "password123")
			pj := helperLogin(t, s, "+6281298765432", "password123")
			payload := map[string]any{"owner_offering_id": 1, "event_kind": kind, "starts_at": "2026-11-09T01:00:00Z", "ends_at": "2026-11-09T02:40:00Z", "origin_pattern_id": 1, "origin_date": "2026-11-09", "reason": "Buka Buku, Pengawas: Dani", "room_id": 1}
			for _, bad := range []map[string]any{
				{"origin_date": nil}, {"origin_date": "2026-02-30"}, {"origin_pattern_id": 999}, {"origin_pattern_id": 2},
			} {
				copy := map[string]any{}
				for k, v := range payload {
					copy[k] = v
				}
				for k, v := range bad {
					copy[k] = v
				}
				w := helperDo(t, s, "POST", "/api/v1/teaching-events", km, copy)
				if w.Code != 422 {
					t.Fatalf("invalid: %d %s", w.Code, w.Body.String())
				}
			}
			payload["owner_offering_id"] = 2
			if w := helperDo(t, s, "POST", "/api/v1/teaching-events", pj, payload); w.Code != 403 {
				t.Fatalf("PJ scope: %d %s", w.Code, w.Body.String())
			}
			payload["owner_offering_id"] = 1
			w := helperDo(t, s, "POST", "/api/v1/teaching-events", pj, payload)
			if w.Code != 201 {
				t.Fatal(w.Code, w.Body.String())
			}
			var created struct {
				Data struct {
					ID     int64  `json:"id"`
					Kind   string `json:"event_kind"`
					Starts string `json:"starts_at"`
				}
			}
			if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
				t.Fatal(err)
			}
			if created.Data.Kind != kind || created.Data.Starts == "" {
				t.Fatal(w.Body.String())
			}
			id := created.Data.ID
			portal := func(date string) string {
				t.Helper()
				w := helperDo(t, s, "GET", "/api/v1/portal/d4-ti-2024-a/schedule?date="+date, "", nil)
				if w.Code != 200 {
					t.Fatal(w.Code, w.Body.String())
				}
				return w.Body.String()
			}
			if strings.Contains(portal("2026-11-09"), `"kind":"`+strings.TrimPrefix(kind, "EXAM_")+`"`) {
				t.Fatal("draft leaked")
			}
			publish := func(version int) *httptest.ResponseRecorder {
				req := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/teaching-events/%d/publish", id), bytes.NewBufferString(fmt.Sprintf(`{"version":%d}`, version)))
				req.Header.Set("Authorization", "Bearer "+pj)
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Idempotency-Key", fmt.Sprintf("exam-%d-%d", id, version))
				w := httptest.NewRecorder()
				s.httpServer.Handler.ServeHTTP(w, req)
				return w
			}
			if w := publish(1); w.Code != 422 {
				t.Fatalf("room gate: %d %s", w.Code, w.Body.String())
			}
			w = helperDo(t, s, "POST", fmt.Sprintf("/api/v1/teaching-events/%d/room-confirmations", id), pj, map[string]any{"room_id": 1, "confirmation_status": "CONFIRMED", "external_contact": "TU QA"})
			if w.Code != 201 {
				t.Fatal(w.Code, w.Body.String())
			}
			if w := publish(2); w.Code != 200 {
				t.Fatal(w.Code, w.Body.String())
			}
			text := portal("2026-11-09")
			if !strings.Contains(text, `"kind":"`+strings.TrimPrefix(kind, "EXAM_")+`"`) || !strings.Contains(text, "Pengawas: Dani") || !strings.Contains(text, "pat_2") || strings.Contains(text, `"id":"pat_1"`) {
				t.Fatal(text)
			}
			if !strings.Contains(portal("2026-11-16"), "pat_1") {
				t.Fatal("other week suppressed")
			}
			w = helperDo(t, s, "POST", fmt.Sprintf("/api/v1/teaching-events/%d/revoke", id), km, map[string]any{"version": 3, "reason": "Ujian dipindahkan"})
			if w.Code != 200 {
				t.Fatal(w.Code, w.Body.String())
			}
			text = portal("2026-11-09")
			if !strings.Contains(text, "pat_1") || strings.Contains(text, `"kind":"`+strings.TrimPrefix(kind, "EXAM_")+`"`) {
				t.Fatal(text)
			}
			payload["origin_pattern_id"] = nil
			payload["origin_date"] = nil
			payload["room_id"] = nil
			payload["starts_at"] = "2026-11-14T02:00:00Z"
			payload["ends_at"] = "2026-11-14T04:00:00Z"
			w = helperDo(t, s, "POST", "/api/v1/teaching-events", pj, payload)
			if w.Code != 201 {
				t.Fatal(w.Code, w.Body.String())
			}
			if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
				t.Fatal(err)
			}
			id = created.Data.ID
			if w := publish(1); w.Code != 200 {
				t.Fatal(w.Code, w.Body.String())
			}
			if !strings.Contains(portal("2026-11-14"), `"kind":"`+strings.TrimPrefix(kind, "EXAM_")+`"`) {
				t.Fatal("standalone missing")
			}
		})
	}
}
