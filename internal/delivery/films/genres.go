package films

import (
	"errors"
	"net/http"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/httpresponse"
	usecase "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/films"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
)

type GenresHandler struct {
	listGenres *usecase.ListGenresUseCase
	appLogger  logger.Logger
}

func NewGenresHandler(listGenres *usecase.ListGenresUseCase, appLogger logger.Logger) (*GenresHandler, error) {
	if listGenres == nil {
		return nil, errors.New("list genres use case is nil")
	}

	if appLogger == nil {
		return nil, errors.New("logger is nil")
	}

	return &GenresHandler{
		listGenres: listGenres,
		appLogger:  appLogger,
	}, nil
}

type genresResponse struct {
	Genres []genreResponseDTO `json:"genres"`
}

func (h *GenresHandler) ListGenres(w http.ResponseWriter, r *http.Request) {
	genres, err := h.listGenres.Run(r.Context())

	if err != nil {
		writeError(r.Context(), w, err, h.appLogger)
		return
	}

	response := genresResponse{Genres: toGenreResponseDTOs(genres)}
	httpresponse.WriteJSONContext(r.Context(), w, http.StatusOK, response, h.appLogger)

}
