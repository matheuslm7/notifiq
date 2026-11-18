package emailsettings

import (
	"context"
	"fmt"

	"github.com/matheuslm7/notifiq/internal/security"
)

type Settings struct {
	Provider    string `json:"provider"`
	Host        string `json:"host"`
	Port        int    `json:"port"`
	Username    string `json:"username"`
	Password    string `json:"password,omitempty"`
	FromEmail   string `json:"from_email"`
	FromName    string `json:"from_name"`
	HasPassword bool   `json:"has_password"`
}

type Repository interface {
	GetEmailSettings(ctx context.Context, organizationID string) (Settings, error)
	GetEmailSettingsForDelivery(ctx context.Context, organizationID string) (Settings, error)
	SaveEmailSettings(ctx context.Context, organizationID string, settings Settings) error
}

type SMTPConfig struct {
	Host      string
	Port      int
	Username  string
	Password  string
	FromEmail string
	FromName  string
}

type Service struct {
	repository Repository
	secrets    *security.SecretBox
}

func NewService(repository Repository, secrets *security.SecretBox) (*Service, error) {
	if repository == nil || secrets == nil {
		return nil, fmt.Errorf("repository and secret box are required")
	}
	return &Service{repository: repository, secrets: secrets}, nil
}

func (s *Service) Get(ctx context.Context, organizationID string) (Settings, error) {
	settings, err := s.repository.GetEmailSettings(ctx, organizationID)
	if err != nil {
		return Settings{}, fmt.Errorf("get email settings: %w", err)
	}
	settings.Password = ""
	return settings, nil
}

func (s *Service) Save(ctx context.Context, organizationID string, settings Settings) error {
	if organizationID == "" || settings.Host == "" || settings.Port < 1 || settings.Port > 65535 || settings.Username == "" || settings.FromEmail == "" || settings.FromName == "" {
		return fmt.Errorf("invalid email settings")
	}
	if settings.Provider == "" {
		settings.Provider = "smtp"
	}
	if settings.Provider != "smtp" {
		return fmt.Errorf("unsupported email provider")
	}
	encryptedPassword, err := s.secrets.Encrypt(settings.Password)
	if err != nil {
		return fmt.Errorf("encrypt email credential: %w", err)
	}
	settings.Password = encryptedPassword
	return s.repository.SaveEmailSettings(ctx, organizationID, settings)
}

func (s *Service) ResolveSMTP(ctx context.Context, organizationID string) (SMTPConfig, error) {
	settings, err := s.repository.GetEmailSettingsForDelivery(ctx, organizationID)
	if err != nil {
		return SMTPConfig{}, fmt.Errorf("get delivery settings: %w", err)
	}
	password, err := s.secrets.Decrypt(settings.Password)
	if err != nil {
		return SMTPConfig{}, fmt.Errorf("decrypt delivery credential: %w", err)
	}
	return SMTPConfig{Host: settings.Host, Port: settings.Port, Username: settings.Username, Password: password, FromEmail: settings.FromEmail, FromName: settings.FromName}, nil
}
