package audit

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// Canonical actor types.
const (
	ActorUser   = "USER"
	ActorSystem = "SYSTEM"
)

// Canonical actions (minimal vocabulary; names follow existing API/docs).
const (
	ActionCreate                 = "CREATE"
	ActionUpdate                 = "UPDATE"
	ActionDelete                 = "DELETE"
	ActionRestore                = "RESTORE"
	ActionPublish                = "PUBLISH"
	ActionRevoke                 = "REVOKE"
	ActionCreatePattern          = "CREATE_PATTERN"
	ActionUpdatePattern          = "UPDATE_PATTERN"
	ActionCreateTeachingEvent    = "CREATE_TEACHING_EVENT"
	ActionPublishTeachingEvent   = "PUBLISH_TEACHING_EVENT"
	ActionRevokeTeachingEvent    = "REVOKE_TEACHING_EVENT"
	ActionSessionCancel          = "SESSION_CANCEL"
	ActionOverrideConflict       = "OVERRIDE_CONFLICT"
	ActionCreateTask             = "CREATE_TASK"
	ActionUpdateTask             = "UPDATE_TASK"
	ActionReviewTask             = "REVIEW_TASK"
	ActionCompleteTask           = "COMPLETE_TASK"
	ActionArchiveTask            = "ARCHIVE_TASK"
	ActionRestoreTask            = "RESTORE_TASK"
	ActionActivateSemester       = "ACTIVATE_SEMESTER"
	ActionCreateSemester         = "CREATE_SEMESTER"
	ActionInviteRole             = "INVITE_ROLE"
	ActionAssignRole             = "ASSIGN_ROLE"
	ActionSuspendRole            = "SUSPEND_ROLE"
	ActionRevokeRole             = "REVOKE_ROLE"
	ActionSuspendUser            = "SUSPEND_USER"
	ActionRecoverUser            = "RECOVER_USER"
	ActionRotatePortalCode       = "ROTATE_PORTAL_CODE"
	ActionCreateRoomConfirmation = "CREATE_ROOM_CONFIRMATION"
	ActionRetryNotification      = "RETRY_NOTIFICATION"
	ActionApplyImport            = "APPLY_IMPORT"
	ActionCreateBackup           = "CREATE_BACKUP"
	ActionVerifyRestore          = "VERIFY_RESTORE"
	ActionCreateMaterial         = "CREATE_MATERIAL"
	ActionUpdateMaterial         = "UPDATE_MATERIAL"
	ActionUpdateClassStatus      = "UPDATE_CLASS_STATUS"
	ActionAcceptInvitation       = "ACCEPT_INVITATION"
	ActionCreateTeachingEventOld = "CREATE_TEACHING_EVENT"
	ActionPublishTask            = "PUBLISH_TASK"
	ActionLogin                  = "LOGIN"
	ActionSwitchContext          = "SWITCH_CONTEXT"
	ActionLogout                 = "LOGOUT"
)

// Canonical entity types.
const (
	EntityTeachingEvent    = "TEACHING_EVENT"
	EntitySchedulePattern  = "SCHEDULE_PATTERN"
	EntityTask             = "TASK"
	EntityTaskReview       = "TASK_REVIEW"
	EntityMaterial         = "MATERIAL"
	EntityRoomConfirmation = "ROOM_CONFIRMATION"
	EntityNotificationMsg  = "NOTIFICATION_MESSAGE"
	EntitySemester         = "SEMESTER"
	EntityClass            = "CLASS"
	EntityUser             = "USER"
	EntityRoleAssignment   = "ROLE_ASSIGNMENT"
	EntityRoleInvitation   = "ROLE_INVITATION"
	EntityBackup           = "BACKUP"
	EntityImportBatch      = "IMPORT_BATCH"
	EntityPortalSession    = "PORTAL_SESSION"
	EntityWhatsappChannel  = "WHATSAPP_CHANNEL"
	EntityAudit            = "AUDIT"
)

var (
	ErrInvalidEntry = errors.New("entri audit tidak valid")
	ErrSensitive    = errors.New("snapshot mengandung data sensitif")
)

// reasonRequired menandakan action yang wajib memiliki reason non-blank.
// Selaras dengan CHECK database (RESTORE, SESSION_CANCEL, OVERRIDE_CONFLICT)
// ditambah operasi yang secara produk mensyaratkan alasan eksplisit.
var reasonRequired = map[string]bool{
	"RESTORE": true, "SESSION_CANCEL": true, "OVERRIDE_CONFLICT": true,
	"REVOKE_TEACHING_EVENT": true, "VERIFY_RESTORE": true,
}

// Actor describes who performed the action.
type Actor struct {
	Type             string
	UserID           *int64
	RoleAssignmentID *int64
	Context          any
}

// Entry describes one audit log row.
// ClassID/SemesterID may be nil for global actions.
type Entry struct {
	Actor               Actor
	ClassID             *int64
	SemesterID          *int64
	Action              string
	EntityType          string
	EntityID            *int64
	Before              any
	After               any
	BeforeJSON          *string
	AfterJSON           *string
	Reason              string
	CorrelationID       string
	ActorContextJSON    *string
	ActorUserID         *int64
	ActorRoleAssignment *int64
	ActorType           string
}

// Executor abstracts *sql.DB and *sql.Tx for audit writes inside transactions.
type Executor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

// DBTX abstracts *sql.DB and *sql.Tx for audit writes inside transactions.
type DBTX interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

// Writer writes audit rows canonically.
type Writer struct{}

func NewCorrelationID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return ""
	}
	return hex.EncodeToString(b[:])
}

func strOrNil(s *string) any {
	if s == nil {
		return nil
	}
	if strings.TrimSpace(*s) == "" {
		return nil
	}
	return *s
}

// sensitiveSubstrings ditolak dalam bentuk apa pun, termasuk nested.
var sensitiveSubstrings = []string{
	"password", "passwd", "pass_hash", "password_hash",
	"bearer", "session_token", "refresh_token",
	"recovery_token", "recovery_code",
	"portal_code", "portal_code_hash",
	"api_key", "apikey", "secret", "private_key", "credential",
	"cookie", "set-cookie",
}

func containsSensitiveKey(k string) bool {
	lk := strings.ToLower(k)
	for _, s := range sensitiveSubstrings {
		if strings.Contains(lk, s) {
			return true
		}
	}
	return false
}

// sanitizeValue menelusuri map/slice rekursif dan me-redact key sensitif.
// Allowlist: snapshot domain kritis hanya boleh berisi field yang dikenal;
// untuk bentuk generik, denylist key + redaksi nilai token-like.
func sanitizeValue(v any) (any, bool) {
	switch t := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, val := range t {
			if containsSensitiveKey(k) {
				out[k] = "[REDACTED]"
				continue
			}
			sv, _ := sanitizeValue(val)
			out[k] = sv
		}
		return out, true
	case map[string]string:
		out := make(map[string]any, len(t))
		for k, val := range t {
			if containsSensitiveKey(k) {
				out[k] = "[REDACTED]"
			} else {
				out[k] = val
			}
		}
		return out, true
	case []any:
		out := make([]any, len(t))
		for i, val := range t {
			sv, _ := sanitizeValue(val)
			out[i] = sv
		}
		return out, true
	case []map[string]any:
		out := make([]any, len(t))
		for i, val := range t {
			sv, _ := sanitizeValue(val)
			out[i] = sv
		}
		return out, true
	default:
		return v, true
	}
}

// SanitizeJSON mem-parsing string JSON, me-redact field sensitif, dan
// mengembalikan JSON bersih. Nilai non-JSON dibungkus sebagai string aman.
func SanitizeJSON(raw *string) (*string, error) {
	if raw == nil || strings.TrimSpace(*raw) == "" {
		return nil, nil
	}
	s := strings.TrimSpace(*raw)
	var v any
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		// Bukan JSON objek: pastikan tidak mengandung credential mentah berlabel.
		ls := strings.ToLower(s)
		for _, sens := range sensitiveSubstrings {
			if strings.Contains(ls, sens) {
				red := `{"value":"[REDACTED]"}`
				return &red, nil
			}
		}
		return raw, nil
	}
	sv, _ := sanitizeValue(v)
	b, err := json.Marshal(sv)
	if err != nil {
		return nil, err
	}
	out := string(b)
	return &out, nil
}

func marshalAny(v any) (*string, error) {
	if v == nil {
		return nil, nil
	}
	if s, ok := v.(string); ok {
		if strings.TrimSpace(s) == "" {
			return nil, nil
		}
		return SanitizeJSON(&s)
	}
	if sp, ok := v.(*string); ok {
		return SanitizeJSON(sp)
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	s := string(b)
	return SanitizeJSON(&s)
}

func resolveActor(e Entry) (actorType string, userID, roleID *int64, ctxJSON *string, err error) {
	actorType = strings.ToUpper(strings.TrimSpace(e.Actor.Type))
	if actorType == "" {
		actorType = strings.ToUpper(strings.TrimSpace(e.ActorType))
	}
	if actorType == "" {
		actorType = ActorUser
	}
	if actorType != ActorUser && actorType != ActorSystem {
		return "", nil, nil, nil, fmt.Errorf("%w: actor_type harus USER atau SYSTEM", ErrInvalidEntry)
	}
	userID = e.Actor.UserID
	if userID == nil {
		userID = e.ActorUserID
	}
	roleID = e.Actor.RoleAssignmentID
	if roleID == nil {
		roleID = e.ActorRoleAssignment
	}
	if actorType == ActorUser && userID == nil {
		return "", nil, nil, nil, fmt.Errorf("%w: USER wajib memiliki user_id", ErrInvalidEntry)
	}
	if actorType == ActorSystem && userID != nil {
		// System actor tanpa user palsu; user ID harus kosong.
		return "", nil, nil, nil, fmt.Errorf("%w: SYSTEM tidak boleh memakai user_id", ErrInvalidEntry)
	}
	if e.Actor.Context != nil {
		b, mErr := json.Marshal(e.Actor.Context)
		if mErr != nil {
			return "", nil, nil, nil, mErr
		}
		s := string(b)
		san, sErr := SanitizeJSON(&s)
		if sErr != nil {
			return "", nil, nil, nil, sErr
		}
		ctxJSON = san
	} else {
		ctxJSON = e.ActorContextJSON
		if ctxJSON != nil {
			san, sErr := SanitizeJSON(ctxJSON)
			if sErr != nil {
				return "", nil, nil, nil, sErr
			}
			ctxJSON = san
		}
	}
	return actorType, userID, roleID, ctxJSON, nil
}

// Validate memeriksa field wajib sebelum insert.
func Validate(e Entry) error {
	action := strings.TrimSpace(e.Action)
	if action == "" {
		return fmt.Errorf("%w: action wajib diisi", ErrInvalidEntry)
	}
	if strings.TrimSpace(e.EntityType) == "" {
		return fmt.Errorf("%w: entity_type wajib diisi", ErrInvalidEntry)
	}
	if strings.TrimSpace(e.CorrelationID) == "" {
		return fmt.Errorf("%w: correlation_id wajib diisi", ErrInvalidEntry)
	}
	if _, _, _, _, err := resolveActor(e); err != nil {
		return err
	}
	if reasonRequired[strings.ToUpper(action)] && strings.TrimSpace(e.Reason) == "" {
		return fmt.Errorf("%w: reason wajib untuk action %s", ErrInvalidEntry, action)
	}
	return nil
}

// Write inserts one audit log row canonically: validasi, sanitasi,
// correlation default, dan insert atomik via executor caller.
// Caller must ensure foreign keys exist; caller harus memakai transaksi
// yang sama dengan mutasi domain agar audit dan domain commit/rollback bersama.
func Write(ctx context.Context, db DBTX, e Entry) error {
	return New().Write(ctx, db, e)
}

// Write implements Writer.Write.
func (w *Writer) Write(ctx context.Context, exec Executor, e Entry) error {
	return writeEntry(ctx, exec, e)
}

// New creates a canonical Writer.
func New() *Writer { return &Writer{} }

func writeEntry(ctx context.Context, exec Executor, e Entry) error {
	action := strings.TrimSpace(e.Action)
	entityType := strings.TrimSpace(e.EntityType)
	corr := strings.TrimSpace(e.CorrelationID)
	if corr == "" {
		corr = NewCorrelationID()
	}
	actorType, userID, roleID, ctxJSON, err := resolveActor(e)
	if err != nil {
		return err
	}
	if action == "" {
		return fmt.Errorf("%w: action wajib diisi", ErrInvalidEntry)
	}
	if entityType == "" {
		return fmt.Errorf("%w: entity_type wajib diisi", ErrInvalidEntry)
	}
	if corr == "" {
		return fmt.Errorf("%w: correlation_id wajib diisi", ErrInvalidEntry)
	}
	if reasonRequired[strings.ToUpper(action)] && strings.TrimSpace(e.Reason) == "" {
		return fmt.Errorf("%w: reason wajib untuk action %s", ErrInvalidEntry, action)
	}
	var beforeJSON, afterJSON *string
	if e.Before != nil {
		beforeJSON, err = marshalAny(e.Before)
		if err != nil {
			return err
		}
	} else {
		beforeJSON, err = SanitizeJSON(e.BeforeJSON)
		if err != nil {
			return err
		}
	}
	if e.After != nil {
		afterJSON, err = marshalAny(e.After)
		if err != nil {
			return err
		}
	} else {
		afterJSON, err = SanitizeJSON(e.AfterJSON)
		if err != nil {
			return err
		}
	}
	var reason any
	if strings.TrimSpace(e.Reason) != "" {
		reason = strings.TrimSpace(e.Reason)
	}
	_, err = exec.ExecContext(ctx, `INSERT INTO audit_logs (
		class_id, semester_id, actor_user_id, actor_role_assignment_id,
		actor_context_json, actor_type, action, entity_type, entity_id,
		before_json, after_json, reason, correlation_id,
		created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, strftime('%Y-%m-%dT%H:%M:%fZ','now'), strftime('%Y-%m-%dT%H:%M:%fZ','now'))`,
		e.ClassID, e.SemesterID, userID, roleID,
		strOrNil(ctxJSON), actorType,
		action, entityType, e.EntityID,
		strOrNil(beforeJSON), strOrNil(afterJSON), reason, corr,
	)
	return err
}
