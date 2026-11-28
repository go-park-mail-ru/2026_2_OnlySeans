package onlineconf

import (
	"net/http"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/httpresponse"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	usecase "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/onlineconf"
)

func (h *AdminHandler) me(w http.ResponseWriter, r *http.Request) {
	actor, ok := AdminFromContext(r.Context())
	if !ok {
		h.writeError(w, r, domain.ErrUnauthorized)
		return
	}

	httpresponse.WriteJSONContext(r.Context(), w, http.StatusOK, actor, h.appLogger)
}

func (h *AdminHandler) listScopes(w http.ResponseWriter, r *http.Request) {
	options, err := requestList(r)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	result, err := h.scopes.List(r.Context(), options)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	httpresponse.WriteJSONContext(r.Context(), w, http.StatusOK, result, h.appLogger)
}

func (h *AdminHandler) createScope(w http.ResponseWriter, r *http.Request) {
	actor, ok := AdminFromContext(r.Context())
	if !ok {
		h.writeError(w, r, domain.ErrUnauthorized)
		return
	}

	req, err := decodeRequest[scopeRequest](w, r)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	if req.Name == nil || req.Title == nil || req.Version != nil {
		h.writeError(w, r, errInvalidInput)
		return
	}

	result, err := h.scopes.Create(r.Context(), usecase.CreateScopeInput{ActorID: actor.ID, Fields: usecase.ScopeFields{Name: *req.Name, Title: *req.Title, Description: optionalString(req.Description)}})
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	httpresponse.WriteJSONContext(r.Context(), w, http.StatusCreated, result, h.appLogger)
}

func (h *AdminHandler) getScope(w http.ResponseWriter, r *http.Request) {
	id, err := requestID(r)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	result, err := h.scopes.Get(r.Context(), usecase.GetResourceInput{ID: id})
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	httpresponse.WriteJSONContext(r.Context(), w, http.StatusOK, result, h.appLogger)
}

func (h *AdminHandler) updateScope(w http.ResponseWriter, r *http.Request) {
	actor, ok := AdminFromContext(r.Context())
	if !ok {
		h.writeError(w, r, domain.ErrUnauthorized)
		return
	}

	id, err := requestID(r)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	req, err := decodeRequest[scopeRequest](w, r)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	if req.Name == nil || req.Title == nil || req.Description == nil || req.Version == nil {
		h.writeError(w, r, errInvalidInput)
		return
	}

	result, err := h.scopes.Update(r.Context(), usecase.UpdateScopeInput{ActorID: actor.ID, ID: id, Version: *req.Version, Fields: usecase.ScopeFields{Name: *req.Name, Title: *req.Title, Description: *req.Description}})
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	httpresponse.WriteJSONContext(r.Context(), w, http.StatusOK, result, h.appLogger)
}

func (h *AdminHandler) deleteScope(w http.ResponseWriter, r *http.Request) {
	input, err := deleteInput(r)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	if err := h.scopes.Delete(r.Context(), input); err != nil {
		h.writeError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
