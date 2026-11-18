package members

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

var ErrMemberNotFound = errors.New("member not found")
var ErrLastOwner = errors.New("organization must have at least one owner")
var ErrOwnerRequired = errors.New("only an owner can manage another owner")

type Member struct {
	UserID         string    `json:"user_id"`
	OrganizationID string    `json:"organization_id"`
	Name           string    `json:"name"`
	Email          string    `json:"email"`
	Role           string    `json:"role"`
	CreatedAt      time.Time `json:"created_at"`
}

type Repository interface {
	ListMembers(context.Context, string) ([]Member, error)
	UpdateMemberRole(context.Context, string, string, string) error
	RemoveMember(context.Context, string, string) error
}

type Service struct{ repository Repository }

func NewService(repository Repository) (*Service, error) {
	if repository == nil {
		return nil, fmt.Errorf("members repository is required")
	}
	return &Service{repository: repository}, nil
}

func (s *Service) List(ctx context.Context, organizationID string) ([]Member, error) {
	if organizationID == "" {
		return nil, fmt.Errorf("organization id is required")
	}
	return s.repository.ListMembers(ctx, organizationID)
}

func (s *Service) UpdateRole(ctx context.Context, organizationID, actorRole, memberID, role string) error {
	role = strings.ToLower(strings.TrimSpace(role))
	if organizationID == "" || memberID == "" || !validRole(role) {
		return fmt.Errorf("organization, member and valid role are required")
	}
	if actorRole != "owner" && role == "owner" {
		return ErrOwnerRequired
	}
	return s.repository.UpdateMemberRole(ctx, organizationID, memberID, role)
}

func (s *Service) Remove(ctx context.Context, organizationID, actorRole, memberID string) error {
	if organizationID == "" || memberID == "" {
		return fmt.Errorf("organization and member are required")
	}
	if actorRole != "owner" {
		return ErrOwnerRequired
	}
	return s.repository.RemoveMember(ctx, organizationID, memberID)
}

func validRole(role string) bool {
	switch role {
	case "owner", "admin", "editor", "viewer":
		return true
	default:
		return false
	}
}
