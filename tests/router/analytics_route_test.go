package router_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	analyticsdelivery "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/analytics"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	analyticsusecase "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/analytics"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
)

type routerAnalyticsRepo struct{}

func (routerAnalyticsRepo) TopGenresByFilmOpens(context.Context, time.Time, time.Time, int) ([]domain.GenreOpenStat, error) {
	return []domain.GenreOpenStat{
		{GenreID: 1, Name: "drama", Slug: "drama", OpenCount: 4},
	}, nil
}

func (routerAnalyticsRepo) GenreOpensByAge(context.Context, string, time.Time, time.Time) ([]domain.AgeGroupOpenStat, error) {
	return []domain.AgeGroupOpenStat{
		{AgeGroup: domain.AgeGroup18To24, GenreOpenCount: 1, TotalOpenCount: 4, Share: 0.25},
	}, nil
}

func newTestAnalyticsHandler(t *testing.T, appLogger logger.Logger) *analyticsdelivery.Handler {
	t.Helper()

	topGenresUC, err := analyticsusecase.NewTopGenresUseCase(routerAnalyticsRepo{}, appLogger)
	require.NoError(t, err)
	genreAgeUC, err := analyticsusecase.NewGenreAgeUseCase(routerAnalyticsRepo{}, appLogger)
	require.NoError(t, err)
	handler, err := analyticsdelivery.NewHandler(topGenresUC, genreAgeUC, appLogger)
	require.NoError(t, err)
	return handler
}

func TestAnalyticsRouteRequiresAdmin(t *testing.T) {
	tests := []struct {
		name       string
		role       domain.Role
		wantStatus int
	}{
		{name: "anonymous", wantStatus: http.StatusUnauthorized},
		{name: "reader", role: domain.RoleReader, wantStatus: http.StatusForbidden},
		{name: "admin", role: domain.RoleAdmin, wantStatus: http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			appLogger := newSpyLogger(t)
			handler, users := newTestRouterWithAvatarStorageAndUsers(t, appLogger, newRouterAvatarStorage(t))
			req := httptest.NewRequest(http.MethodGet, "/api/admin/analytics/genres?from=2026-11-01&to=2026-12-01", nil)

			if tt.role != "" {
				email := tt.name + "@example.com"
				register := httptest.NewRecorder()
				handler.ServeHTTP(register, httptest.NewRequest(
					http.MethodPost,
					"/api/register",
					strings.NewReader(`{"email":"`+email+`","username":"`+tt.name+`user","password":"Password1"}`),
				))
				require.Equal(t, http.StatusCreated, register.Code, register.Body.String())
				if tt.role == domain.RoleAdmin {
					user, err := users.GetByEmail(context.Background(), email)
					require.NoError(t, err)
					user.Role = domain.RoleAdmin
				}
				req.AddCookie(register.Result().Cookies()[0])
			}

			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, req)
			require.Equal(t, tt.wantStatus, recorder.Code, recorder.Body.String())
			if tt.wantStatus == http.StatusOK {
				require.JSONEq(t, `{"genres":[{"genre_id":1,"name":"drama","slug":"drama","open_count":4}]}`, recorder.Body.String())
			}
		})
	}
}

func TestGenreAgeAnalyticsRouteRequiresAdmin(t *testing.T) {
	tests := []struct {
		name       string
		role       domain.Role
		wantStatus int
	}{
		{name: "anonymous", wantStatus: http.StatusUnauthorized},
		{name: "reader", role: domain.RoleReader, wantStatus: http.StatusForbidden},
		{name: "admin", role: domain.RoleAdmin, wantStatus: http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			appLogger := newSpyLogger(t)
			handler, users := newTestRouterWithAvatarStorageAndUsers(t, appLogger, newRouterAvatarStorage(t))
			req := httptest.NewRequest(http.MethodGet, "/api/admin/analytics/genres/drama/age?from=2026-11-01&to=2026-12-01", nil)

			if tt.role != "" {
				email := "age-" + tt.name + "@example.com"
				register := httptest.NewRecorder()
				handler.ServeHTTP(register, httptest.NewRequest(
					http.MethodPost,
					"/api/register",
					strings.NewReader(`{"email":"`+email+`","username":"age`+tt.name+`user","password":"Password1"}`),
				))
				require.Equal(t, http.StatusCreated, register.Code, register.Body.String())
				if tt.role == domain.RoleAdmin {
					user, err := users.GetByEmail(context.Background(), email)
					require.NoError(t, err)
					user.Role = domain.RoleAdmin
				}
				req.AddCookie(register.Result().Cookies()[0])
			}

			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, req)
			require.Equal(t, tt.wantStatus, recorder.Code, recorder.Body.String())
			if tt.wantStatus == http.StatusOK {
				require.JSONEq(t, `{"genre_slug":"drama","age_groups":[{"age_group":"18_24","genre_open_count":1,"total_open_count":4,"share":0.25}]}`, recorder.Body.String())
			}
		})
	}
}
