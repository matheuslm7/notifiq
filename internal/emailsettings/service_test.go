package emailsettings

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"testing"

	"github.com/matheuslm7/notifiq/internal/security"
)

type repositoryStub struct {
	settings Settings
	saved    Settings
}

func (r *repositoryStub) GetEmailSettings(context.Context, string) (Settings, error) {
	return r.settings, nil
}
func (r *repositoryStub) GetEmailSettingsForDelivery(context.Context, string) (Settings, error) {
	return r.settings, nil
}
func (r *repositoryStub) SaveEmailSettings(_ context.Context, _ string, settings Settings) error {
	r.saved = settings
	return nil
}

func TestSaveEncryptsPasswordAndGetNeverReturnsIt(t *testing.T) {
	key := make([]byte, 32)
	_, _ = rand.Read(key)
	box, _ := security.NewSecretBox(base64.RawStdEncoding.EncodeToString(key))
	repository := &repositoryStub{settings: Settings{Host: "smtp.example.com", Port: 587, Username: "user", FromEmail: "from@example.com", FromName: "NotifiQ", HasPassword: true}}
	service, _ := NewService(repository, box)
	settings := Settings{Provider: "smtp", Host: "smtp.example.com", Port: 587, Username: "user", Password: "smtp-password", FromEmail: "from@example.com", FromName: "NotifiQ"}
	if err := service.Save(context.Background(), "org-1", settings); err != nil {
		t.Fatal(err)
	}
	if repository.saved.Password == "smtp-password" || repository.saved.Password == "" {
		t.Fatal("expected encrypted password")
	}
	got, err := service.Get(context.Background(), "org-1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Password != "" {
		t.Fatal("password must never be returned")
	}
}
