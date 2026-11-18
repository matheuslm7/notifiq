package members

import (
	"context"
	"testing"
)

type repositoryStub struct {
	role      string
	removedID string
}

func (r *repositoryStub) ListMembers(context.Context, string) ([]Member, error) { return nil, nil }
func (r *repositoryStub) UpdateMemberRole(_ context.Context, _, _, role string) error {
	r.role = role
	return nil
}
func (r *repositoryStub) RemoveMember(_ context.Context, _, memberID string) error {
	r.removedID = memberID
	return nil
}

func TestAdminCannotPromoteMemberToOwner(t *testing.T) {
	service, _ := NewService(&repositoryStub{})
	if err := service.UpdateRole(context.Background(), "org-1", "admin", "user-2", "owner"); err != ErrOwnerRequired {
		t.Fatalf("expected owner restriction, got %v", err)
	}
}

func TestOwnerCanChangeRole(t *testing.T) {
	repository := &repositoryStub{}
	service, _ := NewService(repository)
	if err := service.UpdateRole(context.Background(), "org-1", "owner", "user-2", "editor"); err != nil {
		t.Fatal(err)
	}
	if repository.role != "editor" {
		t.Fatalf("expected editor role, got %q", repository.role)
	}
}

func TestOnlyOwnerCanRemoveMember(t *testing.T) {
	repository := &repositoryStub{}
	service, _ := NewService(repository)
	if err := service.Remove(context.Background(), "org-1", "admin", "user-2"); err != ErrOwnerRequired {
		t.Fatalf("expected owner restriction, got %v", err)
	}
	if err := service.Remove(context.Background(), "org-1", "owner", "user-2"); err != nil {
		t.Fatal(err)
	}
	if repository.removedID != "user-2" {
		t.Fatalf("expected removed user-2, got %q", repository.removedID)
	}
}
