package v1

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"bot-jadwal/internal/schedule"
)

func checkPatternRange(ctx context.Context, db *sql.DB, candidate schedule.Candidate, from, until string) ([]schedule.Conflict, error) {
	start, err := time.Parse("2006-01-02", from)
	if err != nil {
		return nil, err
	}
	end, err := time.Parse("2006-01-02", until)
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	out := []schedule.Conflict{}
	for day := start; !day.After(end); day = day.AddDate(0, 0, 1) {
		weekday := int(day.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		if weekday != candidate.PatternDay {
			continue
		}
		candidate.PatternDate = day.Format("2006-01-02")
		conflicts, err := schedule.CheckConflicts(ctx, db, candidate)
		if err != nil {
			return nil, err
		}
		for _, conflict := range conflicts {
			key := fmt.Sprintf("%s:%s:%d", conflict.Code, conflict.EntityType, conflict.EntityID)
			if !seen[key] {
				seen[key] = true
				out = append(out, conflict)
			}
		}
	}
	return out, nil
}
