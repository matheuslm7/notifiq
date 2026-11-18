package members

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/matheuslm7/notifiq/internal/auth"
)

var ErrInvalidInvite = errors.New("invalid or expired invitation")

type Invite struct {
	OrganizationID string
	InvitedBy      string
	Email          string
	Role           string
	Token          string
	ExpiresAt      time.Time
}

type InviteRepository interface {
	CreateInvite(context.Context, Invite, string) error
	AcceptInvite(context.Context, string, string, string, string) (Member, error)
}

type InviteNotifier interface {
	SendInvite(context.Context, Invite) error
}

func CreateInvite(ctx context.Context, repository InviteRepository, notifier InviteNotifier, organizationID, invitedBy, email, role string) error {
	email = strings.ToLower(strings.TrimSpace(email))
	role = strings.ToLower(strings.TrimSpace(role))
	if organizationID == "" || invitedBy == "" || email == "" || !inviteRole(role) {
		return fmt.Errorf("email and valid role are required")
	}
	token, err := auth.GenerateRefreshToken()
	if err != nil {
		return err
	}
	invite := Invite{OrganizationID: organizationID, InvitedBy: invitedBy, Email: email, Role: role, Token: token, ExpiresAt: time.Now().UTC().Add(72 * time.Hour)}
	if err := repository.CreateInvite(ctx, invite, auth.HashRefreshToken(token)); err != nil {
		return fmt.Errorf("create invite: %w", err)
	}
	if notifier != nil {
		if err := notifier.SendInvite(ctx, invite); err != nil {
			return fmt.Errorf("send invite: %w", err)
		}
	}
	return nil
}

func AcceptInvite(ctx context.Context, repository InviteRepository, token, name, password string) (Member, error) {
	name = strings.TrimSpace(name)
	if token == "" || name == "" || len(password) < 8 {
		return Member{}, ErrInvalidInvite
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return Member{}, err
	}
	member, err := repository.AcceptInvite(ctx, auth.HashRefreshToken(token), name, hash, strings.TrimSpace(token))
	if err != nil {
		return Member{}, ErrInvalidInvite
	}
	return member, nil
}

func inviteRole(role string) bool {
	return role == "admin" || role == "editor" || role == "viewer"
}
