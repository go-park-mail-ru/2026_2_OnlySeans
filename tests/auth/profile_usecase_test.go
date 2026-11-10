package auth_test

import (
	"context"
	"errors"
	"testing"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	uc "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/auth"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/tests/mocks"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/tests/testutil"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestGetProfileUseCase(t *testing.T) {
	failure := errors.New("storage unavailable")
	for _, tc := range []struct {
		name             string
		user             domain.UserID
		profile          *domain.Profile
		repoErr, wantErr error
	}{
		{name: "success", user: 7, profile: &domain.Profile{User: domain.ProfileUser{ID: 7}, Reviews: 3}},
		{name: "unauthorized", wantErr: domain.ErrUnauthorized},
		{name: "invalid user", user: -1, wantErr: domain.ErrUnauthorized},
		{name: "not found", user: 7, repoErr: domain.ErrUserNotFound, wantErr: domain.ErrUserNotFound},
		{name: "nil result", user: 7, wantErr: domain.ErrUserNotFound},
		{name: "database failure", user: 7, repoErr: failure, wantErr: failure},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := mocks.NewMockProfileReader(ctrl)
			posters := mocks.NewMockProfilePosterStorage(ctrl)
			if tc.user > 0 {
				repo.EXPECT().GetProfile(t.Context(), tc.user).Return(tc.profile, tc.repoErr)
			}
			usecase, err := uc.NewGetProfileUseCase(repo, posters, testutil.DiscardLogger())
			require.NoError(t, err)
			result, err := usecase.Run(t.Context(), tc.user)
			require.ErrorIs(t, err, tc.wantErr)
			if err != nil {
				require.Nil(t, result)
			} else {
				require.Equal(t, tc.profile, result)
			}
		})
	}
}

func TestProfilePosters(t *testing.T) {
	external, key, signed := "https://images.example/poster.png", "posters/private.png", "https://media.example/poster?signature=private"
	failure := errors.New("S3 unavailable")
	for _, tc := range []struct {
		name       string
		storageErr error
	}{
		{name: "signed"}, {name: "missing", storageErr: domain.ErrFileNotFound}, {name: "failure", storageErr: failure},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := mocks.NewMockProfileReader(ctrl)
			posters := mocks.NewMockProfilePosterStorage(ctrl)
			source := &domain.Profile{
				User:      domain.ProfileUser{ID: 7},
				Favorites: []domain.ProfileFilm{{ID: 1, PosterURL: &external}, {ID: 2, PosterStorageKey: &key}},
				Ratings:   []domain.ProfileFilm{{ID: 2, PosterStorageKey: &key}, {ID: 3}},
			}
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			lookup := repo.EXPECT().GetProfile(ctx, domain.UserID(7)).Return(source, nil)
			posters.EXPECT().GetURL(ctx, key).Return(signed, tc.storageErr).Times(1).After(lookup)
			usecase, err := uc.NewGetProfileUseCase(repo, posters, testutil.DiscardLogger())
			require.NoError(t, err)
			result, err := usecase.Run(ctx, 7)
			if tc.storageErr == failure {
				require.ErrorIs(t, err, failure)
				require.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.Equal(t, &external, result.Favorites[0].PosterURL)
				require.Nil(t, result.Ratings[1].PosterURL)
				if tc.storageErr == nil {
					require.Equal(t, &signed, result.Favorites[1].PosterURL)
					require.Equal(t, &signed, result.Ratings[0].PosterURL)
				} else {
					require.Nil(t, result.Favorites[1].PosterURL)
					require.Nil(t, result.Ratings[0].PosterURL)
				}
			}
			require.Nil(t, source.Favorites[1].PosterURL)
			require.Nil(t, source.Ratings[0].PosterURL)
		})
	}
}

func TestNewGetProfileUseCaseRejectsNilDependencies(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockProfileReader(ctrl)
	posters := mocks.NewMockProfilePosterStorage(ctrl)
	_, err := uc.NewGetProfileUseCase(nil, posters, testutil.DiscardLogger())
	require.ErrorIs(t, err, domain.ErrInvalidUserRepo)
	_, err = uc.NewGetProfileUseCase(repo, nil, testutil.DiscardLogger())
	require.Error(t, err)
	_, err = uc.NewGetProfileUseCase(repo, posters, nil)
	require.ErrorIs(t, err, domain.ErrInvalidAppLogger)
}

func TestUpdateProfileValidation(t *testing.T) {
	for _, tc := range []struct {
		name            string
		user            domain.UserID
		email, username string
		wantErr         error
	}{
		{"anonymous", 0, "anna@example.com", "anna", domain.ErrUnauthorized},
		{"invalid email", 7, "invalid", "anna", domain.ErrInvalidEmail},
		{"invalid name", 7, "anna@example.com", "", domain.ErrInvalidUsername},
		{"trimmed", 7, " anna@example.com ", " anna ", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := mocks.NewMockProfileRepository(gomock.NewController(t))
			if tc.wantErr == nil {
				repo.EXPECT().UpdateProfile(t.Context(), tc.user, "anna@example.com", "anna").Return(&domain.User{ID: tc.user}, nil)
			}
			result, err := uc.NewUpdateProfileUseCase(repo).Run(t.Context(), tc.user, tc.email, tc.username)
			require.ErrorIs(t, err, tc.wantErr)
			if err != nil {
				require.Nil(t, result)
			}
		})
	}
}
