package auth

import (
	"context"
	"testing"
	"time"

	"github.com/pquerna/otp/totp"
)

type userRepositoryStub struct {
	created User
	stored  string
	user    User
	err     error
	rotated bool
}

func (r *userRepositoryStub) CreateUser(_ context.Context, user User) (User, error) {
	r.created = user
	if user.ID == "" {
		user.ID = "user-1"
	}
	if user.OrganizationID == "" {
		user.OrganizationID = "org-1"
	}
	return user, r.err
}

func (r *userRepositoryStub) FindByEmail(_ context.Context, _ string) (User, error) {
	return r.user, r.err
}

func (r *userRepositoryStub) FindByID(_ context.Context, _ string) (User, error) {
	return r.user, r.err
}

func (r *userRepositoryStub) SaveRefreshToken(_ context.Context, _, tokenHash string, _ time.Time) error {
	r.stored = tokenHash
	return r.err
}

func (r *userRepositoryStub) RotateRefreshToken(_ context.Context, _, nextHash string, _ time.Time) (User, error) {
	r.rotated = true
	r.stored = nextHash
	return r.user, r.err
}

func (r *userRepositoryStub) RevokeRefreshToken(_ context.Context, _ string) error {
	r.rotated = true
	return r.err
}

func (r *userRepositoryStub) SaveMFASecret(_ context.Context, _ string, secret string) error {
	r.user.MFASecret = secret
	return r.err
}

func (r *userRepositoryStub) EnableMFA(_ context.Context, _ string) error {
	r.user.MFAEnabled = true
	return r.err
}

func (r *userRepositoryStub) SavePasswordResetToken(context.Context, string, string, time.Time) error {
	return r.err
}
func (r *userRepositoryStub) ConsumePasswordResetToken(context.Context, string) (string, error) {
	return r.user.ID, r.err
}
func (r *userRepositoryStub) UpdatePassword(_ context.Context, _ string, passwordHash string) error {
	r.user.PasswordHash = passwordHash
	return r.err
}

func TestRegisterNormalizesUserAndCreatesSession(t *testing.T) {
	repository := &userRepositoryStub{}
	tokens, err := NewTokenService(testSecret, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	service, err := NewService(repository, tokens)
	if err != nil {
		t.Fatal(err)
	}

	session, err := service.Register(context.Background(), " Ana ", " ANA@EXAMPLE.COM ", "strong-pass")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if repository.created.Name != "Ana" || repository.created.Email != "ana@example.com" {
		t.Fatalf("unexpected user: %+v", repository.created)
	}
	if repository.created.PasswordHash == "strong-pass" || repository.stored == "" {
		t.Fatal("expected hashed credentials")
	}
	if session.AccessToken == "" || session.RefreshToken == "" {
		t.Fatal("expected access and refresh tokens")
	}
}

func TestLoginRejectsInvalidCredentials(t *testing.T) {
	hash, err := HashPassword("strong-pass")
	if err != nil {
		t.Fatal(err)
	}
	repository := &userRepositoryStub{user: User{ID: "user-1", OrganizationID: "org-1", PasswordHash: hash}}
	tokens, _ := NewTokenService(testSecret, time.Minute)
	service, _ := NewService(repository, tokens)

	if _, err := service.Login(context.Background(), "user@example.com", "wrong-pass", ""); err != ErrInvalidCredentials {
		t.Fatalf("expected invalid credentials, got %v", err)
	}
}

func TestRegisterValidatesInput(t *testing.T) {
	repository := &userRepositoryStub{}
	tokens, _ := NewTokenService(testSecret, time.Minute)
	service, _ := NewService(repository, tokens)

	if _, err := service.Register(context.Background(), "", "user@example.com", "strong-pass"); err == nil {
		t.Fatal("expected missing name to be rejected")
	}
	if _, err := service.Register(context.Background(), "Ana", "user@example.com", "short"); err == nil {
		t.Fatal("expected short password to be rejected")
	}
}

func TestRefreshRotatesToken(t *testing.T) {
	repository := &userRepositoryStub{user: User{ID: "user-1", OrganizationID: "org-1"}}
	tokens, _ := NewTokenService(testSecret, time.Minute)
	service, _ := NewService(repository, tokens)

	session, err := service.Refresh(context.Background(), "old-refresh-token")
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if !repository.rotated || session.RefreshToken == "old-refresh-token" {
		t.Fatal("expected refresh token rotation")
	}
}

func TestRefreshRejectsEmptyToken(t *testing.T) {
	repository := &userRepositoryStub{}
	tokens, _ := NewTokenService(testSecret, time.Minute)
	service, _ := NewService(repository, tokens)

	if _, err := service.Refresh(context.Background(), ""); err != ErrInvalidRefreshToken {
		t.Fatalf("expected invalid refresh token, got %v", err)
	}
}

func TestMFASetupAndConfirmation(t *testing.T) {
	repository := &userRepositoryStub{user: User{ID: "user-1", OrganizationID: "org-1"}}
	tokens, _ := NewTokenService(testSecret, time.Minute)
	service, _ := NewService(repository, tokens)

	setup, err := service.SetupMFA(context.Background(), "user-1", "ana@example.com")
	if err != nil {
		t.Fatalf("setup mfa: %v", err)
	}
	code, err := totp.GenerateCode(setup.Secret, time.Now())
	if err != nil {
		t.Fatalf("generate mfa code: %v", err)
	}
	if err := service.ConfirmMFA(context.Background(), "user-1", code); err != nil {
		t.Fatalf("confirm mfa: %v", err)
	}
	if !repository.user.MFAEnabled || repository.user.MFASecret == "" {
		t.Fatal("expected mfa to be enabled with a stored secret")
	}
}
