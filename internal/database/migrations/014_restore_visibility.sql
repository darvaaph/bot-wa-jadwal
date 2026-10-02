ALTER TABLE room_confirmations ADD COLUMN superseded_at TEXT;
CREATE INDEX IF NOT EXISTS idx_room_confirmations_active_event ON room_confirmations(teaching_event_id, superseded_at);
