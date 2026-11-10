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
		_, err := newTestGetLibraryUseCase(t, repo).Run(ctx, user)
		require.ErrorIs(t, err, domain.ErrUnauthorized)
		_, err = newTestGetPreferencesUseCase(t, repo).Run(ctx, user)
		require.ErrorIs(t, err, domain.ErrUnauthorized)
		err = newTestSetSavedUseCase(t, repo).Run(ctx, user, domain.Favorite, 5, true)
		require.ErrorIs(t, err, domain.ErrUnauthorized)
		err = newTestSavePreferenceUseCase(t, repo).Run(ctx, user, "frame:catalog-view", json.RawMessage(`"grid"`))
		require.ErrorIs(t, err, domain.ErrUnauthorized)
	}
}

func TestSetSavedValidation(t *testing.T) {
	for _, tc := range []struct {
		list domain.List
		film int64
	}{{"unknown", 5}, {"", 5}, {domain.Favorite, 0}, {domain.Watched, -1}} {
		err := newTestSetSavedUseCase(t, mocks.NewMockLibraryRepository(gomock.NewController(t))).Run(context.Background(), 7, tc.list, tc.film, true)
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
			require.ErrorIs(t, newTestSetSavedUseCase(t, repo).Run(ctx, 7, list, 5, saved), domain.ErrFilmNotFound)
		}
	}
}

func TestSavePreferenceValidation(t *testing.T) {
	for _, tc := range []struct{ key, value string }{
		{"unknown", "true"}, {"frame:catalog-view", `"table"`},
		{"frame:catalog-view", "true"}, {"frame:sidebar-collapsed", `"true"`},
		{"frame:sidebar-collapsed", "null"}, {"frame:catalog-view", ""},
	} {
		err := newTestSavePreferenceUseCase(t, mocks.NewMockLibraryRepository(gomock.NewController(t))).Run(context.Background(), 7, tc.key, json.RawMessage(tc.value))
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
		require.ErrorIs(t, newTestSavePreferenceUseCase(t, repo).Run(ctx, 7, tc.key, json.RawMessage(tc.value)), dbErr)
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
		result, err := newTestGetLibraryUseCase(t, repo).Run(ctx, 7)
		require.Equal(t, expected, result)
		require.ErrorIs(t, err, dbErr)
		settings, err := newTestGetPreferencesUseCase(t, repo).Run(ctx, 7)
		require.Equal(t, prefs, settings)
		require.ErrorIs(t, err, dbErr)
	}
}

func newTestGetLibraryUseCase(t *testing.T, repo domain.LibraryRepository) *uc.GetLibraryUseCase {
	t.Helper()
	result, err := uc.NewGetLibraryUseCase(repo)
	require.NoError(t, err)
	return result
}

func newTestSetSavedUseCase(t *testing.T, repo domain.LibraryRepository) *uc.SetSavedUseCase {
	t.Helper()
	result, err := uc.NewSetSavedUseCase(repo)
	require.NoError(t, err)
	return result
}

func newTestGetPreferencesUseCase(t *testing.T, repo domain.LibraryRepository) *uc.GetPreferencesUseCase {
	t.Helper()
	result, err := uc.NewGetPreferencesUseCase(repo)
	require.NoError(t, err)
	return result
}

func newTestSavePreferenceUseCase(t *testing.T, repo domain.LibraryRepository) *uc.SavePreferenceUseCase {
	t.Helper()
	result, err := uc.NewSavePreferenceUseCase(repo)
	require.NoError(t, err)
	return result
}

func TestLibraryConstructorsRejectNilRepositories(t *testing.T) {
	t.Run("GetLibrary", func(t *testing.T) {
		result, err := uc.NewGetLibraryUseCase(nil)
		require.Nil(t, result)
		require.ErrorIs(t, err, domain.ErrNilRepo)
	})
	t.Run("SetSaved", func(t *testing.T) {
		result, err := uc.NewSetSavedUseCase(nil)
		require.Nil(t, result)
		require.ErrorIs(t, err, domain.ErrNilRepo)
	})
	t.Run("GetPreferences", func(t *testing.T) {
		result, err := uc.NewGetPreferencesUseCase(nil)
		require.Nil(t, result)
		require.ErrorIs(t, err, domain.ErrNilRepo)
	})
	t.Run("SavePreference", func(t *testing.T) {
		result, err := uc.NewSavePreferenceUseCase(nil)
		require.Nil(t, result)
		require.ErrorIs(t, err, domain.ErrNilRepo)
	})
}
