package onlineconfruntime

import (
	"errors"
	"net/http"

	authdelivery "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/auth"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/httpresponse"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
)

type Handler struct {
	policy domain.RuntimeProvider
	logger logger.Logger
}

func NewHandler(policy domain.RuntimeProvider, appLogger logger.Logger) (*Handler, error) {
	if policy == nil {
		return nil, domain.ErrNilReviewPolicy
	}
	if appLogger == nil {
		return nil, errors.New("onlineconf handler logger is nil")
	}
	return &Handler{policy: policy, logger: appLogger}, nil
}

func (h *Handler) Runtime(w http.ResponseWriter, r *http.Request) {
	userID, _ := authdelivery.UserIDFromContext(r.Context())
	policy := h.policy.PublicRuntime(userID)
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Add("Vary", "Cookie")
	httpresponse.WriteJSONContext(r.Context(), w, http.StatusOK, runtimeResponse{
		Revision:      policy.Revision,
		SnapshotReady: policy.SnapshotReady,
		Configs: map[string]any{
			domain.ReviewMaxLengthKey: policy.MaxLength,
			domain.HomeBlocksKey:      policy.HomeBlocks,
		},
		Flags: map[string]bool{
			domain.ReviewWriteEnabledKey: policy.WriteEnabled,
			domain.HomeBlocksEnabledKey:  policy.HomeBlocksEnabled,
		},
	}, h.logger)
}

type runtimeResponse struct {
	SnapshotReady bool            `json:"snapshot_ready"`
	Revision      int64           `json:"revision"`
	Configs       map[string]any  `json:"configs"`
	Flags         map[string]bool `json:"flags"`
}
