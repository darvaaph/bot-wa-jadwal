package common

import (
	"context"
	"database/sql"
	"net/http"
)

type contextKey string

const userContextKey = contextKey("v1_user_context")

// UserContext menyimpan data sesi dan konteks penugasan peran aktif dari token pengguna.
type UserContext struct {
	SessionID              int64
	UserID                 int64
	IdentityKey            string
	DisplayName            string
	UserStatus             string
	SessionVersion         int
	ActiveAssignmentID     int64
	ActiveRole             string
	ActiveScopeType        string
	ActiveClassID          sql.NullInt64
	ActiveClassSlug        string
	ActiveSemesterID       sql.NullInt64
	ActiveCourseOfferingID sql.NullInt64
}

// GetAuthContext mengambil konteks pengguna terautentikasi dari request context.
func GetAuthContext(r *http.Request) (*UserContext, bool) {
	ctxVal := r.Context().Value(userContextKey)
	if ctxVal == nil {
		return nil, false
	}
	u, ok := ctxVal.(*UserContext)
	return u, ok
}

// WithAuthContext menyisipkan konteks pengguna ke dalam request context.
func WithAuthContext(r *http.Request, u *UserContext) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), userContextKey, u))
}
