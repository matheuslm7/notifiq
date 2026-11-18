package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/matheuslm7/notifiq/internal/audit"
	"github.com/matheuslm7/notifiq/internal/auth"
	"github.com/matheuslm7/notifiq/internal/members"
)

type MembersService interface {
	List(context.Context, string) ([]members.Member, error)
	UpdateRole(context.Context, string, string, string, string) error
	Remove(context.Context, string, string, string) error
}

type MembersHandler struct {
	Service        MembersService
	Audit          audit.Recorder
	Invites        members.InviteRepository
	InviteNotifier members.InviteNotifier
}

type memberRoleRequest struct {
	MemberID string `json:"member_id"`
	Role     string `json:"role"`
}

type removeMemberRequest struct {
	MemberID string `json:"member_id"`
}

type inviteRequest struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

type acceptInviteRequest struct {
	Token    string `json:"token"`
	Name     string `json:"name"`
	Password string `json:"password"`
}

func (h *MembersHandler) Members(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authorization required")
		return
	}
	switch r.Method {
	case http.MethodGet:
		items, err := h.Service.List(r.Context(), claims.OrganizationID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "could not load members")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(items)
	case http.MethodPut:
		var request memberRoleRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		if err := h.Service.UpdateRole(r.Context(), claims.OrganizationID, claims.Role, request.MemberID, request.Role); err != nil {
			writeMemberError(w, err)
			return
		}
		if h.Audit != nil {
			_ = h.Audit.Record(r.Context(), audit.Entry{OrganizationID: claims.OrganizationID, UserID: claims.UserID, Action: "member.role_updated", Resource: "organization_member", ResourceID: request.MemberID})
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *MembersHandler) Invite(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authorization required")
		return
	}
	var request inviteRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if err := members.CreateInvite(r.Context(), h.Invites, h.InviteNotifier, claims.OrganizationID, claims.UserID, request.Email, request.Role); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *MembersHandler) AcceptInvite(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var request acceptInviteRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	member, err := members.AcceptInvite(r.Context(), h.Invites, request.Token, request.Name, request.Password)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid or expired invitation")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(member)
}

func (h *MembersHandler) Remove(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authorization required")
		return
	}
	var request removeMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if err := h.Service.Remove(r.Context(), claims.OrganizationID, claims.Role, request.MemberID); err != nil {
		writeMemberError(w, err)
		return
	}
	if h.Audit != nil {
		_ = h.Audit.Record(r.Context(), audit.Entry{OrganizationID: claims.OrganizationID, UserID: claims.UserID, Action: "member.removed", Resource: "organization_member", ResourceID: request.MemberID})
	}
	w.WriteHeader(http.StatusNoContent)
}

func writeMemberError(w http.ResponseWriter, err error) {
	status := http.StatusBadRequest
	if errors.Is(err, members.ErrOwnerRequired) || errors.Is(err, members.ErrLastOwner) {
		status = http.StatusForbidden
	}
	writeError(w, status, err.Error())
}
