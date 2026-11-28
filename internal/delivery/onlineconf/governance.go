package onlineconf

import (
	"context"
	"encoding/json"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/httpresponse"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	oc "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain/onlineconf"
	service "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/services/onlineconf"
	"github.com/gorilla/mux"
	"net/http"
)

type Governance interface {
	GovernanceView(context.Context, oc.AdminIdentity, string) (json.RawMessage, error)
	GovernanceChange(context.Context, oc.AdminIdentity, string, service.GovernanceInput) error
	MayAccess(context.Context, domain.UserID, string, int64, string) (bool, error)
}

func (h *AdminHandler) governancePage(w http.ResponseWriter, r *http.Request) {
	actor, ok := AdminFromContext(r.Context())
	if !ok {
		h.writeError(w, r, domain.ErrUnauthorized)
		return
	}
	name := mux.Vars(r)["section"]
	if r.Method == "GET" {
		v, err := h.gov.GovernanceView(r.Context(), actor, name)
		if err != nil {
			h.writeError(w, r, err)
			return
		}
		httpresponse.WriteJSONContext(r.Context(), w, 200, v, h.appLogger)
		return
	}
	body, err := decodeRequest[service.GovernanceInput](w, r)
	if err == nil {
		err = h.gov.GovernanceChange(r.Context(), actor, name, *body)
	}
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	w.WriteHeader(204)
}
func (h *AdminHandler) resourceAccess(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		actor, ok := AdminFromContext(r.Context())
		if !ok || !oc.HasAdminAccess(actor.Role) {
			h.writeError(w, r, domain.ErrForbidden)
			return
		}
		if actor.Role != "scoped" {
			next.ServeHTTP(w, r)
			return
		}
		if h.gov == nil {
			h.writeError(w, r, domain.ErrForbidden)
			return
		}
		ctx := oc.WithScopedUser(r.Context(), actor.ID)
		r = r.WithContext(ctx)
		route := mux.CurrentRoute(r)
		pattern, _ := route.GetPathTemplate()
		switch pattern {
		case mePath, configsPath, flagsPath, scopesPath, AdminBasePath + "/governance/{section}":
		case configPath, flagPath, evaluatePath, scopePath:
			{
				kind := "flag"
				if pattern == configPath {
					kind = "config"
				} else if pattern == scopePath {
					kind = "scope"
				}
				id, err := requestID(r)
				if err != nil {
					h.writeError(w, r, err)
					return
				}
				action := "read"
				if r.Method == http.MethodPut {
					action = "update"
				} else if r.Method == http.MethodDelete {
					action = "delete"
				}
				ok, err := h.gov.MayAccess(ctx, actor.ID, kind, id, action)
				if err != nil {
					h.writeError(w, r, err)
					return
				}
				if !ok {
					h.writeError(w, r, domain.ErrForbidden)
					return
				}
			}
		case AdminBasePath + "/catalog":
			h.writeError(w, r, domain.ErrForbidden)
			return
		default:
			h.writeError(w, r, domain.ErrForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// Return only a write receipt unless the caller may also read the resource.
func (h *AdminHandler) writeMutation(w http.ResponseWriter, r *http.Request, status int, kind string, id, version int64, result any) {
	if uid := oc.ScopedUser(r.Context()); uid != 0 {
		allowed, err := h.gov.MayAccess(r.Context(), uid, kind, id, "read")
		if err != nil || !allowed {
			result = map[string]int64{"id": id, "version": version}
		}
	}
	httpresponse.WriteJSONContext(r.Context(), w, status, result, h.appLogger)
}
