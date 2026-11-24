package analytics_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	analyticsdelivery "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/analytics"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	analyticsusecase "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/analytics"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/tests/testutil"
)

func newAnalyticsHandler(t *testing.T, repo *analyticsRepoStub) *analyticsdelivery.Handler {
	t.Helper()

	appLogger := testutil.DiscardLogger()
	uc, err := analyticsusecase.NewTopGenresUseCase(repo, appLogger)
	require.NoError(t, err)
	handler, err := analyticsdelivery.NewHandler(uc, appLogger)
	require.NoError(t, err)
	return handler
}

func TestNewAnalyticsHandler(t *testing.T) {
	appLogger := testutil.DiscardLogger()
	uc, err := analyticsusecase.NewTopGenresUseCase(&analyticsRepoStub{}, appLogger)
	require.NoError(t, err)

	handler, err := analyticsdelivery.NewHandler(nil, appLogger)
	assert.Error(t, err)
	assert.Nil(t, handler)

	handler, err = analyticsdelivery.NewHandler(uc, nil)
	assert.Error(t, err)
	assert.Nil(t, handler)
}

func TestAnalyticsHandler_TopGenres(t *testing.T) {
	repo := &analyticsRepoStub{stats: []domain.GenreOpenStat{
		{GenreID: 1, Name: "drama", Slug: "drama", OpenCount: 12},
		{GenreID: 3, Name: "comedy", Slug: "comedy", OpenCount: 7},
	}}
	handler := newAnalyticsHandler(t, repo)
	req := httptest.NewRequest(http.MethodGet, "/api/admin/analytics/genres?from=2026-11-01&to=2026-12-01&limit=2", nil)
	recorder := httptest.NewRecorder()

	handler.TopGenres(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.JSONEq(t, `{
		"genres": [
			{"genre_id": 1, "name": "drama", "slug": "drama", "open_count": 12},
			{"genre_id": 3, "name": "comedy", "slug": "comedy", "open_count": 7}
		]
	}`, recorder.Body.String())
	assert.Equal(t, 1, repo.calls)
	assert.Equal(t, 2, repo.gotLimit)
}

func TestAnalyticsHandler_TopGenresEmptyUsesDefaultLimit(t *testing.T) {
	repo := &analyticsRepoStub{stats: []domain.GenreOpenStat{}}
	handler := newAnalyticsHandler(t, repo)
	req := httptest.NewRequest(http.MethodGet, "/api/admin/analytics/genres?from=2026-11-01&to=2026-12-01", nil)
	recorder := httptest.NewRecorder()

	handler.TopGenres(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.JSONEq(t, `{"genres": []}`, recorder.Body.String())
	assert.Equal(t, 1, repo.calls)
	assert.Equal(t, 10, repo.gotLimit)
}

func TestAnalyticsHandler_TopGenresInvalidInput(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{name: "missing from", url: "/api/admin/analytics/genres?to=2026-12-01"},
		{name: "invalid from", url: "/api/admin/analytics/genres?from=01-11-2026&to=2026-12-01"},
		{name: "missing to", url: "/api/admin/analytics/genres?from=2026-11-01"},
		{name: "equal bounds", url: "/api/admin/analytics/genres?from=2026-11-01&to=2026-11-01"},
		{name: "non integer limit", url: "/api/admin/analytics/genres?from=2026-11-01&to=2026-12-01&limit=many"},
		{name: "zero limit", url: "/api/admin/analytics/genres?from=2026-11-01&to=2026-12-01&limit=0"},
		{name: "too large limit", url: "/api/admin/analytics/genres?from=2026-11-01&to=2026-12-01&limit=101"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &analyticsRepoStub{}
			handler := newAnalyticsHandler(t, repo)
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			recorder := httptest.NewRecorder()

			handler.TopGenres(recorder, req)

			assert.Equal(t, http.StatusBadRequest, recorder.Code)
			assert.Contains(t, recorder.Body.String(), `"error"`)
			assert.Zero(t, repo.calls)
		})
	}
}

func TestAnalyticsHandler_TopGenresInternalError(t *testing.T) {
	repo := &analyticsRepoStub{err: errors.New("database unavailable")}
	handler := newAnalyticsHandler(t, repo)
	req := httptest.NewRequest(http.MethodGet, "/api/admin/analytics/genres?from=2026-11-01&to=2026-12-01", nil)
	recorder := httptest.NewRecorder()

	handler.TopGenres(recorder, req)

	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
	assert.JSONEq(t, `{"error": "internal server error"}`, recorder.Body.String())
	assert.Equal(t, 1, repo.calls)
}
