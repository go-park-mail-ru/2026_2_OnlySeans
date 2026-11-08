package films

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/httpresponse"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	persons "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/services/persons"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
	"github.com/gorilla/mux"
)

type filmLookup interface {
	GetFilmByID(context.Context, domain.FilmID) (domain.Film, error)
}

func RegisterPeople(repo *persons.CatalogRepo, films filmLookup, log logger.Logger) func(*mux.Router) {
	fail := func(w http.ResponseWriter, err error) {
		status := 500
		if errors.Is(err, domain.ErrFilmNotFound) || errors.Is(err, domain.ErrPersonNotFound) {
			status = 404
		}
		if status == 500 {
			log.Error("people catalog failed", map[string]any{"error": err})
		}
		httpresponse.WriteError(w, status, "Не удалось загрузить участников фильма.", log)
	}
	return func(r *mux.Router) {
		r.HandleFunc("/api/films/{id}/people", func(w http.ResponseWriter, r *http.Request) {
			id, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
			if err != nil || id < 1 {
				httpresponse.WriteError(w, 400, "invalid id", log)
				return
			}
			if _, err = films.GetFilmByID(r.Context(), domain.FilmID(id)); err != nil {
				fail(w, err)
				return
			}
			people, err := repo.FilmPeople(r.Context(), id)
			if err != nil {
				fail(w, err)
				return
			}
			httpresponse.WriteJSON(w, 200, map[string]any{"people": people}, log)
		}).Methods(http.MethodGet)
		r.HandleFunc("/api/people/{id}", func(w http.ResponseWriter, r *http.Request) {
			id, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
			if err != nil || id < 1 {
				httpresponse.WriteError(w, 400, "invalid id", log)
				return
			}
			person, credits, err := repo.Get(r.Context(), id)
			if err != nil {
				fail(w, err)
				return
			}
			items := []map[string]any{}
			for _, credit := range credits {
				film, err := films.GetFilmByID(r.Context(), credit.FilmID)
				if err != nil {
					fail(w, err)
					return
				}
				items = append(items, map[string]any{"film": toFilmResponseDTO(film), "role": credit.Role, "character": credit.Character})
			}
			httpresponse.WriteJSON(w, 200, map[string]any{"person": person, "films": items}, log)
		}).Methods(http.MethodGet)
	}
}
