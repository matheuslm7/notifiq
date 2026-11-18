package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/matheuslm7/notifiq/internal/campaigns"
	"github.com/matheuslm7/notifiq/internal/delivery"
	"github.com/matheuslm7/notifiq/internal/metrics"
)

func (r *UserRepository) ListTemplates(ctx context.Context, organizationID string) ([]campaigns.Template, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT id, organization_id, name, subject, body_html, created_at FROM email_templates WHERE organization_id = ? AND deleted_at IS NULL ORDER BY created_at DESC", organizationID)
	if err != nil {
		return nil, fmt.Errorf("query templates: %w", err)
	}
	defer rows.Close()
	items := make([]campaigns.Template, 0)
	for rows.Next() {
		var item campaigns.Template
		if err := rows.Scan(&item.ID, &item.OrganizationID, &item.Name, &item.Subject, &item.BodyHTML, &item.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan template: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *UserRepository) CreateTemplate(ctx context.Context, item campaigns.Template) (campaigns.Template, error) {
	id, err := newID()
	if err != nil {
		return campaigns.Template{}, fmt.Errorf("generate template id: %w", err)
	}
	if _, err := r.db.ExecContext(ctx, "INSERT INTO email_templates (id, organization_id, name, subject, body_html) VALUES (?, ?, ?, ?, ?)", id, item.OrganizationID, item.Name, item.Subject, item.BodyHTML); err != nil {
		return campaigns.Template{}, fmt.Errorf("insert template: %w", err)
	}
	item.ID = id
	return item, nil
}

func (r *UserRepository) DeleteTemplate(ctx context.Context, organizationID, templateID string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin template deletion: %w", err)
	}
	defer tx.Rollback()
	rows, err := tx.QueryContext(ctx, "SELECT id FROM campaigns WHERE organization_id = ? AND template_id = ?", organizationID, templateID)
	if err != nil {
		return fmt.Errorf("find template campaigns: %w", err)
	}
	for rows.Next() {
		var campaignID string
		if err := rows.Scan(&campaignID); err != nil {
			rows.Close()
			return fmt.Errorf("scan campaign id: %w", err)
		}
		if _, err := tx.ExecContext(ctx, "DELETE FROM campaign_recipients WHERE campaign_id = ?", campaignID); err != nil {
			rows.Close()
			return fmt.Errorf("delete campaign recipients: %w", err)
		}
		if _, err := tx.ExecContext(ctx, "DELETE FROM campaigns WHERE id = ? AND organization_id = ?", campaignID, organizationID); err != nil {
			rows.Close()
			return fmt.Errorf("delete campaign: %w", err)
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("iterate linked campaigns: %w", err)
	}
	rows.Close()
	result, err := tx.ExecContext(ctx, "UPDATE email_templates SET deleted_at = UTC_TIMESTAMP(6) WHERE id = ? AND organization_id = ? AND deleted_at IS NULL", templateID, organizationID)
	if err != nil {
		return fmt.Errorf("soft delete template: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read template delete result: %w", err)
	}
	if count == 0 {
		return campaigns.ErrCampaignNotEditable
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit template deletion: %w", err)
	}
	return nil
}

func (r *UserRepository) ListCampaigns(ctx context.Context, organizationID string) ([]campaigns.Campaign, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT id, organization_id, template_id, name, status, scheduled_at, created_at FROM campaigns WHERE organization_id = ? ORDER BY created_at DESC", organizationID)
	if err != nil {
		return nil, fmt.Errorf("query campaigns: %w", err)
	}
	defer rows.Close()
	items := make([]campaigns.Campaign, 0)
	for rows.Next() {
		var item campaigns.Campaign
		if err := rows.Scan(&item.ID, &item.OrganizationID, &item.TemplateID, &item.Name, &item.Status, &item.ScheduledAt, &item.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan campaign: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *UserRepository) CreateCampaign(ctx context.Context, item campaigns.Campaign) (campaigns.Campaign, error) {
	id, err := newID()
	if err != nil {
		return campaigns.Campaign{}, fmt.Errorf("generate campaign id: %w", err)
	}
	if _, err := r.db.ExecContext(ctx, "INSERT INTO campaigns (id, organization_id, template_id, name, status, scheduled_at) VALUES (?, ?, ?, ?, ?, ?)", id, item.OrganizationID, item.TemplateID, item.Name, item.Status, item.ScheduledAt); err != nil {
		return campaigns.Campaign{}, fmt.Errorf("insert campaign: %w", err)
	}
	item.ID = id
	return item, nil
}

func (r *UserRepository) DeleteCampaign(ctx context.Context, organizationID, campaignID string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin campaign deletion: %w", err)
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, "DELETE FROM campaign_recipients WHERE campaign_id = ?", campaignID); err != nil {
		return fmt.Errorf("delete campaign recipients: %w", err)
	}
	result, err := tx.ExecContext(ctx, "DELETE FROM campaigns WHERE id = ? AND organization_id = ?", campaignID, organizationID)
	if err != nil {
		return fmt.Errorf("delete campaign: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read campaign delete result: %w", err)
	}
	if count == 0 {
		return campaigns.ErrCampaignNotEditable
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit campaign deletion: %w", err)
	}
	return nil
}

func (r *UserRepository) AddRecipients(ctx context.Context, organizationID, campaignID string, contactIDs []string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin add recipients: %w", err)
	}
	defer tx.Rollback()
	var exists bool
	if err := tx.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM campaigns WHERE id = ? AND organization_id = ?)", campaignID, organizationID).Scan(&exists); err != nil {
		return fmt.Errorf("check campaign ownership: %w", err)
	}
	if !exists {
		return fmt.Errorf("campaign not found")
	}
	for _, contactID := range contactIDs {
		if _, err := tx.ExecContext(ctx, "INSERT IGNORE INTO campaign_recipients (campaign_id, contact_id) SELECT ?, id FROM contacts WHERE id = ? AND organization_id = ? AND deleted_at IS NULL", campaignID, contactID, organizationID); err != nil {
			return fmt.Errorf("add campaign recipient: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit recipients: %w", err)
	}
	return nil
}

func (r *UserRepository) ClaimRecipient(ctx context.Context, campaignID, contactID string) (bool, int, error) {
	result, err := r.db.ExecContext(ctx, "UPDATE campaign_recipients SET status = 'sending', attempts = attempts + 1, queued_at = NULL WHERE campaign_id = ? AND contact_id = ? AND status = 'pending' AND (next_attempt_at IS NULL OR next_attempt_at <= UTC_TIMESTAMP(6))", campaignID, contactID)
	if err != nil {
		return false, 0, fmt.Errorf("claim recipient: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return false, 0, fmt.Errorf("read claim result: %w", err)
	}
	if count != 1 {
		return false, 0, nil
	}
	var attempts int
	if err := r.db.QueryRowContext(ctx, "SELECT attempts FROM campaign_recipients WHERE campaign_id = ? AND contact_id = ?", campaignID, contactID).Scan(&attempts); err != nil {
		return false, 0, fmt.Errorf("read recipient attempts: %w", err)
	}
	return true, attempts, nil
}

func (r *UserRepository) MarkRecipientSent(ctx context.Context, campaignID, contactID, messageID string) error {
	if _, err := r.db.ExecContext(ctx, "UPDATE campaign_recipients SET status = 'sent', sent_at = UTC_TIMESTAMP(6), provider_message_id = ?, error_message = NULL WHERE campaign_id = ? AND contact_id = ? AND status = 'sending'", messageID, campaignID, contactID); err != nil {
		return fmt.Errorf("mark recipient sent: %w", err)
	}
	return nil
}

func (r *UserRepository) MarkRecipientFailed(ctx context.Context, campaignID, contactID, message string) error {
	if len(message) > 500 {
		message = message[:500]
	}
	if _, err := r.db.ExecContext(ctx, "UPDATE campaign_recipients SET status = 'failed', error_message = ? WHERE campaign_id = ? AND contact_id = ? AND status = 'sending'", message, campaignID, contactID); err != nil {
		return fmt.Errorf("mark recipient failed: %w", err)
	}
	return nil
}

func (r *UserRepository) MarkRecipientRetry(ctx context.Context, campaignID, contactID, message string, nextAttemptAt time.Time) error {
	if len(message) > 500 {
		message = message[:500]
	}
	if _, err := r.db.ExecContext(ctx, "UPDATE campaign_recipients SET status = 'pending', error_message = ?, next_attempt_at = ?, queued_at = NULL WHERE campaign_id = ? AND contact_id = ? AND status = 'sending'", message, nextAttemptAt, campaignID, contactID); err != nil {
		return fmt.Errorf("schedule recipient retry: %w", err)
	}
	return nil
}

var _ delivery.Repository = (*UserRepository)(nil)

func (r *UserRepository) ListDueEmailJobs(ctx context.Context, now time.Time, limit int) ([]delivery.EmailJob, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT c.organization_id, c.id, cr.contact_id, co.email, co.name,
			epc.from_email, epc.from_name, et.subject, et.body_html
		FROM campaigns c
		JOIN campaign_recipients cr ON cr.campaign_id = c.id AND cr.status = 'pending'
		JOIN contacts co ON co.id = cr.contact_id AND co.deleted_at IS NULL
		JOIN email_templates et ON et.id = c.template_id AND et.deleted_at IS NULL
		JOIN email_provider_configs epc ON epc.organization_id = c.organization_id
		WHERE c.status IN ('scheduled', 'sending') AND c.scheduled_at <= ? AND (cr.next_attempt_at IS NULL OR cr.next_attempt_at <= UTC_TIMESTAMP(6)) AND (cr.queued_at IS NULL OR cr.queued_at <= UTC_TIMESTAMP(6) - INTERVAL 5 MINUTE)
		ORDER BY c.scheduled_at ASC
		LIMIT ?
	`, now, limit)
	if err != nil {
		return nil, fmt.Errorf("query due jobs: %w", err)
	}
	defer rows.Close()
	jobs := make([]delivery.EmailJob, 0)
	for rows.Next() {
		var job delivery.EmailJob
		if err := rows.Scan(&job.OrganizationID, &job.CampaignID, &job.ContactID, &job.To, &job.ToName, &job.From, &job.FromName, &job.Subject, &job.BodyHTML); err != nil {
			return nil, fmt.Errorf("scan due job: %w", err)
		}
		jobs = append(jobs, job)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate due jobs: %w", err)
	}
	return jobs, nil
}

func (r *UserRepository) MarkEmailJobQueued(ctx context.Context, campaignID, contactID string) error {
	if _, err := r.db.ExecContext(ctx, "UPDATE campaign_recipients SET queued_at = UTC_TIMESTAMP(6) WHERE campaign_id = ? AND contact_id = ? AND status = 'pending'", campaignID, contactID); err != nil {
		return fmt.Errorf("mark email job queued: %w", err)
	}
	return nil
}

func (r *UserRepository) MarkCampaignSending(ctx context.Context, campaignID string) error {
	if _, err := r.db.ExecContext(ctx, "UPDATE campaigns SET status = 'sending' WHERE id = ? AND status = 'scheduled'", campaignID); err != nil {
		return fmt.Errorf("mark campaign sending: %w", err)
	}
	return nil
}

func (r *UserRepository) UpdateCampaign(ctx context.Context, organizationID, campaignID, templateID string, scheduledAt *time.Time, name string) error {
	result, err := r.db.ExecContext(ctx, "UPDATE campaigns SET template_id = ?, name = ?, scheduled_at = ?, status = CASE WHEN ? IS NULL THEN 'draft' ELSE 'scheduled' END WHERE id = ? AND organization_id = ? AND status IN ('draft', 'scheduled')", templateID, name, scheduledAt, scheduledAt, campaignID, organizationID)
	if err != nil {
		return fmt.Errorf("update campaign: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read campaign update result: %w", err)
	}
	if count == 0 {
		return campaigns.ErrCampaignNotEditable
	}
	return nil
}

func (r *UserRepository) CancelCampaign(ctx context.Context, organizationID, campaignID string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin campaign cancellation: %w", err)
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, "UPDATE campaigns SET status = 'cancelled' WHERE id = ? AND organization_id = ? AND status IN ('draft', 'scheduled')", campaignID, organizationID)
	if err != nil {
		return fmt.Errorf("cancel campaign: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read cancellation result: %w", err)
	}
	if count == 0 {
		return campaigns.ErrCampaignNotEditable
	}
	if _, err := tx.ExecContext(ctx, "UPDATE campaign_recipients SET status = 'cancelled' WHERE campaign_id = ? AND status IN ('pending', 'sending')", campaignID); err != nil {
		return fmt.Errorf("cancel campaign recipients: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit campaign cancellation: %w", err)
	}
	return nil
}

func (r *UserRepository) DuplicateCampaign(ctx context.Context, organizationID, campaignID string) (campaigns.Campaign, error) {
	var source campaigns.Campaign
	err := r.db.QueryRowContext(ctx, "SELECT id, template_id, name FROM campaigns WHERE id = ? AND organization_id = ?", campaignID, organizationID).Scan(&source.ID, &source.TemplateID, &source.Name)
	if err == sql.ErrNoRows {
		return campaigns.Campaign{}, campaigns.ErrCampaignNotEditable
	}
	if err != nil {
		return campaigns.Campaign{}, fmt.Errorf("find campaign to duplicate: %w", err)
	}
	id, err := newID()
	if err != nil {
		return campaigns.Campaign{}, fmt.Errorf("generate duplicated campaign id: %w", err)
	}
	name := source.Name + " (copy)"
	if _, err := r.db.ExecContext(ctx, "INSERT INTO campaigns (id, organization_id, template_id, name, status) VALUES (?, ?, ?, ?, 'draft')", id, organizationID, source.TemplateID, name); err != nil {
		return campaigns.Campaign{}, fmt.Errorf("duplicate campaign: %w", err)
	}
	source.ID, source.OrganizationID, source.Name, source.Status = id, organizationID, name, "draft"
	return source, nil
}

func (r *UserRepository) GetDeliveryStats(ctx context.Context, organizationID string) (metrics.DeliveryStats, error) {
	var stats metrics.DeliveryStats
	err := r.db.QueryRowContext(ctx, `
		SELECT
			COALESCE(COUNT(cr.campaign_id), 0),
			COALESCE(SUM(CASE WHEN cr.status = 'pending' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN cr.status = 'sending' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN cr.status = 'sent' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN cr.status = 'failed' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN cr.status = 'cancelled' THEN 1 ELSE 0 END), 0)
		FROM campaigns c
		LEFT JOIN campaign_recipients cr ON cr.campaign_id = c.id
		WHERE c.organization_id = ?
	`, organizationID).Scan(&stats.Total, &stats.Pending, &stats.Sending, &stats.Sent, &stats.Failed, &stats.Cancelled)
	if err == sql.ErrNoRows {
		return metrics.DeliveryStats{}, nil
	}
	if err != nil {
		return metrics.DeliveryStats{}, fmt.Errorf("query delivery stats: %w", err)
	}
	return stats, nil
}

func (r *UserRepository) ListDeliveryHistory(ctx context.Context, organizationID, campaignID string) ([]metrics.DeliveryHistory, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT cr.campaign_id, cr.contact_id, co.name, co.email, cr.status, cr.attempts, cr.sent_at,
			me.id, me.provider_event_id, me.provider_message_id, me.event_type, me.occurred_at
		FROM campaign_recipients cr
		JOIN campaigns c ON c.id = cr.campaign_id AND c.organization_id = ?
		JOIN contacts co ON co.id = cr.contact_id
		LEFT JOIN message_events me ON me.provider_message_id = cr.provider_message_id
		WHERE cr.campaign_id = ?
		ORDER BY co.name ASC, me.occurred_at ASC
	`, organizationID, campaignID)
	if err != nil {
		return nil, fmt.Errorf("query delivery history: %w", err)
	}
	defer rows.Close()
	items := make([]metrics.DeliveryHistory, 0)
	indexes := make(map[string]int)
	for rows.Next() {
		var item metrics.DeliveryHistory
		var event metrics.DeliveryEvent
		var eventID, eventProviderID, eventMessageID, eventType sql.NullString
		var eventOccurredAt sql.NullTime
		if err := rows.Scan(&item.CampaignID, &item.ContactID, &item.ContactName, &item.Email, &item.Status, &item.Attempts, &item.SentAt, &eventID, &eventProviderID, &eventMessageID, &eventType, &eventOccurredAt); err != nil {
			return nil, fmt.Errorf("scan delivery history: %w", err)
		}
		index, exists := indexes[item.ContactID]
		if !exists {
			item.Events = make([]metrics.DeliveryEvent, 0)
			items = append(items, item)
			index = len(items) - 1
			indexes[item.ContactID] = index
		}
		if eventID.Valid {
			event = metrics.DeliveryEvent{ID: eventID.String, ProviderEventID: eventProviderID.String, ProviderMessageID: eventMessageID.String, Type: eventType.String}
			if eventOccurredAt.Valid {
				event.OccurredAt = &eventOccurredAt.Time
			}
			items[index].Events = append(items[index].Events, event)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate delivery history: %w", err)
	}
	return items, nil
}
