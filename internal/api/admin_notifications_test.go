package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func seedNotifications(t *testing.T, db interface {
	Exec(query string, args ...any) (Result, error)
}) {
	t.Helper()
}
