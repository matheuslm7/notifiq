ALTER TABLE campaign_recipients
    DROP INDEX campaign_recipients_queue_idx,
    DROP COLUMN queued_at;
