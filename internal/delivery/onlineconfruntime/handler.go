package onlineconfruntime

import (
	"errors"
	"maps"
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
	configs := make(map[string]any, len(policy.Configs)+2)

	for key, value := range policy.Configs {
		configs[key] = value
	}

	configs[domain.ReviewMaxLengthKey] = policy.MaxLength
	configs[domain.HomeBlocksKey] = policy.HomeBlocks

	flags := maps.Clone(policy.Flags)
	if flags == nil {
		flags = make(map[string]bool)
	}

	flags[domain.ReviewWriteEnabledKey] = policy.WriteEnabled
	flags[domain.HomeBlocksEnabledKey] = policy.HomeBlocksEnabled

	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Add("Vary", "Cookie")
	httpresponse.WriteJSONContext(r.Context(), w, http.StatusOK, runtimeResponse{
		Revision:      policy.Revision,
		SnapshotReady: policy.SnapshotReady,
		Configs:       configs,
		Flags:         flags,
	}, h.logger)
}

type runtimeResponse struct {
	SnapshotReady bool            `json:"snapshot_ready"`
	Revision      int64           `json:"revision"`
	Configs       map[string]any  `json:"configs"`
	Flags         map[string]bool `json:"flags"`
}
