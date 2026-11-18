package campaigns

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

var ErrCampaignNotEditable = errors.New("campaign cannot be changed after sending starts")

type Template struct {
	ID             string    `json:"id"`
	OrganizationID string    `json:"organization_id"`
	Name           string    `json:"name"`
	Subject        string    `json:"subject"`
	BodyHTML       string    `json:"body_html"`
	CreatedAt      time.Time `json:"created_at"`
}

type Campaign struct {
	ID             string     `json:"id"`
	OrganizationID string     `json:"organization_id"`
	TemplateID     string     `json:"template_id"`
	Name           string     `json:"name"`
	Status         string     `json:"status"`
	ScheduledAt    *time.Time `json:"scheduled_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}

type Repository interface {
	ListTemplates(ctx context.Context, organizationID string) ([]Template, error)
	CreateTemplate(ctx context.Context, template Template) (Template, error)
	DeleteTemplate(ctx context.Context, organizationID, templateID string) error
	ListCampaigns(ctx context.Context, organizationID string) ([]Campaign, error)
	CreateCampaign(ctx context.Context, campaign Campaign) (Campaign, error)
	DeleteCampaign(ctx context.Context, organizationID, campaignID string) error
	AddRecipients(ctx context.Context, organizationID, campaignID string, contactIDs []string) error
}

type Service struct{ repository Repository }

type ActionsRepository interface {
	UpdateCampaign(context.Context, string, string, string, *time.Time, string) error
	CancelCampaign(context.Context, string, string) error
	DuplicateCampaign(context.Context, string, string) (Campaign, error)
}

func NewService(repository Repository) (*Service, error) {
	if repository == nil {
		return nil, fmt.Errorf("campaigns repository is required")
	}
	return &Service{repository: repository}, nil
}

func (s *Service) ListTemplates(ctx context.Context, organizationID string) ([]Template, error) {
	if organizationID == "" {
		return nil, fmt.Errorf("organization id is required")
	}
	return s.repository.ListTemplates(ctx, organizationID)
}

func (s *Service) CreateTemplate(ctx context.Context, organizationID, name, subject, bodyHTML string) (Template, error) {
	name, subject, bodyHTML = strings.TrimSpace(name), strings.TrimSpace(subject), strings.TrimSpace(bodyHTML)
	if organizationID == "" || name == "" || subject == "" || bodyHTML == "" {
		return Template{}, fmt.Errorf("name, subject and body are required")
	}
	return s.repository.CreateTemplate(ctx, Template{OrganizationID: organizationID, Name: name, Subject: subject, BodyHTML: bodyHTML})
}

func (s *Service) DeleteTemplate(ctx context.Context, organizationID, templateID string) error {
	if organizationID == "" || templateID == "" {
		return fmt.Errorf("organization and template are required")
	}
	return s.repository.DeleteTemplate(ctx, organizationID, templateID)
}

func (s *Service) ListCampaigns(ctx context.Context, organizationID string) ([]Campaign, error) {
	if organizationID == "" {
		return nil, fmt.Errorf("organization id is required")
	}
	return s.repository.ListCampaigns(ctx, organizationID)
}

func (s *Service) CreateCampaign(ctx context.Context, organizationID, templateID, name string, scheduledAt *time.Time) (Campaign, error) {
	name = strings.TrimSpace(name)
	if organizationID == "" || templateID == "" || name == "" {
		return Campaign{}, fmt.Errorf("organization, template and name are required")
	}
	status := "draft"
	if scheduledAt != nil {
		if scheduledAt.Before(time.Now()) {
			return Campaign{}, fmt.Errorf("scheduled time must be in the future")
		}
		status = "scheduled"
	}
	return s.repository.CreateCampaign(ctx, Campaign{OrganizationID: organizationID, TemplateID: templateID, Name: name, Status: status, ScheduledAt: scheduledAt})
}

func (s *Service) DeleteCampaign(ctx context.Context, organizationID, campaignID string) error {
	if organizationID == "" || campaignID == "" {
		return fmt.Errorf("organization and campaign are required")
	}
	return s.repository.DeleteCampaign(ctx, organizationID, campaignID)
}

func (s *Service) AddRecipients(ctx context.Context, organizationID, campaignID string, contactIDs []string) error {
	if organizationID == "" || campaignID == "" || len(contactIDs) == 0 {
		return fmt.Errorf("organization, campaign and contacts are required")
	}
	return s.repository.AddRecipients(ctx, organizationID, campaignID, contactIDs)
}

func (s *Service) UpdateCampaign(ctx context.Context, organizationID, campaignID, templateID, name string, scheduledAt *time.Time) error {
	name = strings.TrimSpace(name)
	if organizationID == "" || campaignID == "" || templateID == "" || name == "" {
		return fmt.Errorf("organization, campaign, template and name are required")
	}
	if scheduledAt != nil && scheduledAt.Before(time.Now()) {
		return fmt.Errorf("scheduled time must be in the future")
	}
	repository, ok := s.repository.(ActionsRepository)
	if !ok {
		return fmt.Errorf("campaign actions repository is unavailable")
	}
	return repository.UpdateCampaign(ctx, organizationID, campaignID, templateID, scheduledAt, name)
}

func (s *Service) CancelCampaign(ctx context.Context, organizationID, campaignID string) error {
	if organizationID == "" || campaignID == "" {
		return fmt.Errorf("organization and campaign are required")
	}
	repository, ok := s.repository.(ActionsRepository)
	if !ok {
		return fmt.Errorf("campaign actions repository is unavailable")
	}
	return repository.CancelCampaign(ctx, organizationID, campaignID)
}

func (s *Service) DuplicateCampaign(ctx context.Context, organizationID, campaignID string) (Campaign, error) {
	if organizationID == "" || campaignID == "" {
		return Campaign{}, fmt.Errorf("organization and campaign are required")
	}
	repository, ok := s.repository.(ActionsRepository)
	if !ok {
		return Campaign{}, fmt.Errorf("campaign actions repository is unavailable")
	}
	return repository.DuplicateCampaign(ctx, organizationID, campaignID)
}
