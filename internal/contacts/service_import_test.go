package contacts

import (
	"context"
	"strings"
	"testing"
)

type bulkRepositoryStub struct{ contacts []Contact }

func (r *bulkRepositoryStub) ListContacts(context.Context, string) ([]Contact, error) {
	return nil, nil
}
func (r *bulkRepositoryStub) CreateContact(context.Context, Contact) (Contact, error) {
	return Contact{}, nil
}
func (r *bulkRepositoryStub) CreateContacts(_ context.Context, contacts []Contact) (int, error) {
	r.contacts = contacts
	return len(contacts), nil
}

func TestImportCSVNormalizesAndSkipsInvalidRows(t *testing.T) {
	repository := &bulkRepositoryStub{}
	service, _ := NewService(repository)
	result, err := service.Import(context.Background(), "org-1", strings.NewReader("name,email\nAna, ANA@EXAMPLE.COM \nInvalid,nope\nAna 2,ana@example.com\n"))
	if err != nil {
		t.Fatal(err)
	}
	if result.Imported != 1 || result.Skipped != 2 || len(repository.contacts) != 1 || repository.contacts[0].Email != "ana@example.com" {
		t.Fatalf("unexpected import result: %+v contacts=%+v", result, repository.contacts)
	}
}

func TestImportCSVRequiresColumns(t *testing.T) {
	service, _ := NewService(&bulkRepositoryStub{})
	if _, err := service.Import(context.Background(), "org-1", strings.NewReader("full_name,address\nAna,ana@example.com")); err == nil {
		t.Fatal("expected missing columns error")
	}
}
