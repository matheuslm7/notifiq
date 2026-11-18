ALTER TABLE message_events
    DROP INDEX message_events_provider_message_idx,
    DROP COLUMN provider_message_id;

ALTER TABLE campaign_recipients
    DROP INDEX campaign_recipients_provider_message_idx,
    DROP COLUMN provider_message_id,
    DROP CHECK campaign_recipients_status_check,
    ADD CONSTRAINT campaign_recipients_status_check CHECK (status IN ('pending', 'sending', 'sent', 'failed', 'cancelled'));
