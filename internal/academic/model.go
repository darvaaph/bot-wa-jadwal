package academic

import "time"

// Status standar entitas akademik.
const (
	StatusActive   = "ACTIVE"
	StatusInactive = "INACTIVE"
	StatusArchived = "ARCHIVED"
	StatusDraft    = "DRAFT"
)

type Class struct {
	ID           int64     `json:"id"`
	Code         string    `json:"code"`
	Slug         string    `json:"slug"`
	StudyProgram string    `json:"study_program"`
	CohortYear   int       `json:"cohort_year"`
	GroupLabel   string    `json:"group_label"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type ClassSettings struct {
	ID                         int64     `json:"id"`
	ClassID                    int64     `json:"class_id"`
	Timezone                   string    `json:"timezone"`
	PortalAccessMode           string    `json:"portal_access_mode"`
	PortalCodeHash             *string   `json:"-"`
	PortalCodeVersion          int       `json:"portal_code_version"`
	MeetingLinkVisibility      string    `json:"meeting_link_visibility"`
	MorningReminderTime        *string   `json:"morning_reminder_time,omitempty"`
	AfternoonReminderTime      *string   `json:"afternoon_reminder_time,omitempty"`
	ReplacementReminderMinutes int       `json:"replacement_reminder_minutes"`
	Version                    int       `json:"version"`
	CreatedAt                  time.Time `json:"created_at"`
	UpdatedAt                  time.Time `json:"updated_at"`
}

type Semester struct {
	ID           int64      `json:"id"`
	ClassID      int64      `json:"class_id"`
	AcademicYear string     `json:"academic_year"`
	Term         string     `json:"term"`
	StartsOn     string     `json:"starts_on"`
	EndsOn       string     `json:"ends_on"`
	Status       string     `json:"status"`
	PublishedAt  *time.Time `json:"published_at,omitempty"`
	ActivatedAt  *time.Time `json:"activated_at,omitempty"`
	ArchivedAt   *time.Time `json:"archived_at,omitempty"`
	Version      int        `json:"version"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type Course struct {
	ID        int64     `json:"id"`
	Code      string    `json:"code"`
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CourseOffering struct {
	ID           int64     `json:"id"`
	SemesterID   int64     `json:"semester_id"`
	CourseID     int64     `json:"course_id"`
	ActivityType string    `json:"activity_type"`
	DisplayName  string    `json:"display_name"`
	Status       string    `json:"status"`
	Version      int       `json:"version"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Lecturer struct {
	ID        int64     `json:"id"`
	Code      string    `json:"code"`
	FullName  string    `json:"full_name"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type OfferingLecturer struct {
	ID               int64     `json:"id"`
	CourseOfferingID int64     `json:"course_offering_id"`
	LecturerID       int64     `json:"lecturer_id"`
	Responsibility   string    `json:"responsibility"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type User struct {
	ID             int64      `json:"id"`
	IdentityKey    string     `json:"identity_key"`
	DisplayName    string     `json:"display_name"`
	PasswordHash   string     `json:"-"`
	Status         string     `json:"status"`
	SessionVersion int        `json:"session_version"`
	LastLoginAt    *time.Time `json:"last_login_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type Room struct {
	ID              int64     `json:"id"`
	Code            string    `json:"code"`
	Name            string    `json:"name"`
	Building        *string   `json:"building,omitempty"`
	RoomType        *string   `json:"room_type,omitempty"`
	Capacity        *int      `json:"capacity,omitempty"`
	Status          string    `json:"status"`
	SourceUpdatedAt *string   `json:"source_updated_at,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type SchedulePattern struct {
	ID               int64     `json:"id"`
	CourseOfferingID int64     `json:"course_offering_id"`
	RoomID           *int64    `json:"room_id,omitempty"`
	DayOfWeek        int       `json:"day_of_week"`
	StartTime        string    `json:"start_time"`
	EndTime          string    `json:"end_time"`
	EffectiveFrom    string    `json:"effective_from"`
	EffectiveUntil   *string   `json:"effective_until,omitempty"`
	Status           string    `json:"status"`
	Version          int       `json:"version"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}
