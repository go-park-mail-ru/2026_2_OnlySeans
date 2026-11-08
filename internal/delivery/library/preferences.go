package library

import (
	"encoding/json"
	"net/http"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/httprequest"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/httpresponse"
)

func (h *Handler) Preferences(w http.ResponseWriter, r *http.Request) {
	data, err := h.repo.Preferences(r.Context(), user(r))
	if err != nil {
		h.fail(w, err)
		return
	}
	httpresponse.WriteJSON(w, http.StatusOK, data, h.log)
}

func (h *Handler) SavePreference(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Key   string          `json:"key"`
		Value json.RawMessage `json:"value"`
	}
	if err := httprequest.DecodeJSON(w, r, &body); err != nil {
		h.fail(w, err)
		return
	}
	if err := h.repo.SavePreference(r.Context(), user(r), body.Key, body.Value); err != nil {
		h.fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
