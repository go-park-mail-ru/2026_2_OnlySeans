package onlineconf

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/httpresponse"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	oc "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain/onlineconf"
)

type snapshotConfigResponse struct {
	Scope string          `json:"scope"`
	Key   string          `json:"key"`
	Value json.RawMessage `json:"value"`
}

type snapshotFlagResponse struct {
	ID              int64           `json:"id"`
	Scope           string          `json:"scope"`
	Key             string          `json:"key"`
	Enabled         bool            `json:"enabled"`
	RolloutPercent  int             `json:"rollout_percent"`
	Salt            string          `json:"salt"`
	ForceOnUserIDs  []domain.UserID `json:"force_on_user_ids"`
	ForceOffUserIDs []domain.UserID `json:"force_off_user_ids"`
}

type snapshotResponse struct {
	Revision    int64                    `json:"revision"`
	GeneratedAt time.Time                `json:"generated_at"`
	Configs     []snapshotConfigResponse `json:"configs"`
	Flags       []snapshotFlagResponse   `json:"flags"`
}

func (h *Handler) snapshot(w http.ResponseWriter, r *http.Request) {
	current, err := h.snapshots.Current()
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	if current == nil {
		h.writeError(w, r, oc.ErrSnapshotNotReady)
		return
	}

	response := snapshotResponse{Revision: current.Revision, GeneratedAt: current.GeneratedAt.UTC(), Configs: make([]snapshotConfigResponse, 0, len(current.Configs)), Flags: make([]snapshotFlagResponse, 0, len(current.Flags))}
	for _, config := range current.Configs {
		response.Configs = append(response.Configs, snapshotConfigResponse{Scope: config.Scope, Key: config.Key, Value: config.Value})
	}

	for _, flag := range current.Flags {
		response.Flags = append(response.Flags, snapshotFlagResponse{ID: flag.ID, Scope: flag.Scope, Key: flag.Key, Enabled: flag.Enabled, RolloutPercent: flag.RolloutPercent, Salt: flag.Salt, ForceOnUserIDs: append([]domain.UserID{}, flag.ForceOnUserIDs...), ForceOffUserIDs: append([]domain.UserID{}, flag.ForceOffUserIDs...)})
	}

	httpresponse.WriteJSONContext(r.Context(), w, http.StatusOK, response, h.appLogger)
}

func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	current, err := h.snapshots.Current()
	if err != nil || current == nil {
		h.writeError(w, r, oc.ErrSnapshotNotReady)
		return
	}

	httpresponse.WriteJSONContext(r.Context(), w, http.StatusOK, struct {
		Status string `json:"status"`
	}{Status: healthyStatus}, h.appLogger)
}
