package task

type Task struct {
	ID                int64   `json:"id"`
	CourseOfferingID  int64   `json:"course_offering_id"`
	Title             string  `json:"title"`
	Instructions      string  `json:"instructions"`
	DeadlineAt        string  `json:"deadline_at"`
	TaskType          string  `json:"task_type"`
	SubmissionText    *string `json:"submission_text,omitempty"`
	SubmissionURL     *string `json:"submission_url,omitempty"`
	PublicationStatus string  `json:"publication_status"`
	ReviewState       string  `json:"review_state"`
	ReviewedVersion   *int    `json:"reviewed_version,omitempty"`
	CreatedByUserID   int64   `json:"created_by_user_id"`
	PublishedAt       *string `json:"published_at,omitempty"`
	CompletedAt       *string `json:"completed_at,omitempty"`
	ArchivedAt        *string `json:"archived_at,omitempty"`
	Version           int     `json:"version"`
	DeletedAt         *string `json:"deleted_at,omitempty"`
	DeletedByUserID   *int64  `json:"deleted_by_user_id,omitempty"`
	CreatedAt         string  `json:"created_at"`
	UpdatedAt         string  `json:"updated_at"`
}

type TaskReview struct {
	ID                       int64   `json:"id"`
	TaskID                   int64   `json:"task_id"`
	TaskVersion              int     `json:"task_version"`
	Decision                 string  `json:"decision"`
	Note                     *string `json:"note,omitempty"`
	ReviewerRoleAssignmentID int64   `json:"reviewer_role_assignment_id"`
	ReviewedAt               string  `json:"reviewed_at"`
	CreatedAt                string  `json:"created_at"`
	UpdatedAt                string  `json:"updated_at"`
}

type TaskItemView struct {
	Task
	CourseCode string `json:"course_code"`
	CourseName string `json:"course_name"`
	ClassCode  string `json:"class_code"`
}

type CreateTaskInput struct {
	CourseOfferingID int64   `json:"course_offering_id"`
	Title            string  `json:"title"`
	Instructions     string  `json:"instructions"`
	DeadlineAt       string  `json:"deadline_at"`
	TaskType         string  `json:"task_type"`
	SubmissionText   *string `json:"submission_text,omitempty"`
	SubmissionURL    *string `json:"submission_url,omitempty"`
	CreatedByUserID  int64   `json:"created_by_user_id"`
}

type ReviewTaskInput struct {
	TaskID                   int64   `json:"task_id"`
	Decision                 string  `json:"decision"`
	Note                     *string `json:"note,omitempty"`
	ReviewerRoleAssignmentID int64   `json:"reviewer_role_assignment_id"`
}
