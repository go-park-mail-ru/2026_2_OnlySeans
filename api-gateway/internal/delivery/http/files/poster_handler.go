package files

import (
	"context"
	"io"

	"errors"
	"net/http"
	"strconv"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/delivery/http/httpresponse"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/grpc_client/catalogclient"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/middleware"

	"github.com/gorilla/mux"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/logger"
)

const posterFieldName = "poster"
const multipartOverhead = 1 << 20

type uploadPosterResponse struct {
	Filename string `json:"filename"`
}

type PosterHandler struct {
	client    catalogclient.CatalogClient
	maxSize   int64
	appLogger logger.Logger
}

func NewPosterHandler(client catalogclient.CatalogClient, maxSize int64, appLogger logger.Logger) (*PosterHandler, error) {
	if client == nil || appLogger == nil {
		return nil, errors.New("catalog client and logger are required")
	}

	if maxSize <= 0 || maxSize > int64(int(^uint(0)>>1))-multipartOverhead {
		return nil, errors.New("poster size must be positive and leave room for multipart overhead")
	}

	return &PosterHandler{
		client:    client,
		maxSize:   maxSize,
		appLogger: appLogger,
	}, nil
}

func (h *PosterHandler) UploadPoster(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok || userID <= 0 {
		h.appLogger.WarnContext(r.Context(), "poster request rejected: missing or invalid user context", nil)
		httpresponse.WriteErrorContext(r.Context(), w, http.StatusUnauthorized, catalogclient.ErrUnauthorized.Error(), h.appLogger)

		return
	}

	filmID, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil || filmID <= 0 {
		h.appLogger.WarnContext(r.Context(), "poster request rejected: invalid film ID", map[string]any{
			"user_id": userID,
		})
		httpresponse.WriteErrorContext(r.Context(), w, http.StatusBadRequest, catalogclient.ErrInvalidPosterFilmID.Error(), h.appLogger)

		return
	}

	fields := map[string]any{
		"user_id": userID,
		"film_id": filmID,
	}
	maxRequestSize := h.maxSize + multipartOverhead
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestSize)
	err = r.ParseMultipartForm(maxRequestSize)

	if r.MultipartForm != nil {
		defer func() {
			if err := r.MultipartForm.RemoveAll(); err != nil {
				h.appLogger.WarnContext(r.Context(), "remove poster multipart temporary files failed", map[string]any{
					"user_id": userID,
					"film_id": filmID,
					"error":   err,
				})
			}
		}()
	}

	if err != nil {
		fields["error"] = err
		h.appLogger.WarnContext(r.Context(), "invalid poster multipart request", fields)
		var sizeErr *http.MaxBytesError

		if errors.As(err, &sizeErr) {
			httpresponse.WriteErrorContext(r.Context(), w, http.StatusRequestEntityTooLarge, "upload request is too large", h.appLogger)
		} else {
			httpresponse.WriteErrorContext(r.Context(), w, http.StatusBadRequest, "invalid multipart request", h.appLogger)
		}

		return
	}

	if len(r.MultipartForm.File[posterFieldName]) != 1 {
		h.appLogger.WarnContext(r.Context(), "poster request rejected: expected one file", fields)
		httpresponse.WriteErrorContext(r.Context(), w, http.StatusBadRequest, "exactly one poster file is required", h.appLogger)

		return
	}

	file, header, err := r.FormFile(posterFieldName)
	if err != nil {
		fields["error"] = err
		h.appLogger.WarnContext(r.Context(), "open poster multipart file failed", fields)
		httpresponse.WriteErrorContext(r.Context(), w, http.StatusBadRequest, catalogclient.ErrPosterMissing.Error(), h.appLogger)

		return
	}

	defer func() {
		if err := file.Close(); err != nil {
			h.appLogger.WarnContext(r.Context(), "close poster upload file failed", map[string]any{
				"user_id": userID,
				"film_id": filmID,
				"error":   err,
			})
		}
	}()

	if header.Size <= 0 {
		h.writePosterError(r.Context(), w, catalogclient.ErrInvalidPosterSize)
		return
	}

	if header.Size > h.maxSize {
		h.writePosterError(r.Context(), w, catalogclient.ErrPosterTooLarge)
		return
	}

	data, err := io.ReadAll(io.LimitReader(file, h.maxSize+1))
	if err != nil {
		h.writePosterError(r.Context(), w, err)
		return
	}

	if int64(len(data)) > h.maxSize {
		h.writePosterError(r.Context(), w, catalogclient.ErrPosterTooLarge)
		return
	}

	result, err := h.client.UploadPoster(r.Context(), userID, filmID, data)
	if err != nil {
		h.writePosterError(r.Context(), w, err)
		return
	}

	httpresponse.WriteJSONContext(r.Context(), w, http.StatusCreated, uploadPosterResponse{
		Filename: result.Filename,
	}, h.appLogger)
}

func (h *PosterHandler) writePosterError(ctx context.Context, w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, catalogclient.ErrUnauthorized):
		httpresponse.WriteErrorContext(ctx, w, http.StatusUnauthorized, catalogclient.ErrUnauthorized.Error(), h.appLogger)
	case errors.Is(err, catalogclient.ErrFilmNotFound), errors.Is(err, catalogclient.ErrFileNotFound):
		httpresponse.WriteErrorContext(ctx, w, http.StatusNotFound, err.Error(), h.appLogger)
	case errors.Is(err, catalogclient.ErrPosterTooLarge):
		httpresponse.WriteErrorContext(ctx, w, http.StatusRequestEntityTooLarge, catalogclient.ErrPosterTooLarge.Error(), h.appLogger)
	case errors.Is(err, catalogclient.ErrInvalidPosterFilmID), errors.Is(err, catalogclient.ErrPosterMissing),
		errors.Is(err, catalogclient.ErrInvalidPosterSize), errors.Is(err, catalogclient.ErrInvalidPosterImage),
		errors.Is(err, catalogclient.ErrPosterDimensionsTooLarge):
		httpresponse.WriteErrorContext(ctx, w, http.StatusBadRequest, err.Error(), h.appLogger)
	default:
		httpresponse.WriteErrorContext(ctx, w, http.StatusInternalServerError, httpresponse.InternalErrorMessage, h.appLogger)
	}
}

func (h *PosterHandler) GetPoster(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil {
		h.writePosterError(r.Context(), w, catalogclient.ErrInvalidPosterFilmID)
		return
	}

	result, err := h.client.GetPosterURL(r.Context(), id)
	if err != nil {
		h.writePosterError(r.Context(), w, err)
		return
	}

	httpresponse.WriteJSONContext(r.Context(), w, http.StatusOK, struct {
		URL string `json:"url"`
	}{
		URL: result.GetUrl(),
	}, h.appLogger)
}

func (h *PosterHandler) DeletePoster(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil {
		h.writePosterError(r.Context(), w, catalogclient.ErrInvalidPosterFilmID)
		return
	}

	if _, err := h.client.DeletePoster(r.Context(), id); err != nil {
		h.writePosterError(r.Context(), w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
