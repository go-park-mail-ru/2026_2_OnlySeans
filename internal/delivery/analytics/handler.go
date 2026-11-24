package analytics

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/httpresponse"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	usecase "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/analytics"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
)

const (
	defaultTopGenresLimit = 10
)

type Handler struct {
	topGenres *usecase.TopGenresUseCase
	appLogger logger.Logger
}

func NewHandler(topGenres *usecase.TopGenresUseCase, appLogger logger.Logger) (*Handler, error) {
	if topGenres == nil {
		return nil, errors.New("top genres use case is nil")
	}

	if appLogger == nil {
		return nil, errors.New("logger is nil")
	}

	return &Handler{
		topGenres: topGenres,
		appLogger: appLogger,
	}, nil
}

func parseTopGenresInput(r *http.Request) (usecase.TopGenresInput, error) {
	query := r.URL.Query()

	from, err := time.Parse(time.DateOnly, query.Get("from"))
	if err != nil {
		return usecase.TopGenresInput{}, fmt.Errorf("from must use YYYY-MM-DD format: %w", domain.ErrInvalidAnalyticsPeriod)
	}

	to, err := time.Parse(time.DateOnly, query.Get("to"))
	if err != nil {
		return usecase.TopGenresInput{}, fmt.Errorf("to must use YYYY-MM-DD format: %w", domain.ErrInvalidAnalyticsPeriod)
	}

	limit := defaultTopGenresLimit
	rawLimit := query.Get("limit")
	if rawLimit != "" {
		limit, err = strconv.Atoi(rawLimit)
		if err != nil {
			return usecase.TopGenresInput{}, fmt.Errorf("limit must be an integer: %w", domain.ErrInvalidAnalyticsLimit)
		}
	}

	return usecase.TopGenresInput{
		From:  from,
		To:    to,
		Limit: limit,
	}, nil
}

func writeError(ctx context.Context, w http.ResponseWriter, err error, appLogger logger.Logger) {
	if errors.Is(err, domain.ErrInvalidAnalyticsPeriod) || errors.Is(err, domain.ErrInvalidAnalyticsLimit) {
		httpresponse.WriteErrorContext(ctx, w, http.StatusBadRequest, err.Error(), appLogger)
		return
	}

	httpresponse.WriteErrorContext(ctx, w, http.StatusInternalServerError, httpresponse.InternalErrorMessage, appLogger)
}

func (h *Handler) TopGenres(w http.ResponseWriter, r *http.Request) {
	input, err := parseTopGenresInput(r)

	if err != nil {
		writeError(r.Context(), w, err, h.appLogger)
		return
	}

	stats, err := h.topGenres.Run(r.Context(), input)
	if err != nil {
		writeError(r.Context(), w, err, h.appLogger)
		return
	}

	response := topGenresResponse{
		Genres: toGenreOpenStatResponses(stats),
	}

	httpresponse.WriteJSONContext(r.Context(), w, http.StatusOK, response, h.appLogger)
}
