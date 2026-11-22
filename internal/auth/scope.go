package auth

import (
	"context"
	"database/sql"
)

type ScopeReader interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

// ValidateScope checks the relationship, including assignments written before validation existed.
func ValidateScope(ctx context.Context, q ScopeReader, role, scope string, classID, semesterID, offeringID sql.NullInt64) error {
	in := InvitationInput{Role: role, ClassID: nullInt64Ptr(classID), SemesterID: nullInt64Ptr(semesterID), CourseOfferingID: nullInt64Ptr(offeringID)}
	want, err := validateInvitationScope(in)
	if err != nil || want != scope {
		return ErrAccessDenied
	}
	if role == RolePJ {
		var valid bool
		if err := q.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM course_offerings co JOIN semesters sem ON sem.id=co.semester_id WHERE co.id=? AND sem.id=? AND sem.class_id=?)`, offeringID, semesterID, classID).Scan(&valid); err != nil {
			return err
		}
		if !valid {
			return ErrAccessDenied
		}
	}
	return nil
}
