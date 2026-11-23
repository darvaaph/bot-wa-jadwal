package schedule

import (
	"context"
	"fmt"
	"time"
)

// CheckPatternRange checks each occurrence and reuses lecturer assignments across dates.
func CheckPatternRange(ctx context.Context, db conflictReader, candidate Candidate, from, until string) ([]Conflict, error) {
	start, err := time.Parse("2006-01-02", from)
	if err != nil {
		return nil, err
	}
	end, err := time.Parse("2006-01-02", until)
	if err != nil {
		return nil, err
	}
	lecturers := map[int64][]int64{}
	seen := map[string]bool{}
	out := []Conflict{}
	for day := start; !day.After(end); day = day.AddDate(0, 0, 1) {
		weekday := int(day.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		if weekday != candidate.PatternDay {
			continue
		}
		candidate.PatternDate = day.Format("2006-01-02")
		conflicts, err := checkConflicts(ctx, db, candidate, lecturers)
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
