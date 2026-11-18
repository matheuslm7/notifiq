package database

import (
	"context"
	"fmt"

	"github.com/matheuslm7/notifiq/internal/webhook"
)

func (r *UserRepository) RecordWebhookEvent(ctx context.Context, provider string, event webhook.Event) error {
	id, err := newID()
	if err != nil {
		return fmt.Errorf("generate webhook event id: %w", err)
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin webhook event: %w", err)
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `
		INSERT IGNORE INTO message_events (id, provider, provider_event_id, provider_message_id, event_type, recipient_email, payload_json, occurred_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, id, provider, event.ProviderEventID, event.ProviderMessageID, event.Type, event.RecipientEmail, event.Raw, event.OccurredAt); err != nil {
		return fmt.Errorf("record webhook event: %w", err)
	}
	if event.ProviderMessageID != "" {
		status := webhook.StatusForType(event.Type)
		if status != "" {
			if _, err := tx.ExecContext(ctx, `UPDATE campaign_recipients SET status = ?, error_message = CASE WHEN ? = 'failed' THEN ? ELSE error_message END WHERE provider_message_id = ? AND status IN ('sent', 'delivered', 'opened', 'clicked')`, status, status, event.Type, event.ProviderMessageID); err != nil {
				return fmt.Errorf("update recipient from webhook: %w", err)
			}
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit webhook event: %w", err)
	}
	return nil
}
