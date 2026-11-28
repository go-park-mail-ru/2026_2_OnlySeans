package onlineconf

import (
	"net/http"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/httpresponse"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	usecase "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/onlineconf"
)

func (h *AdminHandler) listConfigs(w http.ResponseWriter, r *http.Request) {
	options, err := requestList(r)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	result, err := h.configs.List(r.Context(), options)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	httpresponse.WriteJSONContext(r.Context(), w, http.StatusOK, result, h.appLogger)
}

func (h *AdminHandler) getConfig(w http.ResponseWriter, r *http.Request) {
	id, err := requestID(r)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	result, err := h.configs.Get(r.Context(), usecase.GetResourceInput{ID: id})
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	httpresponse.WriteJSONContext(r.Context(), w, http.StatusOK, result, h.appLogger)
}

func (h *AdminHandler) createConfig(w http.ResponseWriter, r *http.Request) {
	actor, ok := AdminFromContext(r.Context())
	if !ok {
		h.writeError(w, r, domain.ErrUnauthorized)
		return
	}

	req, err := decodeRequest[configRequest](w, r)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	fields, err := req.fields(false)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	result, err := h.configs.Create(r.Context(), usecase.CreateConfigInput{ActorID: actor.ID, Fields: fields})
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	httpresponse.WriteJSONContext(r.Context(), w, http.StatusCreated, result, h.appLogger)
}

func (h *AdminHandler) updateConfig(w http.ResponseWriter, r *http.Request) {
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

	req, err := decodeRequest[configRequest](w, r)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	fields, err := req.fields(true)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	result, err := h.configs.Update(r.Context(), usecase.UpdateConfigInput{ActorID: actor.ID, ID: id, Version: *req.Version, Fields: fields})
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	httpresponse.WriteJSONContext(r.Context(), w, http.StatusOK, result, h.appLogger)
}

func (h *AdminHandler) deleteConfig(w http.ResponseWriter, r *http.Request) {
	input, err := deleteInput(r)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	if err := h.configs.Delete(r.Context(), input); err != nil {
		h.writeError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
