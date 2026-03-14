CREATE INDEX IF NOT EXISTS idx_outbox_status_next_retry_id ON outbox(status, next_retry_at, id);
CREATE INDEX IF NOT EXISTS idx_outbox_status_updated_at ON outbox(status, updated_at);
