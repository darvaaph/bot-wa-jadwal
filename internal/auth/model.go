package auth

import (
	"errors"
	"time"
)

const (
	RoleSystemAdmin = "SYSTEM_ADMIN"
	RoleKM          = "KM"
	RolePJ          = "PJ"

	ScopeGlobal         = "GLOBAL"
	ScopeClass          = "CLASS"
	ScopeCourseOffering = "COURSE_OFFERING"
)

var (
	ErrAuthenticationFailed = errors.New("identitas atau kata sandi tidak valid")
	ErrAccessDenied         = errors.New("tindakan tidak tersedia pada cakupan aktif")
	ErrInvalidSession       = errors.New("sesi tidak valid")
	ErrInvalidInput         = errors.New("input tidak valid")
	ErrAlreadyProvisioned   = errors.New("system admin awal sudah tersedia")
	ErrTooFrequent          = errors.New("permintaan terlalu sering, coba lagi nanti")
)

type User struct {
	ID             int64
	IdentityKey    string
	DisplayName    string
	Status         string
	SessionVersion int
	LastLoginAt    *time.Time
}

type RoleAssignment struct {
	ID               int64
	UserID           int64
	Role             string
	ScopeType        string
	ClassID          *int64
	SemesterID       *int64
	CourseOfferingID *int64
	Status           string
	ValidFrom        time.Time
	ValidUntil       *time.Time
	Version          int
}

type Principal struct {
	UserID                int64
	IdentityKey           string
	DisplayName           string
	RoleAssignmentID      int64
	RoleAssignmentVersion int
	Role                  string
	ScopeType             string
	ClassID               *int64
	SemesterID            *int64
	CourseOfferingID      *int64
	SessionID             int64
	SessionVersion        int
}

type Session struct {
	ID                     int64
	UserID                 int64
	ActiveRoleAssignmentID int64
	LastSeenAt             time.Time
	AbsoluteExpiresAt      time.Time
	CreatedAt              time.Time
	sessionVersion         int
}

type SessionCredential struct {
	Token   string
	Session Session
}

type AuthenticatedIdentity struct {
	User        User
	Assignments []RoleAssignment
}

type ProvisionInput struct {
	IdentityKey string
	DisplayName string
	Password    string
}

type LoginInput struct {
	IdentityKey string
	Password    string
	Source      string
}

type Scope struct {
	ClassID          int64
	SemesterID       int64
	CourseOfferingID int64
}

func (p Principal) IsSystemAdmin() bool {
	return p.Role == RoleSystemAdmin && p.ScopeType == ScopeGlobal
}

func (p Principal) CanManageClass(classID int64) bool {
	return p.Role == RoleKM && p.ScopeType == ScopeClass && p.ClassID != nil && *p.ClassID == classID
}

func (p Principal) CanManageOffering(scope Scope) bool {
	if p.CanManageClass(scope.ClassID) {
		return true
	}
	return p.Role == RolePJ && p.ScopeType == ScopeCourseOffering &&
		p.ClassID != nil && *p.ClassID == scope.ClassID &&
		p.SemesterID != nil && *p.SemesterID == scope.SemesterID &&
		p.CourseOfferingID != nil && *p.CourseOfferingID == scope.CourseOfferingID
}

func (p Principal) CanReviewClass(classID int64) bool {
	return p.CanManageClass(classID)
}

func (p Principal) CanRevokePublishedTeachingEvent(classID int64) bool {
	return p.CanManageClass(classID)
}

func (p Principal) RequireSystemAdmin() error {
	if !p.IsSystemAdmin() {
		return ErrAccessDenied
	}
	return nil
}

func (p Principal) RequireClass(classID int64) error {
	if !p.CanManageClass(classID) {
		return ErrAccessDenied
	}
	return nil
}

func (p Principal) RequireOffering(scope Scope) error {
	if !p.CanManageOffering(scope) {
		return ErrAccessDenied
	}
	return nil
}
