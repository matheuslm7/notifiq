ALTER TABLE campaign_recipients
    ADD COLUMN provider_message_id VARCHAR(255) NULL AFTER sent_at,
    ADD KEY campaign_recipients_provider_message_idx (provider_message_id),
    DROP CHECK campaign_recipients_status_check,
    ADD CONSTRAINT campaign_recipients_status_check CHECK (status IN ('pending', 'sending', 'sent', 'delivered', 'opened', 'clicked', 'failed', 'cancelled'));

ALTER TABLE message_events
    ADD COLUMN provider_message_id VARCHAR(255) NULL AFTER provider_event_id,
    ADD KEY message_events_provider_message_idx (provider_message_id);
