CREATE TABLE email_provider_configs (
    organization_id CHAR(36) NOT NULL PRIMARY KEY,
    provider VARCHAR(20) NOT NULL DEFAULT 'smtp',
    host VARCHAR(255) NOT NULL,
    port SMALLINT UNSIGNED NOT NULL,
    username VARCHAR(255) NOT NULL,
    encrypted_password TEXT NOT NULL,
    from_email VARCHAR(255) NOT NULL,
    from_name VARCHAR(120) NOT NULL,
    created_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    CONSTRAINT email_configs_organization_fk
        FOREIGN KEY (organization_id) REFERENCES organizations (id),
    CONSTRAINT email_configs_provider_check
        CHECK (provider IN ('smtp'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
