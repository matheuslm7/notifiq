package contacts

import (
	"context"
	"testing"
)

type repositoryStub struct {
	created  Contact
	contacts []Contact
}

func (r *repositoryStub) ListContacts(context.Context, string) ([]Contact, error) {
	return r.contacts, nil
}
func (r *repositoryStub) CreateContact(_ context.Context, contact Contact) (Contact, error) {
	r.created = contact
	contact.ID = "contact-1"
	return contact, nil
}

func TestCreateNormalizesContact(t *testing.T) {
	repository := &repositoryStub{}
	service, _ := NewService(repository)
	contact, err := service.Create(context.Background(), "org-1", " Ana ", " ANA@EXAMPLE.COM ")
	if err != nil {
		t.Fatal(err)
	}
	if contact.Name != "Ana" || contact.Email != "ana@example.com" {
		t.Fatalf("unexpected contact: %+v", contact)
	}
}

func TestCreateRejectsInvalidEmail(t *testing.T) {
	service, _ := NewService(&repositoryStub{})
	if _, err := service.Create(context.Background(), "org-1", "Ana", "invalid"); err == nil {
		t.Fatal("expected invalid email rejection")
	}
}
