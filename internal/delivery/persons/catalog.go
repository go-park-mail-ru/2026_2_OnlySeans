package persons

import (
	"context"

	"errors"
	"net/http"
	"strconv"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/filmdto"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/httpresponse"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	personusecase "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/persons"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
	"github.com/gorilla/mux"
)

type CatalogHandler struct {
	getFilmPersons *personusecase.GetFilmPersonsUseCase
	getPerson      *personusecase.GetPersonCatalogUseCase
	log            logger.Logger
}

func NewCatalogHandler(getFilmPersons *personusecase.GetFilmPersonsUseCase, getPerson *personusecase.GetPersonCatalogUseCase, log logger.Logger) *CatalogHandler {
	return &CatalogHandler{getFilmPersons: getFilmPersons, getPerson: getPerson, log: log}
}

func (h *CatalogHandler) fail(ctx context.Context, w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	if errors.Is(err, domain.ErrFilmNotFound) || errors.Is(err, domain.ErrPersonNotFound) {
		status = http.StatusNotFound
	}
	if status == http.StatusInternalServerError {
		h.log.ErrorContext(ctx, "persons catalog failed", map[string]any{"error": err})
	}
	httpresponse.WriteErrorContext(ctx, w, status, "Не удалось загрузить участников фильма.", h.log)
}

func (h *CatalogHandler) GetFilmPersons(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil || id < 1 {
		httpresponse.WriteErrorContext(r.Context(), w, http.StatusBadRequest, "invalid id", h.log)
		return
	}
	persons, err := h.getFilmPersons.Run(r.Context(), domain.FilmID(id))
	if err != nil {
		h.fail(r.Context(), w, err)
		return
	}
	httpresponse.WriteJSONContext(r.Context(), w, http.StatusOK, map[string]any{"people": persons}, h.log)
}

func (h *CatalogHandler) GetPersonCatalog(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil || id < 1 {
		httpresponse.WriteErrorContext(r.Context(), w, http.StatusBadRequest, "invalid id", h.log)
		return
	}
	result, err := h.getPerson.Run(r.Context(), domain.PersonID(id))
	if err != nil {
		h.fail(r.Context(), w, err)
		return
	}
	items := []map[string]any{}
	for _, credit := range result.Films {
		items = append(items, map[string]any{"film": filmdto.FromFilm(credit.Film), "role": credit.Role, "character": credit.Character})
	}
	httpresponse.WriteJSONContext(r.Context(), w, http.StatusOK, map[string]any{"person": result.Person, "films": items}, h.log)
}
