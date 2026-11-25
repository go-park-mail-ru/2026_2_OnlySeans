package analytics_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
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
	topGenresUC, err := analyticsusecase.NewTopGenresUseCase(repo, appLogger)
	require.NoError(t, err)
	genreAgeUC, err := analyticsusecase.NewGenreAgeUseCase(repo, appLogger)
	require.NoError(t, err)
	handler, err := analyticsdelivery.NewHandler(topGenresUC, genreAgeUC, appLogger)
	require.NoError(t, err)
	return handler
}

func TestNewAnalyticsHandler(t *testing.T) {
	appLogger := testutil.DiscardLogger()
	repo := &analyticsRepoStub{}
	topGenresUC, err := analyticsusecase.NewTopGenresUseCase(repo, appLogger)
	require.NoError(t, err)
	genreAgeUC, err := analyticsusecase.NewGenreAgeUseCase(repo, appLogger)
	require.NoError(t, err)

	handler, err := analyticsdelivery.NewHandler(nil, genreAgeUC, appLogger)
	assert.Error(t, err)
	assert.Nil(t, handler)

	handler, err = analyticsdelivery.NewHandler(topGenresUC, nil, appLogger)
	assert.Error(t, err)
	assert.Nil(t, handler)

	handler, err = analyticsdelivery.NewHandler(topGenresUC, genreAgeUC, nil)
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

func TestAnalyticsHandler_GenreAge(t *testing.T) {
	repo := &analyticsRepoStub{ageStats: []domain.AgeGroupOpenStat{
		{AgeGroup: domain.AgeGroup18To24, GenreOpenCount: 3, TotalOpenCount: 12, Share: 0.25},
		{AgeGroup: domain.AgeGroupUnknown, GenreOpenCount: 2, TotalOpenCount: 4, Share: 0.5},
	}}
	handler := newAnalyticsHandler(t, repo)
	req := httptest.NewRequest(http.MethodGet, "/api/admin/analytics/genres/drama/age?from=2026-11-01&to=2026-12-01", nil)
	req = mux.SetURLVars(req, map[string]string{"slug": "drama"})
	recorder := httptest.NewRecorder()

	handler.GenreAge(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.JSONEq(t, `{
		"genre_slug": "drama",
		"age_groups": [
			{"age_group": "18_24", "genre_open_count": 3, "total_open_count": 12, "share": 0.25},
			{"age_group": "unknown", "genre_open_count": 2, "total_open_count": 4, "share": 0.5}
		]
	}`, recorder.Body.String())
	assert.Equal(t, 1, repo.ageCalls)
	assert.Equal(t, "drama", repo.gotGenreSlug)
}

func TestAnalyticsHandler_GenreAgeEmpty(t *testing.T) {
	repo := &analyticsRepoStub{ageStats: []domain.AgeGroupOpenStat{}}
	handler := newAnalyticsHandler(t, repo)
	req := httptest.NewRequest(http.MethodGet, "/api/admin/analytics/genres/drama/age?from=2026-11-01&to=2026-12-01", nil)
	req = mux.SetURLVars(req, map[string]string{"slug": "drama"})
	recorder := httptest.NewRecorder()

	handler.GenreAge(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.JSONEq(t, `{"genre_slug":"drama","age_groups":[]}`, recorder.Body.String())
	assert.Equal(t, 1, repo.ageCalls)
}

func TestAnalyticsHandler_GenreAgeInvalidInput(t *testing.T) {
	tests := []struct {
		name      string
		url       string
		genreSlug string
	}{
		{name: "missing genre slug", url: "/api/admin/analytics/genres//age?from=2026-11-01&to=2026-12-01"},
		{name: "missing from", genreSlug: "drama", url: "/api/admin/analytics/genres/drama/age?to=2026-12-01"},
		{name: "invalid from", genreSlug: "drama", url: "/api/admin/analytics/genres/drama/age?from=01-11-2026&to=2026-12-01"},
		{name: "missing to", genreSlug: "drama", url: "/api/admin/analytics/genres/drama/age?from=2026-11-01"},
		{name: "equal bounds", genreSlug: "drama", url: "/api/admin/analytics/genres/drama/age?from=2026-11-01&to=2026-11-01"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &analyticsRepoStub{}
			handler := newAnalyticsHandler(t, repo)
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			if tt.genreSlug != "" {
				req = mux.SetURLVars(req, map[string]string{"slug": tt.genreSlug})
			}
			recorder := httptest.NewRecorder()

			handler.GenreAge(recorder, req)

			assert.Equal(t, http.StatusBadRequest, recorder.Code)
			assert.Contains(t, recorder.Body.String(), `"error"`)
			assert.Zero(t, repo.ageCalls)
		})
	}
}

func TestAnalyticsHandler_GenreAgeInternalError(t *testing.T) {
	repo := &analyticsRepoStub{ageErr: errors.New("database unavailable")}
	handler := newAnalyticsHandler(t, repo)
	req := httptest.NewRequest(http.MethodGet, "/api/admin/analytics/genres/drama/age?from=2026-11-01&to=2026-12-01", nil)
	req = mux.SetURLVars(req, map[string]string{"slug": "drama"})
	recorder := httptest.NewRecorder()

	handler.GenreAge(recorder, req)

	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
	assert.JSONEq(t, `{"error":"internal server error"}`, recorder.Body.String())
	assert.Equal(t, 1, repo.ageCalls)
}
