package ratelimit

import (
	"context"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"bot-jadwal/internal/database"
)

func testKey() []byte { return []byte("0123456789abcdef0123456789abcdef") }

func newTestService(t *testing.T, policies map[PolicyKey]Policy) (*Service, context.Context) {
	t.Helper()
	db, err := database.InitDB(filepath.Join(t.TempDir(), "ratelimit.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := database.MigrateV1(db); err != nil {
		t.Fatal(err)
	}
	svc, err := NewService(db, testKey(), policies)
	if err != nil {
		t.Fatal(err)
	}
	return svc, context.Background()
}

func TestRatelimit_ThresholdAndBlock(t *testing.T) {
	pol := map[PolicyKey]Policy{PolicyAuthLogin: {MaxFailures: 3, Window: time.Minute, BlockPeriod: time.Minute}}
	svc, ctx := newTestService(t, pol)
	for i := 0; i < 3; i++ {
		res, err := svc.Check(ctx, PolicyAuthLogin, "user", "1.2.3.4")
		if err != nil || !res.Allowed {
			t.Fatalf("attempt %d harus lolos: %+v err=%v", i, res, err)
		}
		if err := svc.Record(ctx, PolicyAuthLogin, "user", "1.2.3.4", "FAILURE"); err != nil {
			t.Fatal(err)
		}
	}
	res, err := svc.Check(ctx, PolicyAuthLogin, "user", "1.2.3.4")
	if err != nil || res.Allowed {
		t.Fatalf("threshold harus diblokir: %+v err=%v", res, err)
	}
	if res.RetryAfter <= 0 {
		t.Fatalf("RetryAfter wajib positif: %v", res.RetryAfter)
	}
}

func TestRatelimit_PortsAndIPv6Normalized(t *testing.T) {
	if PeerIP("1.2.3.4:52341") != PeerIP("1.2.3.4:1234") {
		t.Fatal("port berbeda harus memberi source sama")
	}
	if PeerIP("::ffff:192.0.2.1") != "192.0.2.1" {
		t.Fatalf("IPv4-mapped IPv6 harus dikanonisasi: %s", PeerIP("::ffff:192.0.2.1"))
	}
	a := PeerIP("2001:0DB8:0000:0000:0000:0000:0000:0001")
	b := PeerIP("2001:db8::1")
	if a != b {
		t.Fatalf("representasi IPv6 setara harus sama: %s vs %s", a, b)
	}
}

func TestRatelimit_SpoofedHeaderIgnored(t *testing.T) {
	got := ClientSource("9.9.9.9:1234", map[string]string{"X-Forwarded-For": "1.2.3.4"}, nil)
	if got != "9.9.9.9" {
		t.Fatalf("header dari peer tak tepercaya harus diabaikan: %s", got)
	}
	got = ClientSource("10.0.0.1:1234", map[string]string{"X-Forwarded-For": "1.2.3.4, 10.0.0.1"}, []string{"10.0.0.0/8"})
	if got != "1.2.3.4" {
		t.Fatalf("trusted proxy harus menurunkan client: %s", got)
	}
	got = ClientSource("10.0.0.1:1234", map[string]string{"Forwarded": `for=1.2.3.4;proto=https, for=10.0.0.1`}, []string{"10.0.0.0/8"})
	if got != "1.2.3.4" {
		t.Fatalf("header Forwarded harus diproses: %s", got)
	}
}

func TestRatelimit_SharedAcrossInstances(t *testing.T) {
	db, err := database.InitDB(filepath.Join(t.TempDir(), "shared.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.MigrateV1(db); err != nil {
		t.Fatal(err)
	}
	pol := map[PolicyKey]Policy{PolicyPortalCode: {MaxFailures: 2, Window: time.Minute, BlockPeriod: time.Minute}}
	a, _ := NewService(db, testKey(), pol)
	b, _ := NewService(db, testKey(), pol)
	ctx := context.Background()
	for i := 0; i < 2; i++ {
		if _, err := a.Check(ctx, PolicyPortalCode, "class:1", "5.5.5.5"); err != nil {
			t.Fatal(err)
		}
		if err := b.Record(ctx, PolicyPortalCode, "class:1", "5.5.5.5", "FAILURE"); err != nil {
			t.Fatal(err)
		}
	}
	res, err := a.Check(ctx, PolicyPortalCode, "class:1", "5.5.5.5")
	if err != nil || res.Allowed {
		t.Fatalf("limit lintas instance harus bersama: %+v err=%v", res, err)
	}
}

func TestRatelimit_ConcurrentThreshold(t *testing.T) {
	pol := map[PolicyKey]Policy{PolicyInviteAccept: {MaxFailures: 5, Window: time.Minute, BlockPeriod: time.Minute}}
	svc, ctx := newTestService(t, pol)
	// Isi 5 kegagalan, lalu 2 check bersamaan: keduanya harus diblokir (tepat).
	for i := 0; i < 5; i++ {
		if err := svc.Record(ctx, PolicyInviteAccept, "tok", "9.9.9.9", "FAILURE"); err != nil {
			t.Fatal(err)
		}
	}
	var wg sync.WaitGroup
	results := make([]bool, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			res, err := svc.Check(ctx, PolicyInviteAccept, "tok", "9.9.9.9")
			if err == nil {
				results[idx] = res.Allowed
			}
		}(i)
	}
	wg.Wait()
	if results[0] || results[1] {
		t.Fatalf("keduanya harus diblokir: %v", results)
	}
}

func TestRatelimit_NoRawStorage(t *testing.T) {
	svc, ctx := newTestService(t, nil)
	if err := svc.Record(ctx, PolicyAuthLogin, "secret-identity", "8.8.8.8", "FAILURE"); err != nil {
		t.Fatal(err)
	}
	var n int
	_ = svc.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM security_attempts WHERE subject_hash LIKE '%secret%' OR source_hash LIKE '%8.8.8.8%'`).Scan(&n)
	if n != 0 {
		t.Fatalf("nilai mentah tidak boleh tersimpan: %d", n)
	}
}

func TestRatelimit_Purge(t *testing.T) {
	svc, ctx := newTestService(t, nil)
	if err := svc.Record(ctx, PolicyAuthLogin, "u", "1.1.1.1", "FAILURE"); err != nil {
		t.Fatal(err)
	}
	n, err := svc.Purge(ctx, -time.Hour)
	if err != nil || n != 1 {
		t.Fatalf("purge diharapkan 1, got %d err=%v", n, err)
	}
}

func TestRatelimit_UnknownPolicy(t *testing.T) {
	svc, ctx := newTestService(t, nil)
	if _, err := svc.Check(ctx, "NOPE", "u", "s"); err == nil {
		t.Fatal("policy tak dikenal harus error")
	}
}
