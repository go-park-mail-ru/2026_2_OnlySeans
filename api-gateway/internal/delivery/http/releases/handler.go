package releases

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/delivery/http/filmdto"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/delivery/http/httprequest"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/delivery/http/httpresponse"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/middleware"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/logger"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/releaselimits"
	pb "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/proto/catalog"
)

const (
	invalidRequestMessage = "invalid request parameters or body"

	decimalBase           = 10
	idBitSize             = 64
	queryNumberBitSize    = 32
	singleQueryValueCount = 1

	idPathParam       = "id"
	limitQueryParam   = "limit"
	offsetQueryParam  = "offset"
	countryQueryParam = "country"
	typeQueryParam    = "type"
	fromQueryParam    = "from"
	toQueryParam      = "to"
	genreIDQueryParam = "genre_id"

	defaultFeaturedLimit = 3
	maxFeaturedLimit     = 10
)

type Handler struct {
	client    pb.ReleaseServiceClient
	appLogger logger.Logger
}

func NewHandler(client pb.ReleaseServiceClient, appLogger logger.Logger) *Handler {
	return &Handler{client: client, appLogger: appLogger}
}

func (h *Handler) fail(w http.ResponseWriter, r *http.Request, err error) {
	code := http.StatusInternalServerError

	switch status.Code(err) {
	case codes.InvalidArgument:
		code = http.StatusBadRequest
	case codes.Unauthenticated:
		code = http.StatusUnauthorized
	case codes.NotFound:
		code = http.StatusNotFound
	case codes.AlreadyExists, codes.FailedPrecondition:
		code = http.StatusConflict
	case codes.DeadlineExceeded:
		code = http.StatusGatewayTimeout
	case codes.Unavailable:
		code = http.StatusServiceUnavailable
	}

	if code >= http.StatusInternalServerError && h.appLogger != nil {
		h.appLogger.ErrorContext(r.Context(), "release request failed", map[string]any{"error": err.Error()})
	}

	message := status.Convert(err).Message()
	if code >= http.StatusInternalServerError {
		message = httpresponse.InternalErrorMessage
	}

	httpresponse.WriteErrorContext(r.Context(), w, code, message, h.appLogger)
}

func invalid() error {
	return status.Error(codes.InvalidArgument, invalidRequestMessage)
}

func pathID(r *http.Request) (int64, error) {
	id, err := strconv.ParseInt(mux.Vars(r)[idPathParam], decimalBase, idBitSize)
	if err != nil || id <= 0 {
		return 0, invalid()
	}

	return id, nil
}

func queryNumber(r *http.Request, key string, fallback int32) (int32, error) {
	raw, ok := r.URL.Query()[key]
	if !ok {
		return fallback, nil
	}

	if len(raw) != singleQueryValueCount {
		return 0, invalid()
	}

	n, err := strconv.ParseInt(raw[0], decimalBase, queryNumberBitSize)
	if err != nil || n < 0 {
		return 0, invalid()
	}

	return int32(n), nil
}

func page(r *http.Request) (int32, int32, error) {
	limit, err := queryNumber(r, limitQueryParam, releaselimits.DefaultLimit)
	if err != nil || limit < releaselimits.MinLimit || limit > releaselimits.MaxLimit {
		return 0, 0, invalid()
	}

	offset, err := queryNumber(r, offsetQueryParam, releaselimits.MinOffset)

	return limit, offset, err
}

func userRequest(r *http.Request) (*pb.ReleaseUserRequest, error) {
	user, ok := middleware.UserIDFromContext(r.Context())
	if !ok || user <= 0 {
		return nil, status.Error(codes.Unauthenticated, middleware.ErrUnauthorized.Error())
	}

	limit, offset, err := page(r)
	if err != nil {
		return nil, err
	}

	return &pb.ReleaseUserRequest{UserId: user, Limit: limit, Offset: offset}, nil
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request)         { h.list(w, r, false, false) }
func (h *Handler) Featured(w http.ResponseWriter, r *http.Request)     { h.list(w, r, true, false) }
func (h *Handler) FilmReleases(w http.ResponseWriter, r *http.Request) { h.list(w, r, false, true) }
func (h *Handler) list(w http.ResponseWriter, r *http.Request, featured, film bool) {
	limit, offset, err := page(r)
	if err != nil {
		h.fail(w, r, err)
		return
	}

	if featured {
		if !r.URL.Query().Has(limitQueryParam) {
			limit = defaultFeaturedLimit
		}

		if limit > maxFeaturedLimit || offset != releaselimits.MinOffset {
			h.fail(w, r, invalid())
			return
		}
	}

	q := r.URL.Query()
	if q.Has(countryQueryParam) || q.Has(typeQueryParam) {
		h.fail(w, r, invalid())
		return
	}

	req := &pb.ListReleasesRequest{From: q.Get(fromQueryParam), To: q.Get(toQueryParam), Limit: limit, Offset: offset, Featured: featured}
	if film {
		req.FilmId, err = pathID(r)
		if err != nil {
			h.fail(w, r, err)
			return
		}
	}

	if raw := q.Get(genreIDQueryParam); q.Has(genreIDQueryParam) {
		req.GenreId, err = strconv.ParseInt(raw, decimalBase, idBitSize)
		if err != nil || req.GenreId <= 0 {
			h.fail(w, r, invalid())
			return
		}
	}

	data, err := h.client.ListReleases(r.Context(), req)
	if err != nil {
		h.fail(w, r, err)
		return
	}

	h.writeReleases(w, r, data, limit, offset)
}

func (h *Handler) writeReleases(w http.ResponseWriter, r *http.Request, data *pb.ReleaseList, limit, offset int32) {
	items := make([]releaseDTO, len(data.GetItems()))
	for i, event := range data.GetItems() {
		items[i] = releaseDTO{
			FilmID:             event.GetFilm().GetId(),
			WorldPremiereDate:  event.WorldPremiereDate,
			RussiaPremiereDate: event.RussiaPremiereDate,
			Film:               filmdto.FromFilm(event.GetFilm()),
		}
	}

	w.Header().Set("Cache-Control", "no-store")
	httpresponse.WriteJSONContext(r.Context(), w, http.StatusOK, struct {
		Items  []releaseDTO `json:"items"`
		Total  int64        `json:"total"`
		Limit  int32        `json:"limit"`
		Offset int32        `json:"offset"`
	}{
		Items:  items,
		Total:  data.GetTotal(),
		Limit:  limit,
		Offset: offset,
	}, h.appLogger)
}

type releaseDTO struct {
	FilmID             int64                   `json:"film_id"`
	WorldPremiereDate  *string                 `json:"world_premiere_date"`
	RussiaPremiereDate *string                 `json:"russia_premiere_date"`
	Film               filmdto.FilmResponseDTO `json:"film"`
}

type input struct {
	WorldPremiereDate  json.RawMessage `json:"world_premiere_date"`
	RussiaPremiereDate json.RawMessage `json:"russia_premiere_date"`
}

func decodeInput(w http.ResponseWriter, r *http.Request) (*pb.ReleaseInput, error) {
	var in input
	if err := httprequest.DecodeJSON(w, r, &in); err != nil {
		return nil, invalid()
	}

	result := &pb.ReleaseInput{WorldPremiereDateSet: in.WorldPremiereDate != nil, RussiaPremiereDateSet: in.RussiaPremiereDate != nil}
	if !result.WorldPremiereDateSet && !result.RussiaPremiereDateSet {
		return nil, invalid()
	}

	if result.WorldPremiereDateSet {
		if err := json.Unmarshal(in.WorldPremiereDate, &result.WorldPremiereDate); err != nil {
			return nil, invalid()
		}
	}

	if result.RussiaPremiereDateSet {
		if err := json.Unmarshal(in.RussiaPremiereDate, &result.RussiaPremiereDate); err != nil {
			return nil, invalid()
		}
	}

	return result, nil
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		h.fail(w, r, err)
		return
	}

	in, err := decodeInput(w, r)
	if err != nil {
		h.fail(w, r, err)
		return
	}

	_, err = h.client.UpdateRelease(r.Context(), &pb.UpdateReleaseRequest{FilmId: id, Input: in})
	if err != nil {
		h.fail(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Subscribe(w http.ResponseWriter, r *http.Request)   { h.setSubscription(w, r, true) }
func (h *Handler) Unsubscribe(w http.ResponseWriter, r *http.Request) { h.setSubscription(w, r, false) }
func (h *Handler) setSubscription(w http.ResponseWriter, r *http.Request, on bool) {
	id, err := pathID(r)
	if err != nil {
		h.fail(w, r, err)
		return
	}

	user, ok := middleware.UserIDFromContext(r.Context())
	if !ok || user <= 0 {
		h.fail(w, r, status.Error(codes.Unauthenticated, middleware.ErrUnauthorized.Error()))
		return
	}

	_, err = h.client.SetReleaseSubscription(r.Context(), &pb.SetReleaseSubscriptionRequest{UserId: user, FilmId: id, Subscribed: on})
	if err != nil {
		h.fail(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Subscriptions(w http.ResponseWriter, r *http.Request) {
	req, err := userRequest(r)
	if err != nil {
		h.fail(w, r, err)
		return
	}

	data, err := h.client.ListReleaseSubscriptions(r.Context(), req)
	if err != nil {
		h.fail(w, r, err)
		return
	}

	h.writeReleases(w, r, data, req.Limit, req.Offset)
}

func (h *Handler) Notifications(w http.ResponseWriter, r *http.Request) {
	req, err := userRequest(r)
	if err != nil {
		h.fail(w, r, err)
		return
	}

	data, err := h.client.ListNotifications(r.Context(), req)
	if err != nil {
		h.fail(w, r, err)
		return
	}

	items := make([]notificationDTO, len(data.Items))
	for i, n := range data.Items {
		items[i] = notificationDTO{
			ID:        n.Id,
			Message:   n.Message,
			IsRead:    n.IsRead,
			CreatedAt: n.CreatedAt.AsTime(),
			FilmID:    n.FilmId,
			Type:      n.NotificationType,
		}
	}

	w.Header().Set("Cache-Control", "no-store")
	httpresponse.WriteJSONContext(r.Context(), w, http.StatusOK, struct {
		Items  []notificationDTO `json:"items"`
		Total  int64             `json:"total"`
		Limit  int32             `json:"limit"`
		Offset int32             `json:"offset"`
	}{
		Items:  items,
		Total:  data.Total,
		Limit:  req.Limit,
		Offset: req.Offset,
	}, h.appLogger)
}

type notificationDTO struct {
	ID        int64     `json:"id"`
	Message   string    `json:"message"`
	IsRead    bool      `json:"is_read"`
	CreatedAt time.Time `json:"created_at"`
	FilmID    *int64    `json:"film_id"`
	Type      *string   `json:"notification_type"`
}

func (h *Handler) ReadNotification(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		h.fail(w, r, err)
		return
	}

	user, ok := middleware.UserIDFromContext(r.Context())
	if !ok || user <= 0 {
		h.fail(w, r, status.Error(codes.Unauthenticated, middleware.ErrUnauthorized.Error()))
		return
	}

	_, err = h.client.ReadNotification(r.Context(), &pb.ReadNotificationRequest{UserId: user, NotificationId: id})
	if err != nil {
		h.fail(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
