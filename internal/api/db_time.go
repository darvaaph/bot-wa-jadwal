package api

import (
	"bot-jadwal/internal/api/common"
)

// dbTimestamp accepts timestamps returned by both the legacy DATETIME schema
// and the canonical TEXT/RFC3339 schema.
type dbTimestamp = common.DBTimestamp
