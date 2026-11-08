package files

import (
	"errors"
	"net/http"
	"strconv"

	authdelivery "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/auth"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/httpresponse"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	fileusecase "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/files"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
	"github.com/gorilla/mux"
)

const posterFieldName = "poster"

type uploadPosterResponse struct {
	Filename string `json:"filename"`
}

type PosterHandler struct {
	uploadPoster *fileusecase.UploadPosterUseCase
	logger       logger.Logger
}

func NewPosterHandler(uploadPoster *fileusecase.UploadPosterUseCase, appLogger logger.Logger) (*PosterHandler, error) {
	if uploadPoster == nil {
		return nil, errors.New("upload poster use case is nil")
	}
	if appLogger == nil {
		return nil, domain.ErrInvalidAppLogger
	}
	return &PosterHandler{uploadPoster: uploadPoster, logger: appLogger}, nil
}

func (h *PosterHandler) UploadPoster(w http.ResponseWriter, r *http.Request) {
	userID, ok := authdelivery.UserIDFromContext(r.Context())
	if !ok || userID <= 0 {
		h.logger.Warn("poster request rejected: missing or invalid user context", nil)
		httpresponse.WriteError(w, http.StatusUnauthorized, domain.ErrUnauthorized.Error(), h.logger)
		return
	}
	filmID, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil || filmID <= 0 {
		h.logger.Warn("poster request rejected: invalid film ID", map[string]any{"user_id": userID})
		httpresponse.WriteError(w, http.StatusBadRequest, fileusecase.ErrInvalidPosterFilmID.Error(), h.logger)
		return
	}
	fields := map[string]any{"user_id": userID, "film_id": filmID}
	maxRequestSize := h.uploadPoster.MaxSize() + multipartOverhead
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestSize)
	err = r.ParseMultipartForm(maxRequestSize)
	if r.MultipartForm != nil {
		defer func() {
			if err := r.MultipartForm.RemoveAll(); err != nil {
				h.logger.Warn("remove poster multipart temporary files failed", map[string]any{"user_id": userID, "film_id": filmID, "error": err})
			}
		}()
	}
	if err != nil {
		fields["error"] = err
		h.logger.Warn("invalid poster multipart request", fields)
		var sizeErr *http.MaxBytesError
		if errors.As(err, &sizeErr) {
			httpresponse.WriteError(w, http.StatusRequestEntityTooLarge, "upload request is too large", h.logger)
		} else {
			httpresponse.WriteError(w, http.StatusBadRequest, "invalid multipart request", h.logger)
		}
		return
	}
	if len(r.MultipartForm.File[posterFieldName]) != 1 {
		h.logger.Warn("poster request rejected: expected one file", fields)
		httpresponse.WriteError(w, http.StatusBadRequest, "exactly one poster file is required", h.logger)
		return
	}
	file, header, err := r.FormFile(posterFieldName)
	if err != nil {
		fields["error"] = err
		h.logger.Warn("open poster multipart file failed", fields)
		httpresponse.WriteError(w, http.StatusBadRequest, fileusecase.ErrPosterMissing.Error(), h.logger)
		return
	}
	defer func() {
		if err := file.Close(); err != nil {
			h.logger.Warn("close poster upload file failed", map[string]any{"user_id": userID, "film_id": filmID, "error": err})
		}
	}()
	result, err := h.uploadPoster.Run(r.Context(), fileusecase.UploadPosterInput{
		UserID: userID, FilmID: domain.FilmID(filmID), Body: file, Size: header.Size,
	})
	if err != nil {
		h.writePosterError(w, err)
		return
	}
	httpresponse.WriteJSON(w, http.StatusCreated, uploadPosterResponse{Filename: result.Filename}, h.logger)
}

func (h *PosterHandler) writePosterError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrUnauthorized), errors.Is(err, domain.ErrUserNotFound):
		httpresponse.WriteError(w, http.StatusUnauthorized, domain.ErrUnauthorized.Error(), h.logger)
	case errors.Is(err, domain.ErrFilmNotFound):
		httpresponse.WriteError(w, http.StatusNotFound, domain.ErrFilmNotFound.Error(), h.logger)
	case errors.Is(err, fileusecase.ErrPosterTooLarge):
		httpresponse.WriteError(w, http.StatusRequestEntityTooLarge, fileusecase.ErrPosterTooLarge.Error(), h.logger)
	case errors.Is(err, fileusecase.ErrInvalidPosterFilmID), errors.Is(err, fileusecase.ErrPosterMissing),
		errors.Is(err, fileusecase.ErrInvalidPosterSize), errors.Is(err, fileusecase.ErrInvalidPosterImage),
		errors.Is(err, fileusecase.ErrPosterDimensionsTooLarge):
		httpresponse.WriteError(w, http.StatusBadRequest, err.Error(), h.logger)
	default:
		httpresponse.WriteError(w, http.StatusInternalServerError, httpresponse.InternalErrorMessage, h.logger)
	}
}
