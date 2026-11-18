package members

import (
	"context"
	"testing"
)

type inviteRepositoryStub struct{ invite Invite }

func (r *inviteRepositoryStub) CreateInvite(_ context.Context, invite Invite, _ string) error {
	r.invite = invite
	return nil
}
func (r *inviteRepositoryStub) AcceptInvite(context.Context, string, string, string, string) (Member, error) {
	return Member{Email: "ana@example.com"}, nil
}

func TestCreateInviteNormalizesEmailAndRole(t *testing.T) {
	repository := &inviteRepositoryStub{}
	if err := CreateInvite(context.Background(), repository, nil, "org-1", "user-1", " ANA@EXAMPLE.COM ", "EDITOR"); err != nil {
		t.Fatal(err)
	}
	if repository.invite.Email != "ana@example.com" || repository.invite.Role != "editor" || repository.invite.Token == "" {
		t.Fatalf("unexpected invite: %+v", repository.invite)
	}
}

func TestCreateInviteRejectsOwnerRole(t *testing.T) {
	if err := CreateInvite(context.Background(), &inviteRepositoryStub{}, nil, "org-1", "user-1", "ana@example.com", "owner"); err == nil {
		t.Fatal("expected owner role to be rejected")
	}
}
