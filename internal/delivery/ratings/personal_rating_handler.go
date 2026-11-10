package ratings

import (
	"net/http"

	authdelivery "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/auth"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/httpresponse"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	usecase "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/ratings"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
)

type MyRatingHandler struct {
	useCase *usecase.GetMyFilmRatingUseCase
	logger  logger.Logger
}

type personalScoreDTO struct {
	Score int `json:"score"`
}
type personalRatingResponse struct {
	Rating *personalScoreDTO `json:"rating"`
}

func (h *MyRatingHandler) GetPersonalRating(w http.ResponseWriter, r *http.Request) {
	filmID, ok := pathID(r)
	if !ok {
		writeError(r.Context(), w, ErrInvalidFilmID, h.logger)
		return
	}

	userID, ok := authdelivery.UserIDFromContext(r.Context())
	if !ok {
		writeError(r.Context(), w, domain.ErrUnauthorized, h.logger)
		return
	}

	result, err := h.useCase.Run(r.Context(), usecase.GetMyFilmRatingInput{AccountID: userID, FilmID: domain.FilmID(filmID)})
	if err != nil {
		writeError(r.Context(), w, err, h.logger)
		return
	}

	response := personalRatingResponse{}

	if result.Score != nil {
		response.Rating = &personalScoreDTO{Score: *result.Score}
	}

	httpresponse.WriteJSONContext(r.Context(), w, http.StatusOK, response, h.logger)
}

func NewMyRatingHandler(uc *usecase.GetMyFilmRatingUseCase, log logger.Logger) *MyRatingHandler {
	return &MyRatingHandler{useCase: uc, logger: log}
}
