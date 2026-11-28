package releases_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	authhttp "github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/delivery/http/auth"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/delivery/http/releases"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/delivery/router"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/middleware"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/catalog/tests/mocks"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/logger"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/releaselimits"
	pb "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/proto/catalog"
)

func TestListAndInputValidation(t *testing.T) {
	for _, query := range []string{"limit=0", "limit=501", "limit=abc", "offset=-1", "limit=2&limit=3", "genre_id=0", "genre_id=no", "country=US", "type=digital"} {
		t.Run(query, func(t *testing.T) {
			client := mocks.NewMockReleaseServiceClient(gomock.NewController(t))
			h := releases.NewHandler(client, nil)
			w := httptest.NewRecorder()
			h.List(w, httptest.NewRequest("GET", "/api/releases?"+query, nil))
			require.Equal(t, 400, w.Code)
		})
	}

	client := mocks.NewMockReleaseServiceClient(gomock.NewController(t))
	gomock.InOrder(
		client.EXPECT().ListReleases(gomock.Any(), &pb.ListReleasesRequest{
			From:  "2026-12-01",
			To:    "2027-01-01",
			Limit: releaselimits.DefaultLimit,
		}).Return(&pb.ReleaseList{}, nil),
		client.EXPECT().ListReleases(gomock.Any(), &pb.ListReleasesRequest{
			Featured: true,
			Limit:    3,
		}).Return(&pb.ReleaseList{}, nil),
	)

	h := releases.NewHandler(client, nil)
	w := httptest.NewRecorder()
	h.List(w, httptest.NewRequest("GET", "/api/releases?from=2026-12-01&to=2027-01-01", nil))
	require.Equal(t, 200, w.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, []any{}, body["items"])
	require.Equal(t, "no-store", w.Header().Get("Cache-Control"))
	w = httptest.NewRecorder()
	h.Featured(w, httptest.NewRequest("GET", "/api/releases/featured", nil))
	require.Equal(t, 200, w.Code)
	w = httptest.NewRecorder()
	h.Featured(w, httptest.NewRequest("GET", "/api/releases/featured?limit=11", nil))
	require.Equal(t, 400, w.Code)
}

func TestPremiereDateResponse(t *testing.T) {
	date := "2026-12-17"
	client := mocks.NewMockReleaseServiceClient(gomock.NewController(t))
	client.EXPECT().ListReleases(gomock.Any(), &pb.ListReleasesRequest{
		From:  "2026-12-01",
		To:    "2027-01-01",
		Limit: releaselimits.DefaultLimit,
	}).Return(
		&pb.ReleaseList{
			Total: 1,
			Items: []*pb.Release{{
				Film: &pb.Film{
					Id:    42,
					Title: "Film",
				},
				RussiaPremiereDate: &date,
			}},
		}, nil,
	)

	h := releases.NewHandler(client, nil)
	w := httptest.NewRecorder()
	h.List(w, httptest.NewRequest("GET", "/api/releases?from=2026-12-01&to=2027-01-01", nil))
	require.Equal(t, 200, w.Code)
	var body struct {
		Items []map[string]any `json:"items"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Len(t, body.Items, 1)
	require.EqualValues(t, 42, body.Items[0]["film_id"])
	require.Equal(t, date, body.Items[0]["russia_premiere_date"])
	require.Contains(t, body.Items[0], "world_premiere_date")
	require.Nil(t, body.Items[0]["world_premiere_date"])
}

func TestPatchPreservesAbsentFieldsAndRejectsBadBody(t *testing.T) {
	date := "2027-01-03"
	client := mocks.NewMockReleaseServiceClient(gomock.NewController(t))
	gomock.InOrder(
		client.EXPECT().UpdateRelease(gomock.Any(), &pb.UpdateReleaseRequest{
			FilmId: 9,
			Input: &pb.ReleaseInput{
				RussiaPremiereDate:    &date,
				RussiaPremiereDateSet: true,
			},
		}).Return(&emptypb.Empty{}, nil),
		client.EXPECT().UpdateRelease(gomock.Any(), &pb.UpdateReleaseRequest{
			FilmId: 9,
			Input:  &pb.ReleaseInput{RussiaPremiereDateSet: true},
		}).Return(&emptypb.Empty{}, nil),
	)

	h := releases.NewHandler(client, nil)

	for _, body := range []string{`{"russia_premiere_date":"2027-01-03"}`, `{"russia_premiere_date":null}`} {
		r := mux.SetURLVars(httptest.NewRequest("PATCH", "/api/films/9/releases", strings.NewReader(body)), map[string]string{"id": "9"})
		w := httptest.NewRecorder()
		h.Update(w, r)
		require.Equal(t, 204, w.Code)
	}

	for _, body := range []string{`{}`, `null`, `{"status":null}`, `{"film_id":1}`, `{"russia_premiere_date":123}`, `{"status":"cancelled"} {}`} {
		r := mux.SetURLVars(httptest.NewRequest("PATCH", "/api/films/9/releases", strings.NewReader(body)), map[string]string{"id": "9"})
		w := httptest.NewRecorder()
		h.Update(w, r)
		require.Equal(t, 400, w.Code, body)
	}
}

func TestSubscriptionIdentityIdempotentMethodsAndErrors(t *testing.T) {
	client := mocks.NewMockReleaseServiceClient(gomock.NewController(t))
	gomock.InOrder(
		client.EXPECT().SetReleaseSubscription(gomock.Any(), &pb.SetReleaseSubscriptionRequest{
			UserId:     42,
			FilmId:     7,
			Subscribed: true,
		}).Return(&emptypb.Empty{}, nil),
		client.EXPECT().SetReleaseSubscription(gomock.Any(), &pb.SetReleaseSubscriptionRequest{
			UserId:     42,
			FilmId:     7,
			Subscribed: false,
		}).Return(&emptypb.Empty{}, nil),
	)

	h := releases.NewHandler(client, nil)
	req := func() *http.Request {
		r := mux.SetURLVars(httptest.NewRequest("PUT", "/api/me/release-subscriptions/7", nil), map[string]string{"id": "7"})
		return r.WithContext(middleware.ContextWithUserID(r.Context(), 42))
	}
	w := httptest.NewRecorder()
	h.Subscribe(w, req())
	require.Equal(t, 204, w.Code)
	w = httptest.NewRecorder()
	h.Unsubscribe(w, req())
	require.Equal(t, 204, w.Code)

	for _, tc := range []struct {
		code codes.Code
		want int
	}{
		{
			code: codes.FailedPrecondition,
			want: 409,
		},
		{
			code: codes.NotFound,
			want: 404,
		},
		{
			code: codes.Unavailable,
			want: 503,
		},
		{
			code: codes.Internal,
			want: 500,
		},
	} {
		client.EXPECT().SetReleaseSubscription(gomock.Any(), &pb.SetReleaseSubscriptionRequest{
			UserId:     42,
			FilmId:     7,
			Subscribed: true,
		}).Return(nil, status.Error(tc.code, "database internals"))

		w = httptest.NewRecorder()
		h.Subscribe(w, req())
		require.Equal(t, tc.want, w.Code)

		if tc.want >= 500 {
			require.NotContains(t, w.Body.String(), "database internals")
		}
	}

	w = httptest.NewRecorder()
	h.Subscribe(w, mux.SetURLVars(httptest.NewRequest("PUT", "/", nil), map[string]string{"id": "7"}))
	require.Equal(t, 401, w.Code)
}

func TestRoutesRequireSessionAndAdvertisePatch(t *testing.T) {
	log, err := logger.NewLogger(io.Discard, logger.LogLevelType("info"), "json")
	require.NoError(t, err)
	client := mocks.NewMockReleaseServiceClient(gomock.NewController(t))
	h := releases.NewHandler(client, log)
	routes, err := router.NewRouter(router.Handlers{
		Auth:     &authhttp.AuthHandler{},
		Releases: h,
	}, middleware.NewAuthMiddleware(nil, nil, log), []string{"http://localhost:8000"}, log)
	require.NoError(t, err)

	for _, tc := range []struct {
		method string
		path   string
	}{
		{
			method: "PATCH",
			path:   "/api/films/1/releases",
		},
		{
			method: "PUT",
			path:   "/api/me/release-subscriptions/1",
		},
		{
			method: "DELETE",
			path:   "/api/me/release-subscriptions/1",
		},
		{
			method: "GET",
			path:   "/api/me/release-subscriptions",
		},
		{
			method: "GET",
			path:   "/api/me/notifications",
		},
		{
			method: "PUT",
			path:   "/api/me/notifications/1/read",
		},
	} {
		w := httptest.NewRecorder()
		routes.ServeHTTP(w, httptest.NewRequest(tc.method, tc.path, nil))
		require.Equal(t, 401, w.Code, tc.path)
	}

	client.EXPECT().ListReleases(gomock.Any(), &pb.ListReleasesRequest{
		Featured: true,
		Limit:    3,
	}).Return(&pb.ReleaseList{}, nil)

	w := httptest.NewRecorder()
	routes.ServeHTTP(w, httptest.NewRequest("GET", "/api/releases/featured", nil))
	require.Equal(t, 200, w.Code)

	r := httptest.NewRequest("OPTIONS", "/api/releases/1", nil)
	r.Header.Set("Origin", "http://localhost:8000")
	r.Header.Set("Access-Control-Request-Method", "PATCH")

	w = httptest.NewRecorder()
	routes.ServeHTTP(w, r)
	require.Equal(t, 204, w.Code)
	require.Contains(t, w.Header().Get("Access-Control-Allow-Methods"), "PATCH")
}

func TestNotificationResponseAndOwnership(t *testing.T) {
	client := mocks.NewMockReleaseServiceClient(gomock.NewController(t))
	film := int64(42)
	kind := "release_reminder"
	client.EXPECT().ListNotifications(gomock.Any(), &pb.ReleaseUserRequest{
		UserId: 7,
		Limit:  10,
	}).Return(&pb.NotificationList{
		Total: 1,
		Items: []*pb.Notification{{
			Id:               12,
			Message:          "Premiere",
			FilmId:           &film,
			NotificationType: &kind,
			CreatedAt:        timestamppb.New(time.Date(2026, time.December, 17, 9, 0, 0, 0, time.UTC)),
		}},
	}, nil)

	h := releases.NewHandler(client, nil)
	r := httptest.NewRequest("GET", "/api/me/notifications?limit=10", nil)
	r = r.WithContext(middleware.ContextWithUserID(r.Context(), 7))
	w := httptest.NewRecorder()
	h.Notifications(w, r)
	require.Equal(t, 200, w.Code)
	require.JSONEq(t, `{"items":[{"id":12,"message":"Premiere","is_read":false,"created_at":"2026-12-17T09:00:00Z","film_id":42,"notification_type":"release_reminder"}],"total":1,"limit":10,"offset":0}`, w.Body.String())
	client.EXPECT().ReadNotification(gomock.Any(), &pb.ReadNotificationRequest{
		UserId:         7,
		NotificationId: 12,
	}).Return(&emptypb.Empty{}, nil)
	r = mux.SetURLVars(httptest.NewRequest("PUT", "/api/me/notifications/12/read", nil), map[string]string{"id": "12"})
	r = r.WithContext(middleware.ContextWithUserID(r.Context(), 7))
	w = httptest.NewRecorder()
	h.ReadNotification(w, r)
	require.Equal(t, 204, w.Code)
}

func TestWorldRegionIsForwarded(t *testing.T) {
	client := mocks.NewMockReleaseServiceClient(gomock.NewController(t))
	client.EXPECT().ListReleases(gomock.Any(), &pb.ListReleasesRequest{
		From: "2026-12-01", To: "2027-01-01", Region: "world", Limit: releaselimits.DefaultLimit,
	}).Return(&pb.ReleaseList{}, nil)

	w := httptest.NewRecorder()
	releases.NewHandler(client, nil).List(w, httptest.NewRequest("GET", "/api/releases?from=2026-12-01&to=2027-01-01&region=world", nil))
	require.Equal(t, http.StatusOK, w.Code)
}
