package database

import (
	"context"
	"crypto/rand"
	"database/sql"
	"fmt"
	"time"

	"github.com/matheuslm7/notifiq/internal/audit"
	"github.com/matheuslm7/notifiq/internal/auth"
	"github.com/matheuslm7/notifiq/internal/contacts"
	"github.com/matheuslm7/notifiq/internal/emailsettings"
	"github.com/matheuslm7/notifiq/internal/members"
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) CreateUser(ctx context.Context, user auth.User) (auth.User, error) {
	organizationID, err := newID()
	if err != nil {
		return auth.User{}, fmt.Errorf("generate organization id: %w", err)
	}
	userID := user.ID
	if userID == "" {
		userID, err = newID()
		if err != nil {
			return auth.User{}, fmt.Errorf("generate user id: %w", err)
		}
	}
	user.Role = "owner"

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return auth.User{}, fmt.Errorf("begin create user: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx,
		"INSERT INTO organizations (id, name) VALUES (?, ?)", organizationID, user.Name+"'s workspace",
	); err != nil {
		return auth.User{}, fmt.Errorf("insert organization: %w", err)
	}
	if _, err := tx.ExecContext(ctx,
		"INSERT INTO users (id, email, password_hash, name) VALUES (?, ?, ?, ?)",
		userID, user.Email, user.PasswordHash, user.Name,
	); err != nil {
		return auth.User{}, fmt.Errorf("insert user: %w", err)
	}
	if _, err := tx.ExecContext(ctx,
		"INSERT INTO organization_members (organization_id, user_id, role) VALUES (?, ?, 'owner')",
		organizationID, userID,
	); err != nil {
		return auth.User{}, fmt.Errorf("insert organization membership: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return auth.User{}, fmt.Errorf("commit create user: %w", err)
	}
	user.ID = userID
	user.OrganizationID = organizationID
	return user, nil
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (auth.User, error) {
	var user auth.User
	err := r.db.QueryRowContext(ctx, `
		SELECT u.id, u.email, u.name, u.password_hash, om.organization_id, om.role, u.mfa_enabled, u.mfa_secret
		FROM users u
		JOIN organization_members om ON om.user_id = u.id
		WHERE u.email = ? AND u.deleted_at IS NULL
		ORDER BY om.created_at ASC
		LIMIT 1
	`, email).Scan(&user.ID, &user.Email, &user.Name, &user.PasswordHash, &user.OrganizationID, &user.Role, &user.MFAEnabled, &user.MFASecret)
	if err == sql.ErrNoRows {
		return auth.User{}, auth.ErrUserNotFound
	}
	if err != nil {
		return auth.User{}, fmt.Errorf("query user: %w", err)
	}
	return user, nil
}

func (r *UserRepository) FindByID(ctx context.Context, id string) (auth.User, error) {
	var user auth.User
	err := r.db.QueryRowContext(ctx, `
		SELECT u.id, u.email, u.name, u.password_hash, om.organization_id, om.role, u.mfa_enabled, u.mfa_secret
		FROM users u
		JOIN organization_members om ON om.user_id = u.id
		WHERE u.id = ? AND u.deleted_at IS NULL
		ORDER BY om.created_at ASC
		LIMIT 1
	`, id).Scan(&user.ID, &user.Email, &user.Name, &user.PasswordHash, &user.OrganizationID, &user.Role, &user.MFAEnabled, &user.MFASecret)
	if err == sql.ErrNoRows {
		return auth.User{}, auth.ErrUserNotFound
	}
	if err != nil {
		return auth.User{}, fmt.Errorf("query user by id: %w", err)
	}
	return user, nil
}

func (r *UserRepository) SaveRefreshToken(ctx context.Context, userID, tokenHash string, expiresAt time.Time) error {
	id, err := newID()
	if err != nil {
		return fmt.Errorf("generate refresh token id: %w", err)
	}
	if _, err := r.db.ExecContext(ctx,
		"INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at) VALUES (?, ?, ?, ?)",
		id, userID, tokenHash, expiresAt,
	); err != nil {
		return fmt.Errorf("insert refresh token: %w", err)
	}
	return nil
}

func (r *UserRepository) RotateRefreshToken(ctx context.Context, currentHash, nextHash string, expiresAt time.Time) (auth.User, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return auth.User{}, fmt.Errorf("begin refresh rotation: %w", err)
	}
	defer tx.Rollback()

	var user auth.User
	err = tx.QueryRowContext(ctx, `
		SELECT u.id, u.email, u.name, u.password_hash, om.organization_id, om.role, u.mfa_enabled, u.mfa_secret
		FROM refresh_tokens rt
		JOIN users u ON u.id = rt.user_id
		JOIN organization_members om ON om.user_id = u.id
		WHERE rt.token_hash = ?
		  AND rt.revoked_at IS NULL
		  AND rt.expires_at > UTC_TIMESTAMP(6)
		  AND u.deleted_at IS NULL
		ORDER BY om.created_at ASC
		LIMIT 1
		FOR UPDATE
	`, currentHash).Scan(&user.ID, &user.Email, &user.Name, &user.PasswordHash, &user.OrganizationID, &user.Role, &user.MFAEnabled, &user.MFASecret)
	if err == sql.ErrNoRows {
		return auth.User{}, auth.ErrInvalidRefreshToken
	}
	if err != nil {
		return auth.User{}, fmt.Errorf("find refresh token: %w", err)
	}
	if _, err := tx.ExecContext(ctx,
		"UPDATE refresh_tokens SET revoked_at = UTC_TIMESTAMP(6) WHERE token_hash = ? AND revoked_at IS NULL",
		currentHash,
	); err != nil {
		return auth.User{}, fmt.Errorf("revoke current refresh token: %w", err)
	}
	nextID, err := newID()
	if err != nil {
		return auth.User{}, fmt.Errorf("generate rotated token id: %w", err)
	}
	if _, err := tx.ExecContext(ctx,
		"INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at) VALUES (?, ?, ?, ?)",
		nextID, user.ID, nextHash, expiresAt,
	); err != nil {
		return auth.User{}, fmt.Errorf("insert rotated refresh token: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return auth.User{}, fmt.Errorf("commit refresh rotation: %w", err)
	}
	return user, nil
}

func (r *UserRepository) SaveMFASecret(ctx context.Context, userID, secret string) error {
	if _, err := r.db.ExecContext(ctx, "UPDATE users SET mfa_secret = ?, mfa_enabled = FALSE WHERE id = ?", secret, userID); err != nil {
		return fmt.Errorf("save mfa secret: %w", err)
	}
	return nil
}

func (r *UserRepository) EnableMFA(ctx context.Context, userID string) error {
	if _, err := r.db.ExecContext(ctx, "UPDATE users SET mfa_enabled = TRUE WHERE id = ? AND mfa_secret IS NOT NULL", userID); err != nil {
		return fmt.Errorf("enable mfa: %w", err)
	}
	return nil
}

func (r *UserRepository) RevokeRefreshToken(ctx context.Context, tokenHash string) error {
	if _, err := r.db.ExecContext(ctx,
		"UPDATE refresh_tokens SET revoked_at = UTC_TIMESTAMP(6) WHERE token_hash = ? AND revoked_at IS NULL",
		tokenHash,
	); err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}
	return nil
}

func (r *UserRepository) SavePasswordResetToken(ctx context.Context, userID, tokenHash string, expiresAt time.Time) error {
	id, err := newID()
	if err != nil {
		return fmt.Errorf("generate password reset id: %w", err)
	}
	if _, err := r.db.ExecContext(ctx, "INSERT INTO password_reset_tokens (id, user_id, token_hash, expires_at) VALUES (?, ?, ?, ?)", id, userID, tokenHash, expiresAt); err != nil {
		return fmt.Errorf("insert password reset token: %w", err)
	}
	return nil
}

func (r *UserRepository) ConsumePasswordResetToken(ctx context.Context, tokenHash string) (string, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("begin password reset: %w", err)
	}
	defer tx.Rollback()
	var userID string
	if err := tx.QueryRowContext(ctx, "SELECT user_id FROM password_reset_tokens WHERE token_hash = ? AND used_at IS NULL AND expires_at > UTC_TIMESTAMP(6) FOR UPDATE", tokenHash).Scan(&userID); err == sql.ErrNoRows {
		return "", auth.ErrInvalidPasswordResetToken
	} else if err != nil {
		return "", fmt.Errorf("find password reset token: %w", err)
	}
	if _, err := tx.ExecContext(ctx, "UPDATE password_reset_tokens SET used_at = UTC_TIMESTAMP(6) WHERE token_hash = ? AND used_at IS NULL", tokenHash); err != nil {
		return "", fmt.Errorf("consume password reset token: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("commit password reset: %w", err)
	}
	return userID, nil
}

func (r *UserRepository) UpdatePassword(ctx context.Context, userID, passwordHash string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin password update: %w", err)
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, "UPDATE users SET password_hash = ? WHERE id = ? AND deleted_at IS NULL", passwordHash, userID); err != nil {
		return fmt.Errorf("update password: %w", err)
	}
	if _, err := tx.ExecContext(ctx, "UPDATE refresh_tokens SET revoked_at = UTC_TIMESTAMP(6) WHERE user_id = ? AND revoked_at IS NULL", userID); err != nil {
		return fmt.Errorf("revoke sessions after password update: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit password update: %w", err)
	}
	return nil
}

func (r *UserRepository) CreateInvite(ctx context.Context, invite members.Invite, tokenHash string) error {
	id, err := newID()
	if err != nil {
		return fmt.Errorf("generate invite id: %w", err)
	}
	_, err = r.db.ExecContext(ctx, `INSERT INTO organization_invites (id, organization_id, invited_by, email, role, token_hash, expires_at) VALUES (?, ?, ?, ?, ?, ?, ?)`, id, invite.OrganizationID, invite.InvitedBy, invite.Email, invite.Role, tokenHash, invite.ExpiresAt)
	if err != nil {
		return fmt.Errorf("insert invite: %w", err)
	}
	return nil
}

func (r *UserRepository) AcceptInvite(ctx context.Context, tokenHash, name, passwordHash, _ string) (members.Member, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return members.Member{}, fmt.Errorf("begin invite acceptance: %w", err)
	}
	defer tx.Rollback()
	var inviteID, organizationID, email, role string
	if err := tx.QueryRowContext(ctx, `SELECT id, organization_id, email, role FROM organization_invites WHERE token_hash = ? AND accepted_at IS NULL AND expires_at > UTC_TIMESTAMP(6) FOR UPDATE`, tokenHash).Scan(&inviteID, &organizationID, &email, &role); err == sql.ErrNoRows {
		return members.Member{}, members.ErrInvalidInvite
	} else if err != nil {
		return members.Member{}, fmt.Errorf("find invite: %w", err)
	}
	var userID string
	err = tx.QueryRowContext(ctx, "SELECT id FROM users WHERE email = ? AND deleted_at IS NULL FOR UPDATE", email).Scan(&userID)
	if err == sql.ErrNoRows {
		userID, err = newID()
		if err != nil {
			return members.Member{}, fmt.Errorf("generate invited user id: %w", err)
		}
		if _, err := tx.ExecContext(ctx, "INSERT INTO users (id, email, password_hash, name) VALUES (?, ?, ?, ?)", userID, email, passwordHash, name); err != nil {
			return members.Member{}, fmt.Errorf("insert invited user: %w", err)
		}
	} else if err != nil {
		return members.Member{}, fmt.Errorf("find invited user: %w", err)
	}
	if _, err := tx.ExecContext(ctx, "INSERT INTO organization_members (organization_id, user_id, role) VALUES (?, ?, ?)", organizationID, userID, role); err != nil {
		return members.Member{}, fmt.Errorf("insert invited membership: %w", err)
	}
	if _, err := tx.ExecContext(ctx, "UPDATE organization_invites SET accepted_at = UTC_TIMESTAMP(6) WHERE id = ?", inviteID); err != nil {
		return members.Member{}, fmt.Errorf("mark invite accepted: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return members.Member{}, fmt.Errorf("commit invite acceptance: %w", err)
	}
	return members.Member{UserID: userID, OrganizationID: organizationID, Name: name, Email: email, Role: role}, nil
}

func (r *UserRepository) GetEmailSettings(ctx context.Context, organizationID string) (emailsettings.Settings, error) {
	var settings emailsettings.Settings
	var password string
	err := r.db.QueryRowContext(ctx, `SELECT provider, host, port, username, encrypted_password, from_email, from_name FROM email_provider_configs WHERE organization_id = ?`, organizationID).
		Scan(&settings.Provider, &settings.Host, &settings.Port, &settings.Username, &password, &settings.FromEmail, &settings.FromName)
	if err == sql.ErrNoRows {
		return emailsettings.Settings{}, nil
	}
	if err != nil {
		return emailsettings.Settings{}, fmt.Errorf("query email settings: %w", err)
	}
	settings.HasPassword = password != ""
	return settings, nil
}

func (r *UserRepository) GetEmailSettingsForDelivery(ctx context.Context, organizationID string) (emailsettings.Settings, error) {
	var settings emailsettings.Settings
	err := r.db.QueryRowContext(ctx, "SELECT provider, host, port, username, encrypted_password, from_email, from_name FROM email_provider_configs WHERE organization_id = ?", organizationID).
		Scan(&settings.Provider, &settings.Host, &settings.Port, &settings.Username, &settings.Password, &settings.FromEmail, &settings.FromName)
	if err != nil {
		return emailsettings.Settings{}, fmt.Errorf("query delivery email settings: %w", err)
	}
	return settings, nil
}

func (r *UserRepository) SaveEmailSettings(ctx context.Context, organizationID string, settings emailsettings.Settings) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO email_provider_configs (organization_id, provider, host, port, username, encrypted_password, from_email, from_name)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE provider = VALUES(provider), host = VALUES(host), port = VALUES(port), username = VALUES(username), encrypted_password = VALUES(encrypted_password), from_email = VALUES(from_email), from_name = VALUES(from_name)
	`, organizationID, settings.Provider, settings.Host, settings.Port, settings.Username, settings.Password, settings.FromEmail, settings.FromName)
	if err != nil {
		return fmt.Errorf("save email settings: %w", err)
	}
	return nil
}

func (r *UserRepository) Record(ctx context.Context, entry audit.Entry) error {
	id, err := newID()
	if err != nil {
		return fmt.Errorf("generate audit id: %w", err)
	}
	metadata := entry.Metadata
	if metadata == "" {
		metadata = "{}"
	}
	if _, err := r.db.ExecContext(ctx, `INSERT INTO audit_logs (id, organization_id, user_id, action, resource, resource_id, metadata_json) VALUES (?, ?, ?, ?, ?, ?, ?)`, id, entry.OrganizationID, entry.UserID, entry.Action, entry.Resource, entry.ResourceID, metadata); err != nil {
		return fmt.Errorf("record audit log: %w", err)
	}
	return nil
}

var _ audit.Recorder = (*UserRepository)(nil)

func (r *UserRepository) ListMembers(ctx context.Context, organizationID string) ([]members.Member, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT om.user_id, om.organization_id, u.name, u.email, om.role, om.created_at
		FROM organization_members om
		JOIN users u ON u.id = om.user_id
		WHERE om.organization_id = ? AND u.deleted_at IS NULL
		ORDER BY FIELD(om.role, 'owner', 'admin', 'editor', 'viewer'), u.name ASC
	`, organizationID)
	if err != nil {
		return nil, fmt.Errorf("query members: %w", err)
	}
	defer rows.Close()
	items := make([]members.Member, 0)
	for rows.Next() {
		var item members.Member
		if err := rows.Scan(&item.UserID, &item.OrganizationID, &item.Name, &item.Email, &item.Role, &item.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan member: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate members: %w", err)
	}
	return items, nil
}

func (r *UserRepository) UpdateMemberRole(ctx context.Context, organizationID, memberID, role string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin member role update: %w", err)
	}
	defer tx.Rollback()
	var currentRole string
	if err := tx.QueryRowContext(ctx, "SELECT role FROM organization_members WHERE organization_id = ? AND user_id = ? FOR UPDATE", organizationID, memberID).Scan(&currentRole); err == sql.ErrNoRows {
		return members.ErrMemberNotFound
	} else if err != nil {
		return fmt.Errorf("find member role: %w", err)
	}
	if currentRole == "owner" && role != "owner" {
		var owners int
		if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM organization_members WHERE organization_id = ? AND role = 'owner'", organizationID).Scan(&owners); err != nil {
			return fmt.Errorf("count owners: %w", err)
		}
		if owners <= 1 {
			return members.ErrLastOwner
		}
	}
	if _, err := tx.ExecContext(ctx, "UPDATE organization_members SET role = ? WHERE organization_id = ? AND user_id = ?", role, organizationID, memberID); err != nil {
		return fmt.Errorf("update member role: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit member role update: %w", err)
	}
	return nil
}

func (r *UserRepository) RemoveMember(ctx context.Context, organizationID, memberID string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin member removal: %w", err)
	}
	defer tx.Rollback()
	var role string
	if err := tx.QueryRowContext(ctx, "SELECT role FROM organization_members WHERE organization_id = ? AND user_id = ? FOR UPDATE", organizationID, memberID).Scan(&role); err == sql.ErrNoRows {
		return members.ErrMemberNotFound
	} else if err != nil {
		return fmt.Errorf("find member: %w", err)
	}
	if role == "owner" {
		var owners int
		if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM organization_members WHERE organization_id = ? AND role = 'owner'", organizationID).Scan(&owners); err != nil {
			return fmt.Errorf("count owners: %w", err)
		}
		if owners <= 1 {
			return members.ErrLastOwner
		}
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM organization_members WHERE organization_id = ? AND user_id = ?", organizationID, memberID); err != nil {
		return fmt.Errorf("remove member: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit member removal: %w", err)
	}
	return nil
}

func (r *UserRepository) ListContacts(ctx context.Context, organizationID string) ([]contacts.Contact, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT id, organization_id, name, email, created_at FROM contacts WHERE organization_id = ? AND deleted_at IS NULL ORDER BY created_at DESC", organizationID)
	if err != nil {
		return nil, fmt.Errorf("query contacts: %w", err)
	}
	defer rows.Close()
	items := make([]contacts.Contact, 0)
	for rows.Next() {
		var contact contacts.Contact
		if err := rows.Scan(&contact.ID, &contact.OrganizationID, &contact.Name, &contact.Email, &contact.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan contact: %w", err)
		}
		items = append(items, contact)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate contacts: %w", err)
	}
	return items, nil
}

func (r *UserRepository) CreateContact(ctx context.Context, contact contacts.Contact) (contacts.Contact, error) {
	id, err := newID()
	if err != nil {
		return contacts.Contact{}, fmt.Errorf("generate contact id: %w", err)
	}
	if _, err := r.db.ExecContext(ctx, "INSERT INTO contacts (id, organization_id, name, email) VALUES (?, ?, ?, ?)", id, contact.OrganizationID, contact.Name, contact.Email); err != nil {
		return contacts.Contact{}, fmt.Errorf("insert contact: %w", err)
	}
	contact.ID = id
	return contact, nil
}

func (r *UserRepository) CreateContacts(ctx context.Context, items []contacts.Contact) (int, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin bulk contacts: %w", err)
	}
	defer tx.Rollback()
	imported := 0
	for _, contact := range items {
		id, err := newID()
		if err != nil {
			return 0, fmt.Errorf("generate contact id: %w", err)
		}
		result, err := tx.ExecContext(ctx, "INSERT IGNORE INTO contacts (id, organization_id, name, email) VALUES (?, ?, ?, ?)", id, contact.OrganizationID, contact.Name, contact.Email)
		if err != nil {
			return 0, fmt.Errorf("insert bulk contact: %w", err)
		}
		count, err := result.RowsAffected()
		if err != nil {
			return 0, fmt.Errorf("read bulk contact result: %w", err)
		}
		imported += int(count)
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit bulk contacts: %w", err)
	}
	return imported, nil
}

func newID() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	bytes[6] = (bytes[6] & 0x0f) | 0x40
	bytes[8] = (bytes[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", bytes[0:4], bytes[4:6], bytes[6:8], bytes[8:10], bytes[10:]), nil
}
