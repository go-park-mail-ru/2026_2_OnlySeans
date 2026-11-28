package onlineconf

import (
	"net/http"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/httpresponse"
	oc "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain/onlineconf"
)

func (h *AdminHandler) snapshot(w http.ResponseWriter, r *http.Request) {
	current, err := h.snapshots.Current()
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	if current == nil {
		h.writeError(w, r, oc.ErrSnapshotNotReady)
		return
	}

	response := oc.NewSnapshotPayload(current)
	if err := response.ValidateSize(); err != nil {
		h.writeError(w, r, err)
		return
	}

	httpresponse.WriteJSONContext(r.Context(), w, http.StatusOK, response, h.appLogger)
}

func (h *AdminHandler) health(w http.ResponseWriter, r *http.Request) {
	current, err := h.snapshots.Current()
	if err != nil || current == nil {
		h.writeError(w, r, oc.ErrSnapshotNotReady)
		return
	}

	httpresponse.WriteJSONContext(r.Context(), w, http.StatusOK, struct {
		Status string `json:"status"`
	}{Status: healthyStatus}, h.appLogger)
}
