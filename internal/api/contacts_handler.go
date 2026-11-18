package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/matheuslm7/notifiq/internal/auth"
	"github.com/matheuslm7/notifiq/internal/contacts"
)

type ContactsService interface {
	List(ctx context.Context, organizationID string) ([]contacts.Contact, error)
	Create(ctx context.Context, organizationID, name, email string) (contacts.Contact, error)
}

type ContactsHandler struct{ Service ContactsService }
type contactRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type ContactsImporter interface {
	Import(context.Context, string, io.Reader) (contacts.ImportResult, error)
}

func (h *ContactsHandler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authorization required")
		return
	}
	items, err := h.Service.List(r.Context(), claims.OrganizationID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load contacts")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(items)
}

func (h *ContactsHandler) Create(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authorization required")
		return
	}
	var request contactRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	contact, err := h.Service.Create(r.Context(), claims.OrganizationID, request.Name, request.Email)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(contact)
}

func (h *ContactsHandler) Import(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authorization required")
		return
	}
	if err := r.ParseMultipartForm(5 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "invalid multipart form")
		return
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "csv file is required")
		return
	}
	defer file.Close()
	service, ok := h.Service.(ContactsImporter)
	if !ok {
		writeError(w, http.StatusInternalServerError, "contact importer unavailable")
		return
	}
	result, err := service.Import(r.Context(), claims.OrganizationID, io.LimitReader(file, 5<<20))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}
