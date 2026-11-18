CREATE TABLE message_events (
    id CHAR(36) NOT NULL PRIMARY KEY,
    provider VARCHAR(40) NOT NULL,
    provider_event_id VARCHAR(255) NOT NULL,
    event_type VARCHAR(40) NOT NULL,
    recipient_email VARCHAR(255) NOT NULL,
    payload_json JSON NOT NULL,
    occurred_at TIMESTAMP(6) NULL,
    created_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    UNIQUE KEY message_events_provider_event_unique (provider, provider_event_id),
    KEY message_events_recipient_idx (recipient_email)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
