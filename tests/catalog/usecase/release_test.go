package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/catalog/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/catalog/internal/usecase"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/catalog/tests/mocks"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/releaselimits"
)

func TestReleaseRejectsInvalidRangeBeforeRepository(t *testing.T) {
	from := time.Date(2026, time.December, 1, 0, 0, 0, 0, time.UTC)
	for _, filter := range []domain.ReleaseFilter{
		{},
		{From: from},
		{From: from, To: from.AddDate(0, 1, 0), Region: "invalid"},
		{From: from, To: from},
		{From: from, To: from.AddDate(0, 0, -1)},
		{From: from, To: from.AddDate(0, 0, 63)},
		{From: from, To: from.AddDate(0, 1, 0), Limit: releaselimits.MaxLimit + 1},
		{From: from, To: from.AddDate(0, 1, 0), Offset: -1},
		{From: from, To: from.AddDate(0, 1, 0), GenreID: -1},
	} {
		repo := mocks.NewMockReleaseRepository(gomock.NewController(t))
		uc := usecase.NewReleaseUseCase(repo, nil)
		_, _, err := uc.List(t.Context(), filter, 0)
		require.ErrorIs(t, err, domain.ErrInvalidRelease)
	}
}

func TestReleaseListPreservesTotalForEmptyPage(t *testing.T) {
	repo := mocks.NewMockReleaseRepository(gomock.NewController(t))
	filter := domain.ReleaseFilter{
		From:    time.Date(2026, time.December, 1, 0, 0, 0, 0, time.UTC),
		To:      time.Date(2027, time.January, 1, 0, 0, 0, 0, time.UTC),
		GenreID: 3,
		Limit:   10,
		Offset:  100,
	}
	repo.EXPECT().List(gomock.Any(), filter, int64(0)).Return([]domain.Release{}, int64(23), nil)
	uc := usecase.NewReleaseUseCase(repo, nil)
	items, total, err := uc.List(t.Context(), filter, 0)
	require.NoError(t, err)
	require.Empty(t, items)
	require.EqualValues(t, 23, total)
}

func TestReleaseRepositoryFailureIsPreserved(t *testing.T) {
	repo := mocks.NewMockReleaseRepository(gomock.NewController(t))
	failure := errors.New("database unavailable")
	repo.EXPECT().List(gomock.Any(), domain.ReleaseFilter{FilmID: 42, Limit: releaselimits.DefaultLimit}, int64(0)).Return(nil, int64(0), failure)
	uc := usecase.NewReleaseUseCase(repo, nil)
	_, _, err := uc.List(t.Context(), domain.ReleaseFilter{FilmID: 42}, 0)
	require.ErrorIs(t, err, failure)
}

func TestReleaseUpdateValidatesAndPreservesExplicitNull(t *testing.T) {
	repo := mocks.NewMockReleaseRepository(gomock.NewController(t))
	uc := usecase.NewReleaseUseCase(repo, nil)
	require.ErrorIs(t, uc.Update(t.Context(), 42, domain.ReleaseInput{}), domain.ErrInvalidRelease)

	date := "2026-02-30"
	require.ErrorIs(t, uc.Update(t.Context(), 42, domain.ReleaseInput{RussiaPremiereDateSet: true, RussiaPremiereDate: &date}), domain.ErrInvalidRelease)
	clear := domain.ReleaseInput{RussiaPremiereDateSet: true}
	repo.EXPECT().Update(gomock.Any(), int64(42), clear).Return(nil)
	require.NoError(t, uc.Update(t.Context(), 42, clear))
	repo.EXPECT().Update(gomock.Any(), int64(99), clear).Return(domain.ErrFilmNotFound)
	require.ErrorIs(t, uc.Update(t.Context(), 99, clear), domain.ErrFilmNotFound)
}

func TestReleasePersonalOperationsRequireUser(t *testing.T) {
	repo := mocks.NewMockReleaseRepository(gomock.NewController(t))
	uc := usecase.NewReleaseUseCase(repo, nil)
	require.ErrorIs(t, uc.SetSubscription(t.Context(), 0, 42, true), domain.ErrUnauthorized)
	require.ErrorIs(t, uc.ReadNotification(t.Context(), 0, 42), domain.ErrUnauthorized)
	_, _, err := uc.Notifications(t.Context(), 0, 10, 0)
	require.ErrorIs(t, err, domain.ErrUnauthorized)
	repo.EXPECT().SetSubscription(gomock.Any(), int64(7), int64(42), true).Return(domain.ErrReleaseUnavailable)
	require.ErrorIs(t, uc.SetSubscription(t.Context(), 7, 42, true), domain.ErrReleaseUnavailable)
	repo.EXPECT().ReadNotification(gomock.Any(), int64(7), int64(42)).Return(domain.ErrNotificationNotFound)
	require.ErrorIs(t, uc.ReadNotification(t.Context(), 7, 42), domain.ErrNotificationNotFound)
}

func TestReleaseWorkerStopsAfterCancellation(t *testing.T) {
	repo := mocks.NewMockReleaseRepository(gomock.NewController(t))

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	repo.EXPECT().DeliverReminders(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, now time.Time) (int64, error) {
		require.WithinDuration(t, time.Now(), now, time.Second)
		cancel()

		return 1, nil
	})

	uc := usecase.NewReleaseUseCase(repo, nil)
	uc.RunReminders(ctx, nil)
}

func TestReleaseRegionIsForwarded(t *testing.T) {
	for _, region := range []string{"", domain.ReleaseRegionRussia, domain.ReleaseRegionWorld} {
		t.Run(region, func(t *testing.T) {
			repo := mocks.NewMockReleaseRepository(gomock.NewController(t))
			filter := domain.ReleaseFilter{Region: region, FilmID: 42, Limit: releaselimits.DefaultLimit}
			repo.EXPECT().List(gomock.Any(), filter, int64(0)).Return([]domain.Release{}, int64(0), nil)
			_, _, err := usecase.NewReleaseUseCase(repo, nil).List(t.Context(), filter, 0)
			require.NoError(t, err)
		})
	}
}
