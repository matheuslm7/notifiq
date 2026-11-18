package contacts

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"net/mail"
	"strings"
	"time"
)

type Contact struct {
	ID             string    `json:"id"`
	OrganizationID string    `json:"organization_id"`
	Name           string    `json:"name"`
	Email          string    `json:"email"`
	CreatedAt      time.Time `json:"created_at"`
}

type Repository interface {
	ListContacts(ctx context.Context, organizationID string) ([]Contact, error)
	CreateContact(ctx context.Context, contact Contact) (Contact, error)
}

type BulkRepository interface {
	CreateContacts(ctx context.Context, contacts []Contact) (int, error)
}

type ImportResult struct {
	Imported int      `json:"imported"`
	Skipped  int      `json:"skipped"`
	Errors   []string `json:"errors,omitempty"`
}

type Service struct{ repository Repository }

func NewService(repository Repository) (*Service, error) {
	if repository == nil {
		return nil, fmt.Errorf("contacts repository is required")
	}
	return &Service{repository: repository}, nil
}

func (s *Service) List(ctx context.Context, organizationID string) ([]Contact, error) {
	if organizationID == "" {
		return nil, fmt.Errorf("organization id is required")
	}
	return s.repository.ListContacts(ctx, organizationID)
}

func (s *Service) Create(ctx context.Context, organizationID, name, email string) (Contact, error) {
	name = strings.TrimSpace(name)
	email = strings.ToLower(strings.TrimSpace(email))
	if organizationID == "" || name == "" {
		return Contact{}, fmt.Errorf("organization and name are required")
	}
	parsed, err := mail.ParseAddress(email)
	if err != nil || parsed.Address != email {
		return Contact{}, fmt.Errorf("valid email is required")
	}
	return s.repository.CreateContact(ctx, Contact{OrganizationID: organizationID, Name: name, Email: email})
}

func (s *Service) Import(ctx context.Context, organizationID string, input io.Reader) (ImportResult, error) {
	if organizationID == "" || input == nil {
		return ImportResult{}, fmt.Errorf("organization and csv are required")
	}
	bulk, ok := s.repository.(BulkRepository)
	if !ok {
		return ImportResult{}, fmt.Errorf("bulk contact repository is unavailable")
	}
	reader := csv.NewReader(io.LimitReader(input, 5<<20))
	reader.FieldsPerRecord = -1
	rows, err := reader.ReadAll()
	if err != nil {
		return ImportResult{}, fmt.Errorf("read csv: %w", err)
	}
	if len(rows) < 2 {
		return ImportResult{}, fmt.Errorf("csv must contain a header and at least one row")
	}
	columns := make(map[string]int)
	for index, column := range rows[0] {
		columns[strings.ToLower(strings.TrimSpace(column))] = index
	}
	nameIndex, nameOK := columns["name"]
	emailIndex, emailOK := columns["email"]
	if !nameOK || !emailOK {
		return ImportResult{}, fmt.Errorf("csv must contain name and email columns")
	}
	result := ImportResult{Errors: make([]string, 0)}
	contacts := make([]Contact, 0, len(rows)-1)
	seen := make(map[string]struct{})
	for rowIndex, row := range rows[1:] {
		line := rowIndex + 2
		if nameIndex >= len(row) || emailIndex >= len(row) {
			result.Skipped++
			result.Errors = append(result.Errors, fmt.Sprintf("line %d: missing columns", line))
			continue
		}
		name := strings.TrimSpace(row[nameIndex])
		email := strings.ToLower(strings.TrimSpace(row[emailIndex]))
		if name == "" || !validEmail(email) {
			result.Skipped++
			result.Errors = append(result.Errors, fmt.Sprintf("line %d: invalid name or email", line))
			continue
		}
		if _, exists := seen[email]; exists {
			result.Skipped++
			continue
		}
		seen[email] = struct{}{}
		contacts = append(contacts, Contact{OrganizationID: organizationID, Name: name, Email: email})
	}
	if len(contacts) > 0 {
		imported, err := bulk.CreateContacts(ctx, contacts)
		if err != nil {
			return ImportResult{}, fmt.Errorf("bulk create contacts: %w", err)
		}
		result.Imported = imported
		result.Skipped += len(contacts) - imported
	}
	return result, nil
}

func validEmail(email string) bool {
	parsed, err := mail.ParseAddress(email)
	return err == nil && parsed.Address == email
}
