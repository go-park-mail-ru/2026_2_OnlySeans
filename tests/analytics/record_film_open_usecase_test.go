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

type analyticsEventRepoStub struct {
	event domain.FilmOpenEvent
	err   error
	calls int
}

func (r *analyticsEventRepoStub) RecordFilmOpen(_ context.Context, event domain.FilmOpenEvent) error {
	r.calls++
	r.event = event
	return r.err
}

func TestNewRecordFilmOpenUseCase(t *testing.T) {
	repo := &analyticsEventRepoStub{}

	uc, err := analyticsusecase.NewRecordFilmOpenUseCase(repo, testutil.DiscardLogger())
	require.NoError(t, err)
	assert.NotNil(t, uc)

	uc, err = analyticsusecase.NewRecordFilmOpenUseCase(nil, testutil.DiscardLogger())
	assert.ErrorIs(t, err, domain.ErrNilAnalyticsRepo)
	assert.Nil(t, uc)

	uc, err = analyticsusecase.NewRecordFilmOpenUseCase(repo, nil)
	assert.ErrorIs(t, err, domain.ErrInvalidAppLogger)
	assert.Nil(t, uc)
}

func TestRecordFilmOpenUseCase_Run(t *testing.T) {
	accountID := domain.UserID(42)
	input := analyticsusecase.RecordFilmOpenInput{
		VisitorPublicID: "visitor_public_1234",
		AccountID:       &accountID,
		FilmID:          7,
	}
	repo := &analyticsEventRepoStub{}
	uc, err := analyticsusecase.NewRecordFilmOpenUseCase(repo, testutil.DiscardLogger())
	require.NoError(t, err)
	startedAt := time.Now().UTC()

	err = uc.Run(context.Background(), input)

	require.NoError(t, err)
	assert.Equal(t, 1, repo.calls)
	assert.Equal(t, input.VisitorPublicID, repo.event.VisitorPublicID)
	assert.Equal(t, input.AccountID, repo.event.AccountID)
	assert.Equal(t, input.FilmID, repo.event.FilmID)
	assert.False(t, repo.event.OccurredAt.Before(startedAt))
	assert.False(t, repo.event.OccurredAt.After(time.Now().UTC()))
	assert.Equal(t, time.UTC, repo.event.OccurredAt.Location())
}

func TestRecordFilmOpenUseCase_RunInvalidInput(t *testing.T) {
	invalidAccountID := domain.UserID(0)
	tests := []struct {
		name  string
		input analyticsusecase.RecordFilmOpenInput
	}{
		{name: "empty visitor", input: analyticsusecase.RecordFilmOpenInput{FilmID: 1}},
		{name: "short visitor", input: analyticsusecase.RecordFilmOpenInput{VisitorPublicID: "short", FilmID: 1}},
		{name: "invalid visitor character", input: analyticsusecase.RecordFilmOpenInput{VisitorPublicID: "visitor public 1234", FilmID: 1}},
		{name: "invalid film", input: analyticsusecase.RecordFilmOpenInput{VisitorPublicID: "visitor_public_1234"}},
		{name: "invalid account", input: analyticsusecase.RecordFilmOpenInput{VisitorPublicID: "visitor_public_1234", FilmID: 1, AccountID: &invalidAccountID}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &analyticsEventRepoStub{}
			uc, err := analyticsusecase.NewRecordFilmOpenUseCase(repo, testutil.DiscardLogger())
			require.NoError(t, err)

			err = uc.Run(context.Background(), tt.input)

			assert.ErrorIs(t, err, domain.ErrInvalidAnalyticsEvent)
			assert.Zero(t, repo.calls)
		})
	}
}

func TestRecordFilmOpenUseCase_RunRepositoryError(t *testing.T) {
	wantErr := errors.New("repository failed")
	repo := &analyticsEventRepoStub{err: wantErr}
	uc, err := analyticsusecase.NewRecordFilmOpenUseCase(repo, testutil.DiscardLogger())
	require.NoError(t, err)

	err = uc.Run(context.Background(), analyticsusecase.RecordFilmOpenInput{
		VisitorPublicID: "visitor_public_1234",
		FilmID:          7,
	})

	assert.ErrorIs(t, err, wantErr)
	assert.Equal(t, 1, repo.calls)
}
