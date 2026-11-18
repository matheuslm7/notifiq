ALTER TABLE users
    ADD COLUMN mfa_secret VARCHAR(128) NULL AFTER mfa_enabled;
