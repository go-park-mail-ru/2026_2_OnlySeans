package files

import (
	"context"

	"errors"
	"net/http"

	authdelivery "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/auth"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/httpresponse"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	fileusecase "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/files"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
	"github.com/gorilla/mux"
)

const (
	multipartOverhead = 1 << 20
	avatarFieldName   = "avatar"
)

type uploadAvatarResponse struct {
	Filename string `json:"filename"`
}

type Handler struct {
	uploadAvatar  *fileusecase.UploadAvatarUseCase
	getAvatarURL  *fileusecase.GetAvatarURLUseCase
	appLogger     logger.Logger
	accountAvatar *fileusecase.AccountAvatarUseCase
}

func NewHandler(uploadAvatar *fileusecase.UploadAvatarUseCase, getAvatarURL *fileusecase.GetAvatarURLUseCase, accountAvatar *fileusecase.AccountAvatarUseCase, appLogger logger.Logger) (*Handler, error) {
	if accountAvatar == nil {
		return nil, errors.New("account avatar use case is nil")
	}

	if uploadAvatar == nil {
		return nil, errors.New("upload avatar use case is nil")
	}

	if appLogger == nil {
		return nil, errors.New("logger is nil")
	}

	if getAvatarURL == nil {
		return nil, errors.New("get avatar URL use case is nil")
	}

	return &Handler{
		accountAvatar: accountAvatar,
		uploadAvatar:  uploadAvatar,
		getAvatarURL:  getAvatarURL,
		appLogger:     appLogger,
	}, nil
}

type avatarURLResponse struct {
	URL string `json:"url"`
}

func (h *Handler) GetAvatarURL(w http.ResponseWriter, r *http.Request) {
	result, err := h.getAvatarURL.Run(r.Context(), mux.Vars(r)["filename"])
	if err != nil {
		switch {
		case errors.Is(err, fileusecase.ErrInvalidAvatarFilename):
			httpresponse.WriteErrorContext(r.Context(), w, http.StatusBadRequest, err.Error(), h.appLogger)
		case errors.Is(err, domain.ErrFileNotFound):
			httpresponse.WriteErrorContext(r.Context(), w, http.StatusNotFound, domain.ErrFileNotFound.Error(), h.appLogger)
		default:
			httpresponse.WriteErrorContext(r.Context(), w, http.StatusInternalServerError, httpresponse.InternalErrorMessage, h.appLogger)
		}
		return
	}

	w.Header().Set("Cache-Control", "no-store")
	httpresponse.WriteJSONContext(r.Context(), w, http.StatusOK, avatarURLResponse{URL: result.URL}, h.appLogger)
}

func (h *Handler) UploadAvatar(w http.ResponseWriter, r *http.Request) {
	h.saveAvatar(w, r, http.StatusCreated)
}

func (h *Handler) UpdateMyAvatar(w http.ResponseWriter, r *http.Request) {
	h.saveAvatar(w, r, http.StatusOK)
}

func (h *Handler) saveAvatar(w http.ResponseWriter, r *http.Request, status int) {
	userID, ok := authdelivery.UserIDFromContext(r.Context())
	if !ok || userID <= 0 {
		h.appLogger.WarnContext(r.Context(), "avatar request rejected: missing or invalid user context", map[string]any{"operation": r.Method})
		httpresponse.WriteErrorContext(r.Context(), w, http.StatusUnauthorized, domain.ErrUnauthorized.Error(), h.appLogger)
		return
	}

	maxUploadRequestSize := h.uploadAvatar.MaxSize() + multipartOverhead
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadRequestSize)
	err := r.ParseMultipartForm(maxUploadRequestSize)

	if r.MultipartForm != nil {
		defer func() {
			if err := r.MultipartForm.RemoveAll(); err != nil {
				h.appLogger.WarnContext(r.Context(), "remove avatar multipart temporary files failed", map[string]any{"user_id": userID, "error": err})
			}
		}()
	}

	if err != nil {
		var sizeErr *http.MaxBytesError

		if errors.As(err, &sizeErr) {
			h.appLogger.WarnContext(r.Context(), "avatar request body exceeds limit", map[string]any{"user_id": userID, "max_size": maxUploadRequestSize})
			httpresponse.WriteErrorContext(r.Context(), w, http.StatusRequestEntityTooLarge, "upload request is too large", h.appLogger)
		} else {
			h.appLogger.WarnContext(r.Context(), "invalid avatar multipart request", map[string]any{"user_id": userID, "error": err})
			httpresponse.WriteErrorContext(r.Context(), w, http.StatusBadRequest, "invalid multipart request", h.appLogger)
		}

		return
	}

	if len(r.MultipartForm.File[avatarFieldName]) != 1 {
		h.appLogger.WarnContext(r.Context(), "avatar request rejected: expected one file", map[string]any{"user_id": userID, "file_count": len(r.MultipartForm.File[avatarFieldName])})
		httpresponse.WriteErrorContext(r.Context(), w, http.StatusBadRequest, "exactly one avatar file is required", h.appLogger)
		return
	}

	file, header, err := r.FormFile(avatarFieldName)
	if err != nil {
		h.appLogger.WarnContext(r.Context(), "open avatar multipart file failed", map[string]any{"user_id": userID, "error": err})
		httpresponse.WriteErrorContext(r.Context(), w, http.StatusBadRequest, fileusecase.ErrAvatarMissing.Error(), h.appLogger)
		return
	}

	defer func() {
		if err := file.Close(); err != nil {
			h.appLogger.WarnContext(r.Context(), "close avatar upload file failed", map[string]any{"user_id": userID, "error": err})
		}
	}()

	result, err := h.uploadAvatar.Run(r.Context(), fileusecase.UploadAvatarInput{
		UserID: userID,
		Body:   file,
		Size:   header.Size,
	})
	if err != nil {
		h.writeUploadError(r.Context(), w, err)
		return
	}

	httpresponse.WriteJSONContext(r.Context(), w, status, uploadAvatarResponse{
		Filename: result.Filename,
	}, h.appLogger)
}

func (h *Handler) writeUploadError(ctx context.Context, w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrUnauthorized), errors.Is(err, domain.ErrUserNotFound):
		httpresponse.WriteErrorContext(ctx, w, http.StatusUnauthorized, domain.ErrUnauthorized.Error(), h.appLogger)
	case errors.Is(err, fileusecase.ErrAvatarTooLarge):
		httpresponse.WriteErrorContext(ctx, w, http.StatusRequestEntityTooLarge, err.Error(), h.appLogger)
	case errors.Is(err, fileusecase.ErrAvatarMissing),
		errors.Is(err, fileusecase.ErrInvalidAvatarSize),
		errors.Is(err, fileusecase.ErrInvalidAvatarImage),
		errors.Is(err, fileusecase.ErrAvatarDimensionsTooLarge):
		httpresponse.WriteErrorContext(ctx, w, http.StatusBadRequest, err.Error(), h.appLogger)
	default:
		httpresponse.WriteErrorContext(ctx, w, http.StatusInternalServerError, httpresponse.InternalErrorMessage, h.appLogger)
	}
}

func (h *Handler) GetMyAvatarURL(w http.ResponseWriter, r *http.Request) {
	userID, ok := authdelivery.UserIDFromContext(r.Context())
	if !ok || userID <= 0 {
		h.appLogger.WarnContext(r.Context(), "avatar request rejected: missing or invalid user context", map[string]any{"operation": r.Method})
		h.writeAccountAvatarError(r.Context(), w, domain.ErrUnauthorized)
		return
	}

	url, err := h.accountAvatar.GetURL(r.Context(), userID)
	if err != nil {
		h.writeAccountAvatarError(r.Context(), w, err)
		return
	}

	w.Header().Set("Cache-Control", "no-store")
	httpresponse.WriteJSONContext(r.Context(), w, http.StatusOK, avatarURLResponse{URL: url}, h.appLogger)
}

func (h *Handler) DeleteMyAvatar(w http.ResponseWriter, r *http.Request) {
	userID, ok := authdelivery.UserIDFromContext(r.Context())
	if !ok || userID <= 0 {
		h.appLogger.WarnContext(r.Context(), "avatar request rejected: missing or invalid user context", map[string]any{"operation": r.Method})
		h.writeAccountAvatarError(r.Context(), w, domain.ErrUnauthorized)
		return
	}

	if err := h.accountAvatar.Delete(r.Context(), userID); err != nil {
		h.writeAccountAvatarError(r.Context(), w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) writeAccountAvatarError(ctx context.Context, w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrUnauthorized), errors.Is(err, domain.ErrUserNotFound):
		httpresponse.WriteErrorContext(ctx, w, http.StatusUnauthorized, domain.ErrUnauthorized.Error(), h.appLogger)
	case errors.Is(err, domain.ErrFileNotFound):
		httpresponse.WriteErrorContext(ctx, w, http.StatusNotFound, domain.ErrFileNotFound.Error(), h.appLogger)
	default:
		httpresponse.WriteErrorContext(ctx, w, http.StatusInternalServerError, httpresponse.InternalErrorMessage, h.appLogger)
	}
}
