package auth

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	"bot-jadwal/internal/database"
	"golang.org/x/crypto/bcrypt"
)

type testClock struct {
	now time.Time
}

func (c *testClock) Now() time.Time { return c.now }
func (c *testClock) Add(d time.Duration) {
	c.now = c.now.Add(d)
}

func newTestService(t *testing.T) (*Service, *sql.DB, *testClock) {
	t.Helper()
	db, err := database.InitDB(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	clock := &testClock{now: time.Date(2026, 9, 25, 2, 0, 0, 0, time.UTC)}
	service, err := NewService(db, Config{
		HashKey: []byte("0123456789abcdef0123456789abcdef"),
		Clock:   clock.Now,
	})
	if err != nil {
		t.Fatal(err)
	}
	return service, db, clock
}

func provisionAdmin(t *testing.T, service *Service) (*User, *RoleAssignment) {
	t.Helper()
	user, assignment, err := service.ProvisionInitialSystemAdmin(context.Background(), ProvisionInput{
		IdentityKey: " Admin@Example.test ",
		DisplayName: "Admin Awal",
		Password:    "kata-sandi-yang-kuat",
	})
	if err != nil {
		t.Fatal(err)
	}
	return user, assignment
}

func TestPasswordPolicyAndBcryptCost(t *testing.T) {
	if _, err := HashPassword("admin", "terlalu"); err == nil {
		t.Fatal("password kurang dari 12 karakter harus ditolak")
	}
	if _, err := HashPassword("identitas-sama", "IDENTITAS-SAMA"); err == nil {
		t.Fatal("password yang sama dengan identitas harus ditolak")
	}
	hash, err := HashPassword("admin", "kata-sandi-yang-kuat")
	if err != nil {
		t.Fatal(err)
	}
	cost, err := bcrypt.Cost([]byte(hash))
	if err != nil {
		t.Fatal(err)
	}
	if cost != PasswordHashCost || cost < 12 {
		t.Fatalf("bcrypt cost = %d, ingin %d", cost, PasswordHashCost)
	}
	if !VerifyPassword(hash, "kata-sandi-yang-kuat") || VerifyPassword(hash, "salah") {
		t.Fatal("verifikasi password tidak sesuai")
	}
}

func TestProvisionInitialSystemAdminIsTransactionalAndOneTime(t *testing.T) {
	service, db, _ := newTestService(t)
	user, assignment := provisionAdmin(t, service)
	if user.IdentityKey != "admin@example.test" || assignment.Role != RoleSystemAdmin || assignment.ScopeType != ScopeGlobal {
		t.Fatalf("hasil provisioning tidak tepat: user=%+v assignment=%+v", user, assignment)
	}
	if _, _, err := service.ProvisionInitialSystemAdmin(context.Background(), ProvisionInput{
		IdentityKey: "other@example.test", DisplayName: "Other", Password: "password-yang-aman",
	}); !errors.Is(err, ErrAlreadyProvisioned) {
		t.Fatalf("provisioning kedua error = %v", err)
	}
	var users int
	if err := db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&users); err != nil {
		t.Fatal(err)
	}
	if users != 1 {
		t.Fatalf("jumlah users = %d, ingin 1", users)
	}
}

func TestAuthenticateUsesGenericFailureAndThrottlesPair(t *testing.T) {
	service, db, clock := newTestService(t)
	user, _ := provisionAdmin(t, service)
	ctx := context.Background()

	for i := 0; i < maxLoginFailures; i++ {
		_, err := service.Authenticate(ctx, LoginInput{IdentityKey: user.IdentityKey, Password: "salah", Source: "198.51.100.1"})
		if !errors.Is(err, ErrAuthenticationFailed) {
			t.Fatalf("kegagalan %d error = %v", i+1, err)
		}
	}
	_, err := service.Authenticate(ctx, LoginInput{IdentityKey: user.IdentityKey, Password: "kata-sandi-yang-kuat", Source: "198.51.100.1"})
	if !errors.Is(err, ErrAuthenticationFailed) {
		t.Fatalf("login terblokir error = %v", err)
	}

	var failures, blocked int
	if err := db.QueryRow(`SELECT
		SUM(CASE WHEN outcome = 'FAILURE' THEN 1 ELSE 0 END),
		SUM(CASE WHEN outcome = 'BLOCKED' THEN 1 ELSE 0 END)
		FROM login_attempts`).Scan(&failures, &blocked); err != nil {
		t.Fatal(err)
	}
	if failures != 5 || blocked != 1 {
		t.Fatalf("failure=%d blocked=%d", failures, blocked)
	}
	var rawStored int
	if err := db.QueryRow(`SELECT COUNT(*) FROM login_attempts WHERE identity_hash = ? OR source_hash = ?`, user.IdentityKey, "198.51.100.1").Scan(&rawStored); err != nil {
		t.Fatal(err)
	}
	if rawStored != 0 {
		t.Fatal("identitas atau sumber mentah tersimpan di login_attempts")
	}

	clock.Add(loginBlockPeriod + time.Second)
	result, err := service.Authenticate(ctx, LoginInput{IdentityKey: user.IdentityKey, Password: "kata-sandi-yang-kuat", Source: "198.51.100.1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Assignments) != 1 || result.Assignments[0].Role != RoleSystemAdmin {
		t.Fatalf("assignments = %+v", result.Assignments)
	}
}

func TestSessionValidationRotationAndRevocation(t *testing.T) {
	service, db, _ := newTestService(t)
	user, admin := provisionAdmin(t, service)
	classID := insertClass(t, db)
	kmID := insertAssignment(t, db, user.ID, RoleKM, ScopeClass, &classID, nil, nil)

	credential, err := service.CreateSession(context.Background(), user.ID, admin.ID)
	if err != nil {
		t.Fatal(err)
	}
	if credential.Token == "" {
		t.Fatal("token sesi kosong")
	}
	var storedToken string
	if err := db.QueryRow(`SELECT token_hash FROM user_sessions WHERE id = ?`, credential.Session.ID).Scan(&storedToken); err != nil {
		t.Fatal(err)
	}
	if storedToken == credential.Token || storedToken != tokenHash(credential.Token) {
		t.Fatal("database harus menyimpan hash token")
	}

	principal, err := service.ValidateSession(context.Background(), credential.Token)
	if err != nil || !principal.IsSystemAdmin() {
		t.Fatalf("validasi sesi admin: principal=%+v err=%v", principal, err)
	}

	rotated, err := service.SwitchContext(context.Background(), credential.Token, kmID)
	if err != nil {
		t.Fatal(err)
	}
	if rotated.Token == credential.Token {
		t.Fatal("switch context tidak merotasi token")
	}
	if _, err := service.ValidateSession(context.Background(), credential.Token); !errors.Is(err, ErrInvalidSession) {
		t.Fatalf("token lama masih valid: %v", err)
	}
	principal, err = service.ValidateSession(context.Background(), rotated.Token)
	if err != nil || !principal.CanManageClass(classID) {
		t.Fatalf("principal KM tidak tepat: %+v err=%v", principal, err)
	}

	if err := service.RevokeSession(context.Background(), rotated.Token, "logout"); err != nil {
		t.Fatal(err)
	}
	if _, err := service.ValidateSession(context.Background(), rotated.Token); !errors.Is(err, ErrInvalidSession) {
		t.Fatalf("sesi tercabut masih valid: %v", err)
	}
}

func TestSessionExpiryAndSessionVersion(t *testing.T) {
	t.Run("idle expiry follows active role", func(t *testing.T) {
		service, _, clock := newTestService(t)
		user, assignment := provisionAdmin(t, service)
		credential, err := service.CreateSession(context.Background(), user.ID, assignment.ID)
		if err != nil {
			t.Fatal(err)
		}
		clock.Add(adminIdle)
		if _, err := service.ValidateSession(context.Background(), credential.Token); !errors.Is(err, ErrInvalidSession) {
			t.Fatalf("sesi idle error = %v", err)
		}
	})

	t.Run("session version revokes all sessions", func(t *testing.T) {
		service, db, _ := newTestService(t)
		user, assignment := provisionAdmin(t, service)
		credential, err := service.CreateSession(context.Background(), user.ID, assignment.ID)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(`UPDATE users SET session_version = session_version + 1 WHERE id = ?`, user.ID); err != nil {
			t.Fatal(err)
		}
		if _, err := service.ValidateSession(context.Background(), credential.Token); !errors.Is(err, ErrInvalidSession) {
			t.Fatalf("sesi lama error = %v", err)
		}
	})
}

func TestRoleRevocationAndScopeAuthorization(t *testing.T) {
	service, db, _ := newTestService(t)
	user, _ := provisionAdmin(t, service)
	classA := insertClass(t, db)
	classB := insertClassWithCode(t, db, "B")
	semesterID, offeringID := insertOffering(t, db, classA)
	pjID := insertAssignment(t, db, user.ID, RolePJ, ScopeCourseOffering, &classA, &semesterID, &offeringID)

	credential, err := service.CreateSession(context.Background(), user.ID, pjID)
	if err != nil {
		t.Fatal(err)
	}
	principal, err := service.ValidateSession(context.Background(), credential.Token)
	if err != nil {
		t.Fatal(err)
	}
	if err := principal.RequireOffering(Scope{ClassID: classA, SemesterID: semesterID, CourseOfferingID: offeringID}); err != nil {
		t.Fatal(err)
	}
	if err := service.RequireOfferingMutation(context.Background(), *principal, Scope{ClassID: classA, SemesterID: semesterID, CourseOfferingID: offeringID}); err != nil {
		t.Fatal(err)
	}
	if principal.CanManageOffering(Scope{ClassID: classB, SemesterID: semesterID, CourseOfferingID: offeringID}) {
		t.Fatal("PJ memperoleh akses di luar kelas assignment")
	}
	if principal.CanReviewClass(classA) || principal.CanRevokePublishedTeachingEvent(classA) {
		t.Fatal("PJ tidak boleh mereview kelas atau mencabut teaching event terbit")
	}
	if _, err := db.Exec(`UPDATE semesters SET status = 'ARCHIVED', archived_at = ? WHERE id = ?`, formatTime(time.Now()), semesterID); err != nil {
		t.Fatal(err)
	}
	if err := service.RequireOfferingMutation(context.Background(), *principal, Scope{ClassID: classA, SemesterID: semesterID, CourseOfferingID: offeringID}); !errors.Is(err, ErrAccessDenied) {
		t.Fatalf("mutasi semester arsip error = %v", err)
	}

	if _, err := db.Exec(`UPDATE role_assignments SET status = 'REVOKED' WHERE id = ?`, pjID); err != nil {
		t.Fatal(err)
	}
	if _, err := service.ValidateSession(context.Background(), credential.Token); !errors.Is(err, ErrInvalidSession) {
		t.Fatalf("assignment tercabut masih berlaku: %v", err)
	}
}

func TestContextCancellationIsPropagated(t *testing.T) {
	service, _, _ := newTestService(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := service.Authenticate(ctx, LoginInput{IdentityKey: "x", Password: "x", Source: "x"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, ingin context.Canceled", err)
	}
}

func insertClass(t *testing.T, db *sql.DB) int64 {
	t.Helper()
	return insertClassWithCode(t, db, "A")
}

func insertClassWithCode(t *testing.T, db *sql.DB, code string) int64 {
	t.Helper()
	var id int64
	err := db.QueryRow(`INSERT INTO classes (code, slug, study_program, cohort_year, group_label)
		VALUES (?, ?, 'TI', 2026, ?) RETURNING id`, "CLASS-"+code, "class-"+strings.ToLower(code), code).Scan(&id)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func insertOffering(t *testing.T, db *sql.DB, classID int64) (int64, int64) {
	t.Helper()
	var semesterID, courseID, offeringID int64
	if err := db.QueryRow(`INSERT INTO semesters (class_id, academic_year, term, starts_on, ends_on)
		VALUES (?, '2026/2027', 'ODD', '2026-08-01', '2026-12-31') RETURNING id`, classID).Scan(&semesterID); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`INSERT INTO courses (code, name) VALUES (?, 'Mata Kuliah') RETURNING id`, "MK-"+time.Now().Format("150405.000000")).Scan(&courseID); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`INSERT INTO course_offerings (semester_id, course_id, activity_type, display_name)
		VALUES (?, ?, 'LECTURE', 'Mata Kuliah') RETURNING id`, semesterID, courseID).Scan(&offeringID); err != nil {
		t.Fatal(err)
	}
	return semesterID, offeringID
}

func insertAssignment(t *testing.T, db *sql.DB, userID int64, role, scope string, classID, semesterID, offeringID *int64) int64 {
	t.Helper()
	var id int64
	err := db.QueryRow(`INSERT INTO role_assignments (
		user_id, role, scope_type, class_id, semester_id, course_offering_id, status, valid_from
	) VALUES (?, ?, ?, ?, ?, ?, 'ACTIVE', '2026-01-01T00:00:00Z') RETURNING id`, userID, role, scope, classID, semesterID, offeringID).Scan(&id)
	if err != nil {
		t.Fatal(err)
	}
	return id
}
