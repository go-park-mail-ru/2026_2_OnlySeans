package onlineconf

import (
	"net/http"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/httpresponse"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	usecase "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/onlineconf"
)

func (h *Handler) listFlags(w http.ResponseWriter, r *http.Request) {
	options, err := requestList(r)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	result, err := h.flags.List(r.Context(), options)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	public := usecase.ListOutput[flagResponse]{Items: make([]flagResponse, 0, len(result.Items)), Total: result.Total}
	for i := range result.Items {
		public.Items = append(public.Items, publicFlag(&result.Items[i]))
	}

	httpresponse.WriteJSONContext(r.Context(), w, http.StatusOK, public, h.appLogger)
}

func (h *Handler) getFlag(w http.ResponseWriter, r *http.Request) {
	id, err := requestID(r)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	result, err := h.flags.Get(r.Context(), usecase.GetResourceInput{ID: id})
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	httpresponse.WriteJSONContext(r.Context(), w, http.StatusOK, publicFlag(result), h.appLogger)
}

func (h *Handler) createFlag(w http.ResponseWriter, r *http.Request) {
	actor, ok := AdminFromContext(r.Context())
	if !ok {
		h.writeError(w, r, domain.ErrUnauthorized)
		return
	}

	req, err := decodeRequest[flagRequest](w, r)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	fields, err := req.fields(false)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	result, err := h.flags.Create(r.Context(), usecase.CreateFlagInput{ActorID: actor.ID, Fields: fields})
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	httpresponse.WriteJSONContext(r.Context(), w, http.StatusCreated, publicFlag(result), h.appLogger)
}

func (h *Handler) updateFlag(w http.ResponseWriter, r *http.Request) {
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

	req, err := decodeRequest[flagRequest](w, r)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	fields, err := req.fields(true)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	result, err := h.flags.Update(r.Context(), usecase.UpdateFlagInput{ActorID: actor.ID, ID: id, Version: *req.Version, Fields: fields})
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	httpresponse.WriteJSONContext(r.Context(), w, http.StatusOK, publicFlag(result), h.appLogger)
}

func (h *Handler) deleteFlag(w http.ResponseWriter, r *http.Request) {
	input, err := deleteInput(r)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	if err := h.flags.Delete(r.Context(), input); err != nil {
		h.writeError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) evaluateFlag(w http.ResponseWriter, r *http.Request) {
	id, err := requestID(r)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	req, err := decodeRequest[evaluateRequest](w, r)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	if req.UserID == nil {
		h.writeError(w, r, errInvalidInput)
		return
	}

	result, err := h.evaluate.Run(r.Context(), usecase.EvaluateFlagInput{FlagID: id, UserID: *req.UserID})
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	httpresponse.WriteJSONContext(r.Context(), w, http.StatusOK, result, h.appLogger)
}
