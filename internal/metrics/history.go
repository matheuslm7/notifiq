package metrics

import "time"

type DeliveryEvent struct {
	ID                string     `json:"id"`
	ProviderEventID   string     `json:"provider_event_id"`
	ProviderMessageID string     `json:"provider_message_id,omitempty"`
	Type              string     `json:"type"`
	OccurredAt        *time.Time `json:"occurred_at,omitempty"`
}

type DeliveryHistory struct {
	CampaignID  string          `json:"campaign_id"`
	ContactID   string          `json:"contact_id"`
	ContactName string          `json:"contact_name"`
	Email       string          `json:"email"`
	Status      string          `json:"status"`
	Attempts    int             `json:"attempts"`
	SentAt      *time.Time      `json:"sent_at,omitempty"`
	Events      []DeliveryEvent `json:"events"`
}
