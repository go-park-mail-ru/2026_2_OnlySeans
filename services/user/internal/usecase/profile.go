package usecase

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/user/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/user/internal/repository"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/s3"
)

type ProfilePosterStorage interface {
	GetURL(ctx context.Context, key string) (string, error)
}

type ProfileUseCase struct {
	profiles repository.ProfileReader
	posters  ProfilePosterStorage
}

func NewProfileUseCase(profiles repository.ProfileReader, posters ProfilePosterStorage) *ProfileUseCase {
	return &ProfileUseCase{profiles: profiles, posters: posters}
}

func (u *ProfileUseCase) GetProfile(ctx context.Context, userID int64) (*domain.Profile, error) {
	profile, err := u.profiles.GetProfile(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("read profile: %w", err)
	}
	if profile == nil {
		return nil, domain.ErrUserNotFound
	}

	result := *profile
	result.Favorites = slices.Clone(profile.Favorites)
	result.Ratings = slices.Clone(profile.Ratings)
	urls := make(map[string]*string)
	if err := u.attachPosterURLs(ctx, result.Favorites, urls); err != nil {
		return nil, err
	}
	if err := u.attachPosterURLs(ctx, result.Ratings, urls); err != nil {
		return nil, err
	}
	return &result, nil
}

func (u *ProfileUseCase) attachPosterURLs(ctx context.Context, films []domain.ProfileFilm, urls map[string]*string) error {
	for i := range films {
		film := &films[i]
		if film.PosterURL != nil && *film.PosterURL != "" {
			continue
		}
		if film.PosterStorageKey == nil || *film.PosterStorageKey == "" {
			continue
		}

		key := *film.PosterStorageKey
		if url, ok := urls[key]; ok {
			film.PosterURL = url
			continue
		}

		url, err := u.posters.GetURL(ctx, key)
		if errors.Is(err, s3.ErrFileNotFound) {
			urls[key] = nil
			film.PosterURL = nil
			continue
		}
		if err != nil {
			return fmt.Errorf("get profile poster for film %d: %w", film.ID, err)
		}
		urls[key] = &url
		film.PosterURL = &url
	}
	return nil
}
