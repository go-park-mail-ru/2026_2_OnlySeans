package analytics

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/delivery/http/httpresponse"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/grpc_client/catalogclient"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/logger"
)

const defaultTopGenresLimit = 10

type Handler struct {
	client    catalogclient.CatalogClient
	appLogger logger.Logger
}

func NewHandler(client catalogclient.CatalogClient, appLogger logger.Logger) *Handler {
	return &Handler{client: client, appLogger: appLogger}
}

type genreOpenStatResponse struct {
	GenreID   int64  `json:"genre_id"`
	Name      string `json:"name"`
	Slug      string `json:"slug"`
	OpenCount int64  `json:"open_count"`
}

type topGenresResponse struct {
	Genres []genreOpenStatResponse `json:"genres"`
}

type ageGroupOpenStatResponse struct {
	AgeGroup       string  `json:"age_group"`
	GenreOpenCount int64   `json:"genre_open_count"`
	TotalOpenCount int64   `json:"total_open_count"`
	Share          float64 `json:"share"`
}

type genreAgeResponse struct {
	GenreSlug string                     `json:"genre_slug"`
	AgeGroups []ageGroupOpenStatResponse `json:"age_groups"`
}

func (h *Handler) TopGenres(w http.ResponseWriter, r *http.Request) {
	from, to, err := parsePeriod(r)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	limit := defaultTopGenresLimit
	if raw := r.URL.Query().Get("limit"); raw != "" {
		limit, err = strconv.Atoi(raw)
		if err != nil {
			h.writeError(w, r, fmt.Errorf("limit must be an integer: %w", catalogclient.ErrInvalidArgument))
			return
		}
	}

	result, err := h.client.TopGenres(r.Context(), from, to, limit)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	genres := make([]genreOpenStatResponse, len(result.GetGenres()))
	for i, stat := range result.GetGenres() {
		genres[i] = genreOpenStatResponse{GenreID: stat.GetGenreId(), Name: stat.GetName(), Slug: stat.GetSlug(), OpenCount: stat.GetOpenCount()}
	}

	httpresponse.WriteJSONContext(r.Context(), w, http.StatusOK, topGenresResponse{Genres: genres}, h.appLogger)
}

func (h *Handler) GenreAge(w http.ResponseWriter, r *http.Request) {
	from, to, err := parsePeriod(r)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	genreSlug := mux.Vars(r)["slug"]
	result, err := h.client.GenreAge(r.Context(), genreSlug, from, to)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	groups := make([]ageGroupOpenStatResponse, len(result.GetAgeGroups()))
	for i, stat := range result.GetAgeGroups() {
		groups[i] = ageGroupOpenStatResponse{
			AgeGroup:       stat.GetAgeGroup(),
			GenreOpenCount: stat.GetGenreOpenCount(),
			TotalOpenCount: stat.GetTotalOpenCount(),
			Share:          stat.GetShare(),
		}
	}

	httpresponse.WriteJSONContext(r.Context(), w, http.StatusOK, genreAgeResponse{GenreSlug: result.GetGenreSlug(), AgeGroups: groups}, h.appLogger)
}

func parsePeriod(r *http.Request) (time.Time, time.Time, error) {
	from, err := time.Parse(time.DateOnly, r.URL.Query().Get("from"))
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("from must use YYYY-MM-DD format: %w", catalogclient.ErrInvalidArgument)
	}

	to, err := time.Parse(time.DateOnly, r.URL.Query().Get("to"))
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("to must use YYYY-MM-DD format: %w", catalogclient.ErrInvalidArgument)
	}

	return from, to, nil
}

func (h *Handler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, catalogclient.ErrInvalidArgument) {
		httpresponse.WriteErrorContext(r.Context(), w, http.StatusBadRequest, err.Error(), h.appLogger)
		return
	}

	httpresponse.WriteErrorContext(r.Context(), w, http.StatusInternalServerError, httpresponse.InternalErrorMessage, h.appLogger)
}
