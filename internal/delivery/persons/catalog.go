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

func RegisterCatalog(getFilmPersons *personusecase.GetFilmPersonsUseCase, getPerson *personusecase.GetPersonCatalogUseCase, log logger.Logger) func(*mux.Router) {
	fail := func(ctx context.Context, w http.ResponseWriter, err error) {
		status := 500
		if errors.Is(err, domain.ErrFilmNotFound) || errors.Is(err, domain.ErrPersonNotFound) {
			status = 404
		}
		if status == 500 {
			log.ErrorContext(ctx, "persons catalog failed", map[string]any{"error": err})
		}
		httpresponse.WriteErrorContext(ctx, w, status, "Не удалось загрузить участников фильма.", log)
	}
	return func(r *mux.Router) {
		r.HandleFunc("/api/films/{id}/people", func(w http.ResponseWriter, r *http.Request) {
			id, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
			if err != nil || id < 1 {
				httpresponse.WriteErrorContext(r.Context(), w, 400, "invalid id", log)
				return
			}
			persons, err := getFilmPersons.Run(r.Context(), domain.FilmID(id))
			if err != nil {
				fail(r.Context(), w, err)
				return
			}
			httpresponse.WriteJSONContext(r.Context(), w, 200, map[string]any{"people": persons}, log)
		}).Methods(http.MethodGet)
		r.HandleFunc("/api/people/{id}", func(w http.ResponseWriter, r *http.Request) {
			id, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
			if err != nil || id < 1 {
				httpresponse.WriteErrorContext(r.Context(), w, 400, "invalid id", log)
				return
			}
			result, err := getPerson.Run(r.Context(), domain.PersonID(id))
			if err != nil {
				fail(r.Context(), w, err)
				return
			}
			items := []map[string]any{}
			for _, credit := range result.Films {
				items = append(items, map[string]any{"film": filmdto.FromFilm(credit.Film), "role": credit.Role, "character": credit.Character})
			}
			httpresponse.WriteJSONContext(r.Context(), w, 200, map[string]any{"person": result.Person, "films": items}, log)
		}).Methods(http.MethodGet)
	}
}
