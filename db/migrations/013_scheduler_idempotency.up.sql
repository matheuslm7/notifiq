ALTER TABLE campaign_recipients
    ADD COLUMN queued_at TIMESTAMP(6) NULL AFTER next_attempt_at,
    ADD KEY campaign_recipients_queue_idx (queued_at);
