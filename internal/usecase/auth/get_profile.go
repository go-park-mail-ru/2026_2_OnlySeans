package usecase

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/logging"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
)

type ProfileReader interface {
	GetProfile(context.Context, domain.UserID) (*domain.Profile, error)
}

type ProfilePosterStorage interface {
	GetURL(context.Context, string) (string, error)
}

type GetProfileUseCase struct {
	profiles  ProfileReader
	posters   ProfilePosterStorage
	appLogger logger.Logger
}

func NewGetProfileUseCase(profiles ProfileReader, posters ProfilePosterStorage, appLogger logger.Logger) (*GetProfileUseCase, error) {
	if profiles == nil {
		return nil, domain.ErrInvalidUserRepo
	}

	if posters == nil {
		return nil, errors.New("profile poster storage is nil")
	}

	if appLogger == nil {
		return nil, domain.ErrInvalidAppLogger
	}

	return &GetProfileUseCase{profiles: profiles, posters: posters, appLogger: appLogger}, nil
}

func (uc *GetProfileUseCase) Run(ctx context.Context, id domain.UserID) (*domain.Profile, error) {
	if id <= 0 {
		return nil, domain.ErrUnauthorized
	}

	fields := map[string]any{"user_id": id}
	uc.appLogger.DebugContext(ctx, "get profile started", fields)
	profile, err := uc.profiles.GetProfile(ctx, id)
	if err != nil {
		logging.Failure(ctx, uc.appLogger, "get profile failed", err, fields)
		return nil, fmt.Errorf("get profile: %w", err)
	}

	if profile == nil {
		return nil, domain.ErrUserNotFound
	}

	result := *profile
	result.Favorites = slices.Clone(profile.Favorites)
	result.Ratings = slices.Clone(profile.Ratings)
	urls := make(map[string]*string)

	if err := uc.attachProfilePosterURLs(ctx, id, result.Favorites, urls); err != nil {
		return nil, err
	}

	if err := uc.attachProfilePosterURLs(ctx, id, result.Ratings, urls); err != nil {
		return nil, err
	}

	uc.appLogger.InfoContext(ctx, "profile retrieved", fields)
	return &result, nil
}

func (uc *GetProfileUseCase) attachProfilePosterURLs(ctx context.Context, id domain.UserID, films []domain.ProfileFilm, urls map[string]*string) error {
	for i := range films {
		film := &films[i]
		if film.PosterURL != nil && *film.PosterURL != "" {
			continue
		}

		if film.PosterStorageKey == nil || *film.PosterStorageKey == "" {
			continue
		}

		key := *film.PosterStorageKey
		if url, cached := urls[key]; cached {
			film.PosterURL = url
			continue
		}

		url, err := uc.posters.GetURL(ctx, key)
		if errors.Is(err, domain.ErrFileNotFound) {
			uc.appLogger.DebugContext(ctx, "profile poster not found", map[string]any{"user_id": id, "film_id": film.ID})
			urls[key] = nil
			film.PosterURL = nil
			continue
		}

		if err != nil {
			logging.Failure(ctx, uc.appLogger, "get profile poster failed", err, map[string]any{"user_id": id, "film_id": film.ID})
			return fmt.Errorf("get profile poster: %w", err)
		}

		urls[key] = &url
		film.PosterURL = &url
	}

	return nil
}
