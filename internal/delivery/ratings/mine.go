package ratings

import (
	authdelivery "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/auth"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/httpresponse"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	usecase "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/ratings"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
	"github.com/gorilla/mux"
	"net/http"
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

func (h *MyRatingHandler) Get(w http.ResponseWriter, r *http.Request) {
	filmID, ok := pathID(r)
	if !ok {
		writeError(w, ErrInvalidFilmID, h.logger)
		return
	}
	userID, ok := authdelivery.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, domain.ErrUnauthorized, h.logger)
		return
	}
	result, err := h.useCase.Run(r.Context(), usecase.GetMyFilmRatingInput{AccountID: userID, FilmID: domain.FilmID(filmID)})
	if err != nil {
		writeError(w, err, h.logger)
		return
	}
	response := personalRatingResponse{}
	if result.Score != nil {
		response.Rating = &personalScoreDTO{Score: *result.Score}
	}
	httpresponse.WriteJSON(w, http.StatusOK, response, h.logger)
}

func RegisterMine(uc *usecase.GetMyFilmRatingUseCase, log logger.Logger, protect func(http.HandlerFunc) http.HandlerFunc) func(*mux.Router) {
	h := &MyRatingHandler{useCase: uc, logger: log}
	return func(r *mux.Router) {
		r.HandleFunc("/api/films/{id}/rating/me", protect(h.Get)).Methods(http.MethodGet)
	}
}
