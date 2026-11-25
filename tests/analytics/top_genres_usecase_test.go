package analytics_test

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	analyticsusecase "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/analytics"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/tests/testutil"
)

type analyticsRepoStub struct {
	stats        []domain.GenreOpenStat
	err          error
	calls        int
	gotFrom      time.Time
	gotTo        time.Time
	gotLimit     int
	ageStats     []domain.AgeGroupOpenStat
	ageErr       error
	ageCalls     int
	gotGenreSlug string
	gotAgeFrom   time.Time
	gotAgeTo     time.Time
}

func (r *analyticsRepoStub) GenreOpensByAge(_ context.Context, genreSlug string, from, to time.Time) ([]domain.AgeGroupOpenStat, error) {
	r.ageCalls++
	r.gotGenreSlug = genreSlug
	r.gotAgeFrom = from
	r.gotAgeTo = to
	return r.ageStats, r.ageErr
}

func (r *analyticsRepoStub) TopGenresByFilmOpens(_ context.Context, from, to time.Time, limit int) ([]domain.GenreOpenStat, error) {
	r.calls++
	r.gotFrom = from
	r.gotTo = to
	r.gotLimit = limit
	return r.stats, r.err
}

func validTopGenresInput() analyticsusecase.TopGenresInput {
	return analyticsusecase.TopGenresInput{
		From:  time.Date(2026, time.November, 1, 0, 0, 0, 0, time.UTC),
		To:    time.Date(2026, time.December, 1, 0, 0, 0, 0, time.UTC),
		Limit: 10,
	}
}

func TestNewTopGenresUseCase(t *testing.T) {
	repo := &analyticsRepoStub{}

	uc, err := analyticsusecase.NewTopGenresUseCase(repo, testutil.DiscardLogger())
	require.NoError(t, err)
	assert.NotNil(t, uc)

	uc, err = analyticsusecase.NewTopGenresUseCase(nil, testutil.DiscardLogger())
	assert.ErrorIs(t, err, domain.ErrNilAnalyticsRepo)
	assert.Nil(t, uc)

	uc, err = analyticsusecase.NewTopGenresUseCase(repo, nil)
	assert.ErrorIs(t, err, domain.ErrInvalidAppLogger)
	assert.Nil(t, uc)
}

func TestTopGenresUseCase_Run(t *testing.T) {
	input := validTopGenresInput()
	want := []domain.GenreOpenStat{
		{GenreID: 1, Name: "drama", Slug: "drama", OpenCount: 12},
		{GenreID: 3, Name: "comedy", Slug: "comedy", OpenCount: 7},
	}
	repo := &analyticsRepoStub{stats: want}
	uc, err := analyticsusecase.NewTopGenresUseCase(repo, testutil.DiscardLogger())
	require.NoError(t, err)

	got, err := uc.Run(context.Background(), input)
	require.NoError(t, err)
	assert.Equal(t, want, got)
	assert.Equal(t, 1, repo.calls)
	assert.Equal(t, input.From, repo.gotFrom)
	assert.Equal(t, input.To, repo.gotTo)
	assert.Equal(t, input.Limit, repo.gotLimit)
}

func TestTopGenresUseCase_RunInvalidPeriod(t *testing.T) {
	valid := validTopGenresInput()
	tests := []struct {
		name  string
		input analyticsusecase.TopGenresInput
	}{
		{name: "zero from", input: analyticsusecase.TopGenresInput{To: valid.To, Limit: 10}},
		{name: "zero to", input: analyticsusecase.TopGenresInput{From: valid.From, Limit: 10}},
		{name: "equal bounds", input: analyticsusecase.TopGenresInput{From: valid.From, To: valid.From, Limit: 10}},
		{name: "reversed bounds", input: analyticsusecase.TopGenresInput{From: valid.To, To: valid.From, Limit: 10}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &analyticsRepoStub{}
			uc, err := analyticsusecase.NewTopGenresUseCase(repo, testutil.DiscardLogger())
			require.NoError(t, err)

			stats, err := uc.Run(context.Background(), tt.input)
			assert.ErrorIs(t, err, domain.ErrInvalidAnalyticsPeriod)
			assert.Nil(t, stats)
			assert.Zero(t, repo.calls)
		})
	}
}

func TestTopGenresUseCase_RunInvalidLimit(t *testing.T) {
	for _, limit := range []int{-1, 0, 101} {
		t.Run(strconv.Itoa(limit), func(t *testing.T) {
			input := validTopGenresInput()
			input.Limit = limit
			repo := &analyticsRepoStub{}
			uc, err := analyticsusecase.NewTopGenresUseCase(repo, testutil.DiscardLogger())
			require.NoError(t, err)

			stats, err := uc.Run(context.Background(), input)
			assert.ErrorIs(t, err, domain.ErrInvalidAnalyticsLimit)
			assert.Nil(t, stats)
			assert.Zero(t, repo.calls)
		})
	}
}

func TestTopGenresUseCase_RunRepositoryError(t *testing.T) {
	wantErr := errors.New("repository failed")
	repo := &analyticsRepoStub{err: wantErr}
	uc, err := analyticsusecase.NewTopGenresUseCase(repo, testutil.DiscardLogger())
	require.NoError(t, err)

	stats, err := uc.Run(context.Background(), validTopGenresInput())
	assert.ErrorIs(t, err, wantErr)
	assert.Nil(t, stats)
	assert.Equal(t, 1, repo.calls)
}
