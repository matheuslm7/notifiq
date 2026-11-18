package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/matheuslm7/notifiq/internal/audit"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

var ErrUserNotFound = errors.New("user not found")
var ErrInvalidCredentials = errors.New("invalid credentials")
var ErrMFARequired = errors.New("mfa required")
var ErrInvalidMFACode = errors.New("invalid mfa code")
var ErrInvalidPasswordResetToken = errors.New("invalid password reset token")
var ErrInvalidCurrentPassword = errors.New("invalid current password")

type User struct {
	ID             string
	Email          string
	Name           string
	PasswordHash   string
	OrganizationID string
	Role           string
	MFAEnabled     bool
	MFASecret      string
}

type UserRepository interface {
	CreateUser(ctx context.Context, user User) (User, error)
	FindByEmail(ctx context.Context, email string) (User, error)
	FindByID(ctx context.Context, id string) (User, error)
	SaveRefreshToken(ctx context.Context, userID, tokenHash string, expiresAt time.Time) error
	RotateRefreshToken(ctx context.Context, currentHash, nextHash string, expiresAt time.Time) (User, error)
	RevokeRefreshToken(ctx context.Context, tokenHash string) error
	SaveMFASecret(ctx context.Context, userID, secret string) error
	EnableMFA(ctx context.Context, userID string) error
	SavePasswordResetToken(ctx context.Context, userID, tokenHash string, expiresAt time.Time) error
	ConsumePasswordResetToken(ctx context.Context, tokenHash string) (string, error)
	UpdatePassword(ctx context.Context, userID, passwordHash string) error
}

type PasswordResetNotifier interface {
	SendPasswordReset(ctx context.Context, user User, token string) error
}

type MFAValidator interface {
	VerifyMFA(ctx context.Context, userID, code string) error
}

type Service struct {
	users    UserRepository
	tokens   *TokenService
	clock    func() time.Time
	notifier PasswordResetNotifier
	audit    audit.Recorder
}

func (s *Service) SetPasswordResetNotifier(notifier PasswordResetNotifier) {
	s.notifier = notifier
}

func (s *Service) SetAuditRecorder(recorder audit.Recorder) { s.audit = recorder }

type Session struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

type MFASetup struct {
	Secret string `json:"secret"`
	URL    string `json:"otpauth_url"`
}

func NewService(users UserRepository, tokens *TokenService) (*Service, error) {
	if users == nil || tokens == nil {
		return nil, fmt.Errorf("users repository and token service are required")
	}
	return &Service{users: users, tokens: tokens, clock: time.Now}, nil
}

func (s *Service) Register(ctx context.Context, name, email, password string) (Session, error) {
	name = strings.TrimSpace(name)
	email = strings.ToLower(strings.TrimSpace(email))
	if name == "" || email == "" {
		return Session{}, fmt.Errorf("name and email are required")
	}
	if len(password) < 8 {
		return Session{}, fmt.Errorf("password must contain at least 8 characters")
	}

	passwordHash, err := HashPassword(password)
	if err != nil {
		return Session{}, err
	}
	user := User{
		Email:        email,
		Name:         name,
		PasswordHash: passwordHash,
	}
	user, err = s.users.CreateUser(ctx, user)
	if err != nil {
		return Session{}, fmt.Errorf("create user: %w", err)
	}
	session, err := s.createSession(ctx, user)
	if err == nil && s.audit != nil {
		_ = s.audit.Record(ctx, audit.Entry{OrganizationID: user.OrganizationID, UserID: user.ID, Action: "auth.login", Resource: "session"})
	}
	return session, err
}

func (s *Service) Login(ctx context.Context, email, password, mfaCode string) (Session, error) {
	user, err := s.users.FindByEmail(ctx, strings.ToLower(strings.TrimSpace(email)))
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return Session{}, ErrInvalidCredentials
		}
		return Session{}, fmt.Errorf("find user: %w", err)
	}
	if ComparePassword(user.PasswordHash, password) != nil {
		return Session{}, ErrInvalidCredentials
	}
	if user.MFAEnabled {
		if mfaCode == "" {
			return Session{}, ErrMFARequired
		}
		valid, err := totp.ValidateCustom(mfaCode, user.MFASecret, s.clock(), totp.ValidateOpts{Period: 30, Skew: 1, Digits: otp.DigitsSix, Algorithm: otp.AlgorithmSHA1})
		if err != nil || !valid {
			return Session{}, ErrInvalidMFACode
		}
	}
	return s.createSession(ctx, user)
}

func (s *Service) SetupMFA(ctx context.Context, userID, accountName string) (MFASetup, error) {
	if userID == "" {
		return MFASetup{}, fmt.Errorf("user id is required")
	}
	key, err := totp.Generate(totp.GenerateOpts{Issuer: "NotifiQ", AccountName: accountName})
	if err != nil {
		return MFASetup{}, fmt.Errorf("generate mfa secret: %w", err)
	}
	if err := s.users.SaveMFASecret(ctx, userID, key.Secret()); err != nil {
		return MFASetup{}, fmt.Errorf("save mfa secret: %w", err)
	}
	return MFASetup{Secret: key.Secret(), URL: key.URL()}, nil
}

func (s *Service) ConfirmMFA(ctx context.Context, userID, code string) error {
	if userID == "" || code == "" {
		return ErrInvalidMFACode
	}
	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("find user for mfa: %w", err)
	}
	if err := s.verifyMFA(user.MFASecret, code); err != nil {
		return err
	}
	if err := s.users.EnableMFA(ctx, userID); err != nil {
		return fmt.Errorf("enable mfa: %w", err)
	}
	return nil
}

func (s *Service) VerifyMFA(ctx context.Context, userID, code string) error {
	if userID == "" || code == "" {
		return ErrInvalidMFACode
	}
	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("find user for mfa: %w", err)
	}
	return s.verifyMFA(user.MFASecret, code)
}

func (s *Service) verifyMFA(secret, code string) error {
	valid, err := totp.ValidateCustom(code, secret, s.clock(), totp.ValidateOpts{Period: 30, Skew: 1, Digits: otp.DigitsSix, Algorithm: otp.AlgorithmSHA1})
	if err != nil || !valid {
		return ErrInvalidMFACode
	}
	return nil
}

func (s *Service) Refresh(ctx context.Context, refreshToken string) (Session, error) {
	if strings.TrimSpace(refreshToken) == "" {
		return Session{}, ErrInvalidRefreshToken
	}
	nextToken, err := GenerateRefreshToken()
	if err != nil {
		return Session{}, err
	}
	expiresAt := s.clock().Add(30 * 24 * time.Hour)
	user, err := s.users.RotateRefreshToken(ctx, HashRefreshToken(refreshToken), HashRefreshToken(nextToken), expiresAt)
	if err != nil {
		if errors.Is(err, ErrInvalidRefreshToken) {
			return Session{}, ErrInvalidRefreshToken
		}
		return Session{}, fmt.Errorf("rotate refresh token: %w", err)
	}
	accessToken, err := s.tokens.IssueWithRole(user.ID, user.OrganizationID, user.Role)
	if err != nil {
		return Session{}, err
	}
	return Session{AccessToken: accessToken, RefreshToken: nextToken, ExpiresAt: expiresAt}, nil
}

func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	if strings.TrimSpace(refreshToken) == "" {
		return ErrInvalidRefreshToken
	}
	if err := s.users.RevokeRefreshToken(ctx, HashRefreshToken(refreshToken)); err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}
	return nil
}

func (s *Service) ChangePassword(ctx context.Context, userID, currentPassword, newPassword string) error {
	if userID == "" || len(newPassword) < 8 {
		return fmt.Errorf("new password must contain at least 8 characters")
	}
	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("find user: %w", err)
	}
	if ComparePassword(user.PasswordHash, currentPassword) != nil {
		return ErrInvalidCurrentPassword
	}
	hash, err := HashPassword(newPassword)
	if err != nil {
		return err
	}
	if err := s.users.UpdatePassword(ctx, userID, hash); err != nil {
		return err
	}
	if s.audit != nil {
		_ = s.audit.Record(ctx, audit.Entry{OrganizationID: user.OrganizationID, UserID: userID, Action: "auth.password_changed", Resource: "user", ResourceID: userID})
	}
	return nil
}

func (s *Service) RequestPasswordReset(ctx context.Context, email string) error {
	user, err := s.users.FindByEmail(ctx, strings.ToLower(strings.TrimSpace(email)))
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil
		}
		return fmt.Errorf("find user for password reset: %w", err)
	}
	token, err := GenerateRefreshToken()
	if err != nil {
		return err
	}
	if err := s.users.SavePasswordResetToken(ctx, user.ID, HashRefreshToken(token), s.clock().Add(30*time.Minute)); err != nil {
		return fmt.Errorf("save password reset token: %w", err)
	}
	if s.notifier != nil {
		if err := s.notifier.SendPasswordReset(ctx, user, token); err != nil {
			return fmt.Errorf("send password reset: %w", err)
		}
	}
	return nil
}

func (s *Service) ResetPassword(ctx context.Context, token, newPassword string) error {
	if strings.TrimSpace(token) == "" || len(newPassword) < 8 {
		return ErrInvalidPasswordResetToken
	}
	hash, err := HashPassword(newPassword)
	if err != nil {
		return err
	}
	userID, err := s.users.ConsumePasswordResetToken(ctx, HashRefreshToken(token))
	if err != nil {
		return ErrInvalidPasswordResetToken
	}
	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("find reset user: %w", err)
	}
	if err := s.users.UpdatePassword(ctx, userID, hash); err != nil {
		return fmt.Errorf("update reset password: %w", err)
	}
	if s.audit != nil {
		_ = s.audit.Record(ctx, audit.Entry{OrganizationID: user.OrganizationID, UserID: userID, Action: "auth.password_reset", Resource: "user", ResourceID: userID})
	}
	return nil
}

func (s *Service) createSession(ctx context.Context, user User) (Session, error) {
	accessToken, err := s.tokens.IssueWithRole(user.ID, user.OrganizationID, user.Role)
	if err != nil {
		return Session{}, err
	}
	refreshToken, err := GenerateRefreshToken()
	if err != nil {
		return Session{}, err
	}
	expiresAt := s.clock().Add(30 * 24 * time.Hour)
	if err := s.users.SaveRefreshToken(ctx, user.ID, HashRefreshToken(refreshToken), expiresAt); err != nil {
		return Session{}, fmt.Errorf("save refresh token: %w", err)
	}
	return Session{AccessToken: accessToken, RefreshToken: refreshToken, ExpiresAt: expiresAt}, nil
}
