package persons

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/delivery/http/httpresponse"
	personsclient "github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/grpc_client/personscleint"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/logger"
)

type FilmReader interface {
	GetFilmJSON(ctx context.Context, filmID int64) (json.RawMessage, error)
}

type CatalogHandler struct {
	personClient personsclient.PersonClient
	films        FilmReader
	appLogger    logger.Logger
}

func NewCatalogHandler(client personsclient.PersonClient, films FilmReader, appLogger logger.Logger) (*CatalogHandler, error) {
	if client == nil {
		return nil, errors.New("person client is nil")
	}

	if appLogger == nil {
		return nil, errors.New("logger is nil")
	}

	return &CatalogHandler{
		personClient: client,
		films:        films,
		appLogger:    appLogger,
	}, nil
}

func (h *CatalogHandler) GetFilmPersons(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writePersonError(r.Context(), w, err, h.appLogger)
		return
	}

	if h.films == nil {
		httpresponse.WriteErrorContext(r.Context(), w, http.StatusServiceUnavailable, "Film catalog is unavailable", h.appLogger)
		return
	}

	if _, err := h.films.GetFilmJSON(r.Context(), id); err != nil {
		writePersonError(r.Context(), w, err, h.appLogger)
		return
	}

	result, err := h.personClient.GetFilmPersons(r.Context(), id)
	if err != nil {
		writePersonError(r.Context(), w, err, h.appLogger)
		return
	}

	people := make([]personCreditDTO, len(result.GetPeople()))
	for i, credit := range result.GetPeople() {
		people[i] = personCreditDTO{
			Person:    toCatalogPersonDTO(credit.GetPerson()),
			Role:      credit.GetRole(),
			Character: credit.GetCharacter(),
		}
	}

	httpresponse.WriteJSONContext(r.Context(), w, http.StatusOK, filmPersonsResponse{People: people}, h.appLogger)
}

func (h *CatalogHandler) GetPersonCatalog(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writePersonError(r.Context(), w, err, h.appLogger)
		return
	}

	result, err := h.personClient.GetPersonCatalog(r.Context(), id)
	if err != nil {
		writePersonError(r.Context(), w, err, h.appLogger)
		return
	}

	films := make([]catalogFilmDTO, 0, len(result.GetFilms()))
	if len(result.GetFilms()) > 0 && h.films == nil {
		httpresponse.WriteErrorContext(r.Context(), w, http.StatusServiceUnavailable, "Film catalog is unavailable", h.appLogger)
		return
	}

	for _, credit := range result.GetFilms() {
		film, err := h.films.GetFilmJSON(r.Context(), credit.GetFilmId())
		if err != nil {
			writePersonError(r.Context(), w, err, h.appLogger)
			return
		}

		films = append(films, catalogFilmDTO{
			Film:      film,
			Role:      credit.GetRole(),
			Character: credit.GetCharacter(),
		})
	}

	response := personCatalogResponse{
		Person: toCatalogPersonDTO(result.GetPerson()),
		Films:  films,
	}

	httpresponse.WriteJSONContext(r.Context(), w, http.StatusOK, response, h.appLogger)
}
