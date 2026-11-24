package user

import (
	"errors"
	"io"
	"net/http"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/delivery/http/httpresponse"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/grpc_client/userclient"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/middleware"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/imageutil"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/logger"
	"github.com/gorilla/mux"
)

const (
	avatarFieldName      = "avatar"
	maxAvatarRequestSize = imageutil.MaxAvatarSize + (1 << 20)
)

type AvatarHandler struct {
	userClient userclient.UserClient
	appLogger  logger.Logger
}

func NewAvatarHandler(client userclient.UserClient, appLogger logger.Logger) *AvatarHandler {
	return &AvatarHandler{userClient: client, appLogger: appLogger}
}

//easyjson:json
type avatarURLResponse struct {
	URL string `json:"url"`
}

//easyjson:json
type uploadAvatarResponse struct {
	Filename string `json:"filename"`
}

func (h *AvatarHandler) GetAvatarURL(w http.ResponseWriter, r *http.Request) {
	url, err := h.userClient.GetAvatarURLByFilename(r.Context(), mux.Vars(r)["filename"])
	h.writeAvatarURL(w, r, url, err)
}

func (h *AvatarHandler) GetMyAvatarURL(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.avatarUserID(w, r)
	if !ok {
		return
	}

	url, err := h.userClient.GetAvatarURL(r.Context(), userID)
	h.writeAvatarURL(w, r, url, err)
}

func (h *AvatarHandler) writeAvatarURL(w http.ResponseWriter, r *http.Request, url string, err error) {
	if err != nil {
		h.writeAvatarError(w, r, err)
		return
	}

	w.Header().Set("Cache-Control", "no-store")
	httpresponse.WriteJSONContext(r.Context(), w, http.StatusOK, avatarURLResponse{URL: url}, h.appLogger)
}

func (h *AvatarHandler) DeleteMyAvatar(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.avatarUserID(w, r)
	if !ok {
		return
	}

	if err := h.userClient.DeleteAvatar(r.Context(), userID); err != nil {
		h.writeAvatarError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *AvatarHandler) UploadAvatar(w http.ResponseWriter, r *http.Request) {
	h.saveAvatar(w, r, http.StatusCreated)
}

func (h *AvatarHandler) UpdateMyAvatar(w http.ResponseWriter, r *http.Request) {
	h.saveAvatar(w, r, http.StatusOK)
}

func (h *AvatarHandler) saveAvatar(w http.ResponseWriter, r *http.Request, status int) {
	userID, ok := h.avatarUserID(w, r)
	if !ok {
		return
	}

	ctx := r.Context()
	if r.ContentLength > maxAvatarRequestSize {
		httpresponse.WriteErrorContext(ctx, w, http.StatusRequestEntityTooLarge, "upload request is too large", h.appLogger)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxAvatarRequestSize)
	err := r.ParseMultipartForm(imageutil.MaxAvatarSize)
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}
	if err != nil {
		var sizeErr *http.MaxBytesError
		if errors.As(err, &sizeErr) {
			httpresponse.WriteErrorContext(ctx, w, http.StatusRequestEntityTooLarge, "upload request is too large", h.appLogger)
			return
		}
		httpresponse.WriteErrorContext(ctx, w, http.StatusBadRequest, "invalid multipart request", h.appLogger)
		return
	}

	if len(r.MultipartForm.File[avatarFieldName]) != 1 {
		httpresponse.WriteErrorContext(ctx, w, http.StatusBadRequest, "exactly one avatar file is required", h.appLogger)
		return
	}

	file, header, err := r.FormFile(avatarFieldName)
	if err != nil {
		httpresponse.WriteErrorContext(ctx, w, http.StatusBadRequest, "avatar file is required", h.appLogger)
		return
	}
	defer file.Close()

	if header.Size > imageutil.MaxAvatarSize {
		h.writeAvatarError(w, r, userclient.ErrAvatarTooLarge)
		return
	}
	imageData, err := io.ReadAll(io.LimitReader(file, imageutil.MaxAvatarSize+1))
	if err != nil {
		h.writeAvatarError(w, r, err)
		return
	}
	if len(imageData) > imageutil.MaxAvatarSize {
		h.writeAvatarError(w, r, userclient.ErrAvatarTooLarge)
		return
	}
	if len(imageData) == 0 {
		h.writeAvatarError(w, r, userclient.ErrInvalidArgument)
		return
	}

	filename, err := h.userClient.UpdateAvatar(ctx, userID, imageData)
	if err != nil {
		h.writeAvatarError(w, r, err)
		return
	}

	httpresponse.WriteJSONContext(ctx, w, status, uploadAvatarResponse{Filename: filename}, h.appLogger)
}

func (h *AvatarHandler) avatarUserID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok || userID <= 0 {
		httpresponse.WriteErrorContext(r.Context(), w, http.StatusUnauthorized, middleware.ErrUnauthorized.Error(), h.appLogger)
		return 0, false
	}
	return userID, true
}

func (h *AvatarHandler) writeAvatarError(w http.ResponseWriter, r *http.Request, err error) {
	var status int
	var message string
	switch {
	case errors.Is(err, userclient.ErrUserNotFound):
		status, message = http.StatusUnauthorized, middleware.ErrUnauthorized.Error()
	case errors.Is(err, userclient.ErrAvatarNotFound):
		status, message = http.StatusNotFound, "file not found"
	case errors.Is(err, userclient.ErrInvalidAvatarName):
		status, message = http.StatusBadRequest, err.Error()
	case errors.Is(err, userclient.ErrInvalidArgument):
		status, message = http.StatusBadRequest, "invalid avatar image"
	case errors.Is(err, userclient.ErrAvatarTooLarge):
		status, message = http.StatusRequestEntityTooLarge, err.Error()
	default:
		status, message = http.StatusInternalServerError, httpresponse.InternalErrorMessage
	}
	httpresponse.WriteErrorContext(r.Context(), w, status, message, h.appLogger)
}
