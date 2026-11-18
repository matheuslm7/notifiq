package auth

import (
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type TokenService struct {
	secret []byte
	ttl    time.Duration
}

type Claims struct {
	UserID         string `json:"user_id"`
	OrganizationID string `json:"organization_id"`
	Role           string `json:"role"`
	jwt.RegisteredClaims
}

func NewTokenService(secret string, ttl time.Duration) (*TokenService, error) {
	if len(strings.TrimSpace(secret)) < 32 {
		return nil, fmt.Errorf("jwt secret must contain at least 32 characters")
	}
	if ttl <= 0 {
		return nil, fmt.Errorf("jwt ttl must be positive")
	}
	return &TokenService{secret: []byte(secret), ttl: ttl}, nil
}

func (s *TokenService) Issue(userID, organizationID string) (string, error) {
	return s.IssueWithRole(userID, organizationID, "owner")
}

func (s *TokenService) IssueWithRole(userID, organizationID, role string) (string, error) {
	if userID == "" || organizationID == "" {
		return "", fmt.Errorf("user id and organization id are required")
	}

	now := time.Now()
	claims := Claims{
		UserID:         userID,
		OrganizationID: organizationID,
		Role:           role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.ttl)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	encoded, err := token.SignedString(s.secret)
	if err != nil {
		return "", fmt.Errorf("sign access token: %w", err)
	}
	return encoded, nil
}

func (s *TokenService) Parse(tokenString string) (*Claims, error) {
	var claims Claims
	token, err := jwt.ParseWithClaims(tokenString, &claims, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method: %s", token.Method.Alg())
		}
		return s.secret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("parse access token: %w", err)
	}
	if !token.Valid {
		return nil, fmt.Errorf("parse access token: token is invalid")
	}
	if claims.UserID == "" || claims.OrganizationID == "" || claims.Role == "" {
		return nil, fmt.Errorf("parse access token: required claims are missing")
	}
	return &claims, nil
}
