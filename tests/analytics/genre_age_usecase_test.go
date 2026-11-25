package analytics_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	analyticsusecase "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/analytics"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/tests/testutil"
)

func validGenreAgeInput() analyticsusecase.GenreAgeInput {
	return analyticsusecase.GenreAgeInput{
		GenreSlug: "drama",
		From:      time.Date(2026, time.November, 1, 0, 0, 0, 0, time.UTC),
		To:        time.Date(2026, time.December, 1, 0, 0, 0, 0, time.UTC),
	}
}

func TestNewGenreAgeUseCase(t *testing.T) {
	repo := &analyticsRepoStub{}

	uc, err := analyticsusecase.NewGenreAgeUseCase(repo, testutil.DiscardLogger())
	require.NoError(t, err)
	assert.NotNil(t, uc)

	uc, err = analyticsusecase.NewGenreAgeUseCase(nil, testutil.DiscardLogger())
	assert.ErrorIs(t, err, domain.ErrNilAnalyticsRepo)
	assert.Nil(t, uc)

	uc, err = analyticsusecase.NewGenreAgeUseCase(repo, nil)
	assert.ErrorIs(t, err, domain.ErrInvalidAppLogger)
	assert.Nil(t, uc)
}

func TestGenreAgeUseCase_Run(t *testing.T) {
	input := validGenreAgeInput()
	input.GenreSlug = "  drama  "
	want := []domain.AgeGroupOpenStat{
		{AgeGroup: domain.AgeGroup18To24, GenreOpenCount: 3, TotalOpenCount: 12, Share: 0.25},
		{AgeGroup: domain.AgeGroupUnknown, GenreOpenCount: 2, TotalOpenCount: 4, Share: 0.5},
	}
	repo := &analyticsRepoStub{ageStats: want}
	uc, err := analyticsusecase.NewGenreAgeUseCase(repo, testutil.DiscardLogger())
	require.NoError(t, err)

	got, err := uc.Run(context.Background(), input)
	require.NoError(t, err)
	assert.Equal(t, want, got)
	assert.Equal(t, 1, repo.ageCalls)
	assert.Equal(t, "drama", repo.gotGenreSlug)
	assert.Equal(t, input.From, repo.gotAgeFrom)
	assert.Equal(t, input.To, repo.gotAgeTo)
}

func TestGenreAgeUseCase_RunInvalidGenreSlug(t *testing.T) {
	for _, genreSlug := range []string{"", "   "} {
		repo := &analyticsRepoStub{}
		uc, err := analyticsusecase.NewGenreAgeUseCase(repo, testutil.DiscardLogger())
		require.NoError(t, err)
		input := validGenreAgeInput()
		input.GenreSlug = genreSlug

		stats, err := uc.Run(context.Background(), input)
		assert.ErrorIs(t, err, domain.ErrInvalidAnalyticsGenreSlug)
		assert.Nil(t, stats)
		assert.Zero(t, repo.ageCalls)
	}
}

func TestGenreAgeUseCase_RunInvalidPeriod(t *testing.T) {
	valid := validGenreAgeInput()
	tests := []struct {
		name  string
		input analyticsusecase.GenreAgeInput
	}{
		{name: "zero from", input: analyticsusecase.GenreAgeInput{GenreSlug: "drama", To: valid.To}},
		{name: "zero to", input: analyticsusecase.GenreAgeInput{GenreSlug: "drama", From: valid.From}},
		{name: "equal bounds", input: analyticsusecase.GenreAgeInput{GenreSlug: "drama", From: valid.From, To: valid.From}},
		{name: "reversed bounds", input: analyticsusecase.GenreAgeInput{GenreSlug: "drama", From: valid.To, To: valid.From}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &analyticsRepoStub{}
			uc, err := analyticsusecase.NewGenreAgeUseCase(repo, testutil.DiscardLogger())
			require.NoError(t, err)

			stats, err := uc.Run(context.Background(), tt.input)
			assert.ErrorIs(t, err, domain.ErrInvalidAnalyticsPeriod)
			assert.Nil(t, stats)
			assert.Zero(t, repo.ageCalls)
		})
	}
}

func TestGenreAgeUseCase_RunRepositoryError(t *testing.T) {
	wantErr := errors.New("repository failed")
	repo := &analyticsRepoStub{ageErr: wantErr}
	uc, err := analyticsusecase.NewGenreAgeUseCase(repo, testutil.DiscardLogger())
	require.NoError(t, err)

	stats, err := uc.Run(context.Background(), validGenreAgeInput())
	assert.ErrorIs(t, err, wantErr)
	assert.Nil(t, stats)
	assert.Equal(t, 1, repo.ageCalls)
}
