package delivery

import (
	"context"
	"time"
)

type EmailJob struct {
	OrganizationID string
	CampaignID     string
	ContactID      string
	To             string
	ToName         string
	From           string
	FromName       string
	Subject        string
	BodyHTML       string
	MessageID      string
}

type Sender interface {
	Send(ctx context.Context, job EmailJob) (string, error)
}

type Repository interface {
	ClaimRecipient(ctx context.Context, campaignID, contactID string) (bool, int, error)
	MarkRecipientSent(ctx context.Context, campaignID, contactID, messageID string) error
	MarkRecipientFailed(ctx context.Context, campaignID, contactID, message string) error
	MarkRecipientRetry(ctx context.Context, campaignID, contactID, message string, nextAttemptAt time.Time) error
}
