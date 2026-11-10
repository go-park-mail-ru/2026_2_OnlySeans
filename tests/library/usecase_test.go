package library_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	uc "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/library"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/tests/mocks"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestUnauthorizedDoesNotCallRepository(t *testing.T) {
	ctx := context.Background()
	repo := mocks.NewMockLibraryRepository(gomock.NewController(t))
	for _, user := range []domain.UserID{0, -1} {
		_, err := uc.NewGetLibraryUseCase(repo).Run(ctx, user)
		require.ErrorIs(t, err, domain.ErrUnauthorized)
		_, err = uc.NewGetPreferencesUseCase(repo).Run(ctx, user)
		require.ErrorIs(t, err, domain.ErrUnauthorized)
		err = uc.NewSetSavedUseCase(repo).Run(ctx, user, domain.Favorite, 5, true)
		require.ErrorIs(t, err, domain.ErrUnauthorized)
		err = uc.NewSavePreferenceUseCase(repo).Run(ctx, user, "frame:catalog-view", json.RawMessage(`"grid"`))
		require.ErrorIs(t, err, domain.ErrUnauthorized)
	}
}

func TestSetSavedValidation(t *testing.T) {
	for _, tc := range []struct {
		list domain.List
		film int64
	}{{"unknown", 5}, {"", 5}, {domain.Favorite, 0}, {domain.Watched, -1}} {
		err := uc.NewSetSavedUseCase(mocks.NewMockLibraryRepository(gomock.NewController(t))).Run(context.Background(), 7, tc.list, tc.film, true)
		require.ErrorIs(t, err, domain.ErrInvalidLibraryData)
	}
}

func TestSetSavedPassesArgumentsAndError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	for _, list := range []domain.List{domain.Favorite, domain.WatchLater, domain.Watched} {
		for _, saved := range []bool{true, false} {
			repo := mocks.NewMockLibraryRepository(gomock.NewController(t))
			repo.EXPECT().SetSaved(ctx, domain.UserID(7), list, int64(5), saved).Return(domain.ErrFilmNotFound).Times(1)
			require.ErrorIs(t, uc.NewSetSavedUseCase(repo).Run(ctx, 7, list, 5, saved), domain.ErrFilmNotFound)
		}
	}
}

func TestSavePreferenceValidation(t *testing.T) {
	for _, tc := range []struct{ key, value string }{
		{"unknown", "true"}, {"frame:catalog-view", `"table"`},
		{"frame:catalog-view", "true"}, {"frame:sidebar-collapsed", `"true"`},
		{"frame:sidebar-collapsed", "null"}, {"frame:catalog-view", ""},
	} {
		err := uc.NewSavePreferenceUseCase(mocks.NewMockLibraryRepository(gomock.NewController(t))).Run(context.Background(), 7, tc.key, json.RawMessage(tc.value))
		require.ErrorIs(t, err, domain.ErrInvalidLibraryData)
	}
}

func TestSavePreferencePassesArgumentsAndError(t *testing.T) {
	ctx := context.Background()
	dbErr := errors.New("database unavailable")
	for _, tc := range []struct{ key, value string }{
		{"frame:catalog-view", `"grid"`}, {"frame:catalog-view", `"list"`},
		{"frame:sidebar-collapsed", "true"}, {"frame:sidebar-collapsed", "false"},
	} {
		repo := mocks.NewMockLibraryRepository(gomock.NewController(t))
		repo.EXPECT().SavePreference(ctx, domain.UserID(7), tc.key, json.RawMessage(tc.value)).Return(dbErr).Times(1)
		require.ErrorIs(t, uc.NewSavePreferenceUseCase(repo).Run(ctx, 7, tc.key, json.RawMessage(tc.value)), dbErr)
	}
}

func TestReadsReturnRepositoryResults(t *testing.T) {
	ctx := context.Background()
	expected := domain.Library{Favorite: []int64{5}, WatchLater: []int64{}, Watched: []int64{8}}
	prefs := json.RawMessage(`{"frame:catalog-view":"grid"}`)
	for _, dbErr := range []error{nil, errors.New("database unavailable")} {
		repo := mocks.NewMockLibraryRepository(gomock.NewController(t))
		repo.EXPECT().Get(ctx, domain.UserID(7)).Return(expected, dbErr).Times(1)
		repo.EXPECT().Preferences(ctx, domain.UserID(7)).Return(prefs, dbErr).Times(1)
		result, err := uc.NewGetLibraryUseCase(repo).Run(ctx, 7)
		require.Equal(t, expected, result)
		require.ErrorIs(t, err, dbErr)
		settings, err := uc.NewGetPreferencesUseCase(repo).Run(ctx, 7)
		require.Equal(t, prefs, settings)
		require.ErrorIs(t, err, dbErr)
	}
}
