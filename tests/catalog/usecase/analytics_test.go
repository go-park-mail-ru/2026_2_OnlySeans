package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/catalog/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/catalog/internal/usecase"
)

type analyticsStub struct {
	topStats   []domain.GenreOpenStat
	ageStats   []domain.AgeGroupOpenStat
	event      domain.FilmOpenEvent
	err        error
	genreSlug  string
	topCalls   int
	ageCalls   int
	eventCalls int
}

func (s *analyticsStub) TopGenresByFilmOpens(context.Context, time.Time, time.Time, int) ([]domain.GenreOpenStat, error) {
	s.topCalls++
	return s.topStats, s.err
}

func (s *analyticsStub) GenreOpensByAge(_ context.Context, genreSlug string, _, _ time.Time) ([]domain.AgeGroupOpenStat, error) {
	s.ageCalls++
	s.genreSlug = genreSlug
	return s.ageStats, s.err
}

func (s *analyticsStub) RecordFilmOpen(_ context.Context, event domain.FilmOpenEvent) error {
	s.eventCalls++
	s.event = event
	return s.err
}

func TestAnalyticsUseCaseReports(t *testing.T) {
	from := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	to := from.AddDate(0, 1, 0)
	repo := &analyticsStub{
		topStats: []domain.GenreOpenStat{{GenreID: 1, Slug: "drama", OpenCount: 3}},
		ageStats: []domain.AgeGroupOpenStat{{AgeGroup: domain.AgeGroup18To24, GenreOpenCount: 1, TotalOpenCount: 4, Share: 0.25}},
	}
	uc := usecase.NewAnalyticsUseCase(repo)

	top, err := uc.TopGenres(t.Context(), from, to, 10)
	require.NoError(t, err)
	require.Equal(t, repo.topStats, top)

	age, err := uc.GenreAge(t.Context(), "  drama  ", from, to)
	require.NoError(t, err)
	require.Equal(t, repo.ageStats, age)
	require.Equal(t, "drama", repo.genreSlug)
}

func TestAnalyticsUseCaseValidation(t *testing.T) {
	from := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	to := from.AddDate(0, 1, 0)
	repo := &analyticsStub{}
	uc := usecase.NewAnalyticsUseCase(repo)

	_, err := uc.TopGenres(t.Context(), to, from, 10)
	require.ErrorIs(t, err, domain.ErrInvalidAnalyticsPeriod)
	_, err = uc.TopGenres(t.Context(), from, to, 0)
	require.ErrorIs(t, err, domain.ErrInvalidAnalyticsLimit)
	_, err = uc.GenreAge(t.Context(), " ", from, to)
	require.ErrorIs(t, err, domain.ErrInvalidAnalyticsGenreSlug)
	require.Zero(t, repo.topCalls)
	require.Zero(t, repo.ageCalls)
}

func TestAnalyticsUseCaseRecordsFilmOpen(t *testing.T) {
	repo := &analyticsStub{}
	uc := usecase.NewAnalyticsUseCase(repo)
	accountID := domain.UserID(42)

	err := uc.RecordFilmOpen(t.Context(), "visitor_public_1234", &accountID, 7)
	require.NoError(t, err)
	require.Equal(t, 1, repo.eventCalls)
	require.Equal(t, domain.FilmID(7), repo.event.FilmID)
	require.Equal(t, &accountID, repo.event.AccountID)
	require.False(t, repo.event.OccurredAt.IsZero())
}

func TestAnalyticsUseCaseRecordErrors(t *testing.T) {
	wantErr := errors.New("storage unavailable")
	repo := &analyticsStub{err: wantErr}
	uc := usecase.NewAnalyticsUseCase(repo)

	err := uc.RecordFilmOpen(t.Context(), "short", nil, 7)
	require.ErrorIs(t, err, domain.ErrInvalidAnalyticsEvent)
	require.Zero(t, repo.eventCalls)

	err = uc.RecordFilmOpen(t.Context(), "visitor_public_1234", nil, 7)
	require.ErrorIs(t, err, wantErr)
}
