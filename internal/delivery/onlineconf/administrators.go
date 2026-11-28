package onlineconf

import (
	"context"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/httpresponse"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	oc "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain/onlineconf"
	"net/http"
	"strings"
)

type AdministratorDirectory interface {
	ListAdministrators(context.Context) ([]oc.AdminIdentity, error)
	AuditAuthors(context.Context) ([]oc.AuditAuthor, error)
	ChangeRole(context.Context, domain.UserID, domain.UserID, domain.Role, domain.Role) (oc.AdminIdentity, error)
	FindAccount(context.Context, string) (oc.AdminIdentity, error)
	GrantAdmin(context.Context, domain.UserID, domain.UserID) (oc.AdminIdentity, error)
}

func (h *AdminHandler) listAdministrators(w http.ResponseWriter, r *http.Request) {
	actor, ok := AdminFromContext(r.Context())
	if !ok || actor.Role != oc.SuperAdmin {
		h.writeError(w, r, domain.ErrForbidden)
		return
	}
	users, err := h.administrators.ListAdministrators(r.Context())
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpresponse.WriteJSONContext(r.Context(), w, http.StatusOK, users, h.appLogger)
}

func (h *AdminHandler) auditAuthors(w http.ResponseWriter, r *http.Request) {
	authors, err := h.administrators.AuditAuthors(r.Context())
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpresponse.WriteJSONContext(r.Context(), w, http.StatusOK, authors, h.appLogger)
}

func (h *AdminHandler) manageAdministrators(w http.ResponseWriter, r *http.Request) {
	actor, ok := AdminFromContext(r.Context())
	if !ok || actor.Role != oc.SuperAdmin {
		h.writeError(w, r, domain.ErrForbidden)
		return
	}
	var result oc.AdminIdentity
	var err error
	if r.Method == http.MethodGet {
		username := strings.TrimSpace(r.URL.Query().Get("username"))
		if username == "" {
			h.writeError(w, r, errInvalidInput)
			return
		}
		result, err = h.administrators.FindAccount(r.Context(), username)
	} else {
		id, idErr := requestID(r)
		if idErr != nil {
			h.writeError(w, r, idErr)
			return
		}
		if r.ContentLength == 0 {
			result, err = h.administrators.GrantAdmin(r.Context(), actor.ID, domain.UserID(id))
		} else {
			body, e := decodeRequest[struct {
				Role     domain.Role `json:"role"`
				Previous domain.Role `json:"previous_role"`
			}](w, r)
			if e != nil {
				h.writeError(w, r, e)
				return
			}
			if body.Previous == "" || !oc.CanAssignRole(body.Role) {
				h.writeError(w, r, errInvalidInput)
				return
			}
			result, err = h.administrators.ChangeRole(r.Context(), actor.ID, domain.UserID(id), body.Role, body.Previous)
		}
	}
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpresponse.WriteJSONContext(r.Context(), w, http.StatusOK, result, h.appLogger)
}
