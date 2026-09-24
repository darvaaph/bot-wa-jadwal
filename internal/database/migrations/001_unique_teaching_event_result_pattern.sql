DROP INDEX IF EXISTS idx_teaching_events_result_pattern;

CREATE UNIQUE INDEX IF NOT EXISTS uq_teaching_events_result_pattern
    ON teaching_events(result_schedule_pattern_id)
    WHERE result_schedule_pattern_id IS NOT NULL;
