package ratings

import (
	"net/http"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/delivery/http/httpresponse"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/grpc_client/catalogclient"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/middleware"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/logger"
)

type MyRatingHandler struct {
	client    catalogclient.CatalogClient
	appLogger logger.Logger
}

type personalScoreDTO struct {
	Score int32 `json:"score"`
}
type personalRatingResponse struct {
	Rating *personalScoreDTO `json:"rating"`
}

func (h *MyRatingHandler) GetPersonalRating(w http.ResponseWriter, r *http.Request) {
	filmID, ok := pathID(r)
	if !ok {
		writeError(r.Context(), w, ErrInvalidFilmID, h.appLogger)
		return
	}

	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(r.Context(), w, catalogclient.ErrUnauthorized, h.appLogger)
		return
	}

	result, err := h.client.GetMyFilmRating(r.Context(), userID, filmID)
	if err != nil {
		writeError(r.Context(), w, err, h.appLogger)
		return
	}

	response := personalRatingResponse{}

	if result.Score != nil {
		response.Rating = &personalScoreDTO{
			Score: *result.Score,
		}
	}

	httpresponse.WriteJSONContext(r.Context(), w, http.StatusOK, response, h.appLogger)
}

func NewMyRatingHandler(client catalogclient.CatalogClient, appLogger logger.Logger) *MyRatingHandler {
	return &MyRatingHandler{
		client:    client,
		appLogger: appLogger,
	}
}
