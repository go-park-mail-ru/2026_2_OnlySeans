package user

import (
	"errors"
	"net/http"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/delivery/http/httprequest"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/delivery/http/httpresponse"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/grpc_client/userclient"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/middleware"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/logger"
)

type ProfileHandler struct {
	userClient userclient.UserClient
	appLogger  logger.Logger
}

//easyjson:json
type updateProfileRequest struct {
	Email    string `json:"email"`
	Username string `json:"username"`
}

func NewProfileHandler(client userclient.UserClient, appLogger logger.Logger) *ProfileHandler {
	return &ProfileHandler{userClient: client, appLogger: appLogger}
}

func (h *ProfileHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, ok := middleware.UserIDFromContext(ctx)

	if !ok {
		httpresponse.WriteErrorContext(ctx, w, http.StatusUnauthorized, middleware.ErrUnauthorized.Error(), h.appLogger)
		return
	}

	profile, err := h.userClient.GetProfile(ctx, id)
	if errors.Is(err, userclient.ErrUserNotFound) {
		httpresponse.WriteErrorContext(ctx, w, http.StatusUnauthorized, middleware.ErrUnauthorized.Error(), h.appLogger)
		return
	}

	if err != nil {
		httpresponse.WriteErrorContext(ctx, w, http.StatusInternalServerError, httpresponse.InternalErrorMessage, h.appLogger)
		return
	}

	w.Header().Set("Cache-Control", "no-store")
	httpresponse.WriteJSONContext(ctx, w, http.StatusOK, toProfileResponse(profile), h.appLogger)
}

func (h *ProfileHandler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, ok := middleware.UserIDFromContext(ctx)

	if !ok {
		httpresponse.WriteErrorContext(ctx, w, http.StatusUnauthorized, middleware.ErrUnauthorized.Error(), h.appLogger)
		return
	}

	var body updateProfileRequest
	if err := httprequest.DecodeJSON(w, r, &body); err != nil {
		httpresponse.WriteErrorContext(ctx, w, http.StatusBadRequest, httprequest.ErrInvalidBody.Error(), h.appLogger)
		return
	}

	user, err := h.userClient.UpdateProfile(ctx, id, body.Email, body.Username)
	if err != nil {
		status, message := profileUpdateError(err, r.URL.Path == "/api/me/profile")
		if status == http.StatusInternalServerError && h.appLogger != nil {
			h.appLogger.ErrorContext(ctx, "update profile failed", map[string]any{"error": err.Error()})
		}

		httpresponse.WriteErrorContext(ctx, w, status, message, h.appLogger)

		return
	}

	w.Header().Set("Cache-Control", "no-store")
	httpresponse.WriteJSONContext(ctx, w, http.StatusOK, profileUpdateResponse{User: toUserResponse(user)}, h.appLogger)
}

func profileUpdateError(err error, myProfile bool) (int, string) {
	switch {
	case errors.Is(err, userclient.ErrInvalidEmail):
		return http.StatusBadRequest, userclient.ErrInvalidEmail.Error()
	case errors.Is(err, userclient.ErrInvalidUsername):
		return http.StatusBadRequest, userclient.ErrInvalidUsername.Error()
	case errors.Is(err, userclient.ErrInvalidArgument):
		return http.StatusBadRequest, userclient.ErrInvalidArgument.Error()
	case errors.Is(err, userclient.ErrEmailAlreadyExists):
		if myProfile {
			return http.StatusConflict, "email taken"
		}

		return http.StatusConflict, userclient.ErrEmailAlreadyExists.Error()
	case errors.Is(err, userclient.ErrUsernameTaken):
		if myProfile {
			return http.StatusConflict, "username taken"
		}

		return http.StatusConflict, userclient.ErrUsernameTaken.Error()
	case errors.Is(err, userclient.ErrUserNotFound):
		return http.StatusUnauthorized, middleware.ErrUnauthorized.Error()
	default:
		return http.StatusInternalServerError, httpresponse.InternalErrorMessage
	}
}
