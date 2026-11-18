CREATE TABLE campaign_recipients (
    campaign_id CHAR(36) NOT NULL,
    contact_id CHAR(36) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    sent_at TIMESTAMP(6) NULL,
    error_message VARCHAR(500) NULL,
    PRIMARY KEY (campaign_id, contact_id),
    CONSTRAINT campaign_recipients_campaign_fk FOREIGN KEY (campaign_id) REFERENCES campaigns (id),
    CONSTRAINT campaign_recipients_contact_fk FOREIGN KEY (contact_id) REFERENCES contacts (id),
    CONSTRAINT campaign_recipients_status_check CHECK (status IN ('pending', 'sending', 'sent', 'failed', 'cancelled'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
