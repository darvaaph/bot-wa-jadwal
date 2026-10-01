package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"testing"
)

func TestV1Channels_LinkListRevoke(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()
	seedSecondClass(t, db)
	adminToken := helperLogin(t, s, "+6281111111111", "password123")

	// 1. Tanpa token 401; PJ 403
	w := helperDo(t, s, "GET", "/api/v1/whatsapp-channels", "", nil)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("tanpa token expected 401, got %d", w.Code)
	}
	pjToken := helperLogin(t, s, "+6281298765432", "password123")
	w = helperDo(t, s, "GET", "/api/v1/whatsapp-channels", pjToken, nil)
	if w.Code != http.StatusForbidden {
		t.Fatalf("PJ expected 403, got %d", w.Code)
	}

	// 2. Tautkan baru -> 201 + audit LINK_CHANNEL
	w = helperDo(t, s, "POST", "/api/v1/whatsapp-channels", adminToken, map[string]string{
		"jid": "120363000000000099@g.us", "class_slug": "d4-ti-2024-a", "display_name": "Grup A",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("link expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var nLink int
	_ = db.QueryRow(`SELECT COUNT(*) FROM audit_logs WHERE action = 'LINK_CHANNEL';`).Scan(&nLink)
	if nLink != 1 {
		t.Fatalf("audit LINK_CHANNEL expected 1, got %d", nLink)
	}

	// 3. JID sama kelas sama -> idempoten 200 changed:false
	w = helperDo(t, s, "POST", "/api/v1/whatsapp-channels", adminToken, map[string]string{
		"jid": "120363000000000099@g.us", "class_slug": "d4-ti-2024-a",
	})
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"changed":false`) {
		t.Fatalf("idempoten expected 200 changed:false, got %d: %s", w.Code, w.Body.String())
	}

	// 4. JID sama kelas lain -> 409 tolak-dulu
	w = helperDo(t, s, "POST", "/api/v1/whatsapp-channels", adminToken, map[string]string{
		"jid": "120363000000000099@g.us", "class_slug": "d4-ti-2024-b",
	})
	if w.Code != http.StatusConflict {
		t.Fatalf("tolak-dulu expected 409, got %d: %s", w.Code, w.Body.String())
	}
	var classID int64
	_ = db.QueryRow(`SELECT class_id FROM whatsapp_channels WHERE jid = '120363000000000099@g.us';`).Scan(&classID)
	if classID != 1 {
		t.Fatalf("tautan tak boleh pindah diam-diam, class=%d", classID)
	}

	// 5. Validasi: JID buruk, slug kosong, kelas tak ada -> 422/404
	for _, tc := range []struct {
		name string
		body map[string]string
		want int
	}{
		{"jid buruk", map[string]string{"jid": "bukanjid", "class_slug": "d4-ti-2024-a"}, http.StatusUnprocessableEntity},
		{"slug kosong", map[string]string{"jid": "1@g.us", "class_slug": ""}, http.StatusUnprocessableEntity},
		{"kelas tak ada", map[string]string{"jid": "1@g.us", "class_slug": "tak-ada"}, http.StatusNotFound},
	} {
		w = helperDo(t, s, "POST", "/api/v1/whatsapp-channels", adminToken, tc.body)
		if w.Code != tc.want {
			t.Fatalf("%s expected %d, got %d: %s", tc.name, tc.want, w.Code, w.Body.String())
		}
	}

	// 6. Daftar + filter (seed 1 + baru 1)
	w = helperDo(t, s, "GET", "/api/v1/whatsapp-channels", adminToken, nil)
	var all struct {
		Data []map[string]any `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &all)
	if len(all.Data) != 2 {
		t.Fatalf("daftar expected 2, got %v", all.Data)
	}
	w = helperDo(t, s, "GET", "/api/v1/whatsapp-channels?status=REVOKED", adminToken, nil)
	var f struct {
		Data []map[string]any `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &f)
	if len(f.Data) != 0 {
		t.Fatalf("filter REVOKED expected 0, got %d", len(f.Data))
	}

	// 7. Lepas: tanpa alasan 422; dengan alasan 200 + audit; ganda 409
	w = helperDo(t, s, "POST", "/api/v1/whatsapp-channels/2/revoke", adminToken, map[string]string{"reason": ""})
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("tanpa alasan expected 422, got %d", w.Code)
	}
	w = helperDo(t, s, "POST", "/api/v1/whatsapp-channels/2/revoke", adminToken, map[string]string{"reason": "Grup bubar"})
	if w.Code != http.StatusOK {
		t.Fatalf("revoke expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var st string
	_ = db.QueryRow(`SELECT status FROM whatsapp_channels WHERE id = 2;`).Scan(&st)
	if st != "REVOKED" {
		t.Fatalf("status expected REVOKED, got %s", st)
	}
	var nRevoke int
	_ = db.QueryRow(`SELECT COUNT(*) FROM audit_logs WHERE action = 'REVOKE_CHANNEL';`).Scan(&nRevoke)
	if nRevoke != 1 {
		t.Fatalf("audit REVOKE_CHANNEL expected 1, got %d", nRevoke)
	}
	w = helperDo(t, s, "POST", "/api/v1/whatsapp-channels/2/revoke", adminToken, map[string]string{"reason": "Lagi"})
	if w.Code != http.StatusConflict {
		t.Fatalf("revoke ganda expected 409, got %d", w.Code)
	}
	w = helperDo(t, s, "POST", "/api/v1/whatsapp-channels/999/revoke", adminToken, map[string]string{"reason": "x"})
	if w.Code != http.StatusNotFound {
		t.Fatalf("id tak ada expected 404, got %d", w.Code)
	}

	// 8. Taut ulang pasca-lepas ke kelas lain -> 201 changed:true (reuse diizinkan)
	w = helperDo(t, s, "POST", "/api/v1/whatsapp-channels", adminToken, map[string]string{
		"jid": "120363000000000099@g.us", "class_slug": "d4-ti-2024-b",
	})
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"changed":true`) {
		t.Fatalf("re-link expected 200 changed:true, got %d: %s", w.Code, w.Body.String())
	}
	var movedClass int64
	var movedStatus string
	_ = db.QueryRow(`SELECT class_id, status FROM whatsapp_channels WHERE jid = '120363000000000099@g.us';`).Scan(&movedClass, &movedStatus)
	if movedClass != 2 || movedStatus != "ACTIVE" {
		t.Fatalf("re-link salah: class=%d status=%s", movedClass, movedStatus)
	}
}

func TestV1Channels_KMScope(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()
	seedSecondClass(t, db)
	adminToken := helperLogin(t, s, "+6281111111111", "password123")
	helperDo(t, s, "POST", "/api/v1/whatsapp-channels", adminToken, map[string]string{
		"jid": "120363000000000002@g.us", "class_slug": "d4-ti-2024-b",
	})

	kmToken := helperLogin(t, s, "+6281234567890", "password123")
	kmToken = helperSwitchContext(t, s, kmToken, 1)

	// 1. KM daftar: hanya kelasnya (seed 1 milik kelas 1)
	w := helperDo(t, s, "GET", "/api/v1/whatsapp-channels", kmToken, nil)
	var k struct {
		Data []map[string]any `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &k)
	if len(k.Data) != 1 || k.Data[0]["class_slug"] != "d4-ti-2024-a" {
		t.Fatalf("KM expected 1 kanal kelasnya, got %v", k.Data)
	}

	// 2. KM tautkan kelasnya -> 201; kelas lain -> 404
	w = helperDo(t, s, "POST", "/api/v1/whatsapp-channels", kmToken, map[string]string{
		"jid": "120363000000000088@g.us", "class_slug": "d4-ti-2024-a",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("KM link sendiri expected 201, got %d: %s", w.Code, w.Body.String())
	}
	// KM tanpa slug -> otomatis kelasnya
	w = helperDo(t, s, "POST", "/api/v1/whatsapp-channels", kmToken, map[string]string{
		"jid": "120363000000000087@g.us",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("KM tanpa slug expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var autoClass int64
	_ = db.QueryRow(`SELECT class_id FROM whatsapp_channels WHERE jid = '120363000000000087@g.us';`).Scan(&autoClass)
	if autoClass != 1 {
		t.Fatalf("otomatis expected kelas 1, got %d", autoClass)
	}
	w = helperDo(t, s, "POST", "/api/v1/whatsapp-channels", kmToken, map[string]string{
		"jid": "120363000000000009@g.us", "class_slug": "d4-ti-2024-b",
	})
	if w.Code != http.StatusNotFound {
		t.Fatalf("KM link asing expected 404, got %d: %s", w.Code, w.Body.String())
	}

	// 3. KM lepas kanal kelas lain -> 404; kanal sendiri -> 200
	var foreignID int64
	_ = db.QueryRow(`SELECT id FROM whatsapp_channels WHERE jid = '120363000000000002@g.us';`).Scan(&foreignID)
	w = helperDo(t, s, "POST", "/api/v1/whatsapp-channels/"+strconv.FormatInt(foreignID, 10)+"/revoke", kmToken, map[string]string{"reason": "Iseng"})
	if w.Code != http.StatusNotFound {
		t.Fatalf("KM revoke asing expected 404, got %d: %s", w.Code, w.Body.String())
	}
	var ownID int64
	_ = db.QueryRow(`SELECT id FROM whatsapp_channels WHERE jid = '120363000000000088@g.us';`).Scan(&ownID)
	w = helperDo(t, s, "POST", "/api/v1/whatsapp-channels/"+strconv.FormatInt(ownID, 10)+"/revoke", kmToken, map[string]string{"reason": "Ganti grup"})
	if w.Code != http.StatusOK {
		t.Fatalf("KM revoke sendiri expected 200, got %d: %s", w.Code, w.Body.String())
	}
	_ = db
}
