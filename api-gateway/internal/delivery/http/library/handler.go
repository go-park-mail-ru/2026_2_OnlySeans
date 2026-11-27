package library

import (
	"context"

	"errors"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/delivery/http/httprequest"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/delivery/http/httpresponse"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/grpc_client/catalogclient"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/middleware"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/logger"
)

const (
	saveFailedMsg   = httpresponse.InternalErrorMessage
	invalidDataMsg  = "invalid request data"
	filmNotFoundMsg = "film not found"
)

type Handler struct {
	client    catalogclient.CatalogClient
	appLogger logger.Logger
}

func NewHandler(client catalogclient.CatalogClient, appLogger logger.Logger) *Handler {
	return &Handler{
		client:    client,
		appLogger: appLogger,
	}
}

func user(r *http.Request) int64 {
	id, _ := middleware.UserIDFromContext(r.Context())
	return id
}

func (h *Handler) fail(ctx context.Context, w http.ResponseWriter, err error) {
	status, message := http.StatusInternalServerError, saveFailedMsg

	switch {
	case errors.Is(err, catalogclient.ErrInvalidLibraryData), errors.Is(err, catalogclient.ErrInvalidPreferences), errors.Is(err, catalogclient.ErrInvalidArgument), errors.Is(err, httprequest.ErrInvalidBody):
		status, message = http.StatusBadRequest, invalidDataMsg
	case errors.Is(err, catalogclient.ErrUnauthorized):
		status, message = http.StatusUnauthorized, catalogclient.ErrUnauthorized.Error()
	case errors.Is(err, catalogclient.ErrFilmNotFound):
		status, message = http.StatusNotFound, filmNotFoundMsg
	default:
		h.appLogger.ErrorContext(ctx, "library request failed", map[string]any{
			"error": err.Error(),
		})
	}

	httpresponse.WriteErrorContext(ctx, w, status, message, h.appLogger)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	data, err := h.client.GetLibrary(r.Context(), user(r))
	if err != nil {
		h.fail(r.Context(), w, err)
		return
	}

	httpresponse.WriteJSONContext(r.Context(), w, http.StatusOK, libraryResponse{
		Favorite:   nonNilIDs(data.GetLibrary().GetFavorite()),
		WatchLater: nonNilIDs(data.GetLibrary().GetWatchlater()),
		Watched:    nonNilIDs(data.GetLibrary().GetWatched()),
	}, h.appLogger)
}

func (h *Handler) SetSaved(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Saved *bool `json:"saved"`
	}

	if err := httprequest.DecodeJSON(w, r, &body); err != nil {
		h.fail(r.Context(), w, err)
		return
	}

	film, err := strconv.ParseInt(mux.Vars(r)["film"], 10, 64)
	if err != nil || body.Saved == nil {
		h.fail(r.Context(), w, catalogclient.ErrInvalidLibraryData)
		return
	}

	if _, err = h.client.SetSaved(r.Context(), user(r), mux.Vars(r)["list"], film, *body.Saved); err != nil {
		h.fail(r.Context(), w, err)
		return
	}

	h.Get(w, r)
}

type libraryResponse struct {
	Favorite   []int64 `json:"favorite"`
	WatchLater []int64 `json:"watchlater"`
	Watched    []int64 `json:"watched"`
}

func nonNilIDs(ids []int64) []int64 {
	if ids == nil {
		return []int64{}
	}

	return ids
}
