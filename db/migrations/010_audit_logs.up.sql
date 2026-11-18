CREATE TABLE audit_logs (
    id CHAR(36) NOT NULL PRIMARY KEY,
    organization_id CHAR(36) NOT NULL,
    user_id CHAR(36) NOT NULL,
    action VARCHAR(80) NOT NULL,
    resource VARCHAR(80) NOT NULL,
    resource_id VARCHAR(255) NULL,
    metadata_json JSON NULL,
    created_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    KEY audit_logs_organization_created_idx (organization_id, created_at),
    KEY audit_logs_user_idx (user_id),
    CONSTRAINT audit_logs_organization_fk FOREIGN KEY (organization_id) REFERENCES organizations (id),
    CONSTRAINT audit_logs_user_fk FOREIGN KEY (user_id) REFERENCES users (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
