CREATE TABLE organization_invites (
    id CHAR(36) NOT NULL PRIMARY KEY,
    organization_id CHAR(36) NOT NULL,
    invited_by CHAR(36) NOT NULL,
    email VARCHAR(255) NOT NULL,
    role VARCHAR(20) NOT NULL,
    token_hash CHAR(64) NOT NULL,
    expires_at TIMESTAMP(6) NOT NULL,
    accepted_at TIMESTAMP(6) NULL,
    created_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    UNIQUE KEY organization_invites_token_unique (token_hash),
    KEY organization_invites_email_idx (organization_id, email),
    CONSTRAINT organization_invites_organization_fk FOREIGN KEY (organization_id) REFERENCES organizations (id),
    CONSTRAINT organization_invites_invited_by_fk FOREIGN KEY (invited_by) REFERENCES users (id),
    CONSTRAINT organization_invites_role_check CHECK (role IN ('admin', 'editor', 'viewer'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
