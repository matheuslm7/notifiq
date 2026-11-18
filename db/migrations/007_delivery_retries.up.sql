ALTER TABLE campaign_recipients
    ADD COLUMN attempts INT UNSIGNED NOT NULL DEFAULT 0 AFTER status,
    ADD COLUMN next_attempt_at TIMESTAMP(6) NULL AFTER sent_at;
