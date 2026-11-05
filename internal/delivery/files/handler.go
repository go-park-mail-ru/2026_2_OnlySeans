package files

import (
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
	multipartOverhead    = 1 << 20
	maxUploadRequestSize = fileusecase.MaxAvatarSize + multipartOverhead
	avatarFieldName      = "avatar"
)

type uploadAvatarResponse struct {
	Filename string `json:"filename"`
}

type Handler struct {
	uploadAvatar *fileusecase.UploadAvatarUseCase
	getAvatarURL *fileusecase.GetAvatarURLUseCase
	logger       logger.Logger
}

func NewHandler(uploadAvatar *fileusecase.UploadAvatarUseCase, getAvatarURL *fileusecase.GetAvatarURLUseCase, appLogger logger.Logger) (*Handler, error) {
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
		uploadAvatar: uploadAvatar,
		getAvatarURL: getAvatarURL,
		logger:       appLogger,
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
			httpresponse.WriteError(w, http.StatusBadRequest, err.Error(), h.logger)
		case errors.Is(err, domain.ErrFileNotFound):
			httpresponse.WriteError(w, http.StatusNotFound, domain.ErrFileNotFound.Error(), h.logger)
		default:
			h.logger.Error("get avatar URL failed", map[string]any{"error": err})
			httpresponse.WriteError(w, http.StatusInternalServerError, httpresponse.InternalErrorMessage, h.logger)
		}
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	httpresponse.WriteJSON(w, http.StatusOK, avatarURLResponse{URL: result.URL}, h.logger)
}

func (h *Handler) UploadAvatar(w http.ResponseWriter, r *http.Request) {
	userID, ok := authdelivery.UserIDFromContext(r.Context())
	if !ok || userID <= 0 {
		httpresponse.WriteError(w, http.StatusUnauthorized, domain.ErrUnauthorized.Error(), h.logger)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadRequestSize)
	err := r.ParseMultipartForm(maxUploadRequestSize)
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}
	if err != nil {
		var sizeErr *http.MaxBytesError
		if errors.As(err, &sizeErr) {
			httpresponse.WriteError(w, http.StatusRequestEntityTooLarge, "upload request is too large", h.logger)
		} else {
			httpresponse.WriteError(w, http.StatusBadRequest, "invalid multipart request", h.logger)
		}
		return
	}

	if len(r.MultipartForm.File[avatarFieldName]) != 1 {
		httpresponse.WriteError(w, http.StatusBadRequest, "exactly one avatar file is required", h.logger)
		return
	}
	file, header, err := r.FormFile(avatarFieldName)
	if err != nil {
		httpresponse.WriteError(w, http.StatusBadRequest, fileusecase.ErrAvatarMissing.Error(), h.logger)
		return
	}
	defer file.Close()

	result, err := h.uploadAvatar.Run(r.Context(), fileusecase.UploadAvatarInput{
		UserID: userID,
		Body:   file,
		Size:   header.Size,
	})
	if err != nil {
		h.writeUploadError(w, err)
		return
	}

	httpresponse.WriteJSON(w, http.StatusCreated, uploadAvatarResponse{
		Filename: result.Filename,
	}, h.logger)
}

func (h *Handler) writeUploadError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrUnauthorized), errors.Is(err, domain.ErrUserNotFound):
		httpresponse.WriteError(w, http.StatusUnauthorized, domain.ErrUnauthorized.Error(), h.logger)
	case errors.Is(err, fileusecase.ErrAvatarTooLarge):
		httpresponse.WriteError(w, http.StatusRequestEntityTooLarge, err.Error(), h.logger)
	case errors.Is(err, fileusecase.ErrAvatarMissing),
		errors.Is(err, fileusecase.ErrInvalidAvatarSize),
		errors.Is(err, fileusecase.ErrInvalidAvatarImage),
		errors.Is(err, fileusecase.ErrAvatarDimensionsTooLarge):
		httpresponse.WriteError(w, http.StatusBadRequest, err.Error(), h.logger)
	default:
		h.logger.Error("upload avatar failed", map[string]any{"error": err})
		httpresponse.WriteError(w, http.StatusInternalServerError, httpresponse.InternalErrorMessage, h.logger)
	}
}
