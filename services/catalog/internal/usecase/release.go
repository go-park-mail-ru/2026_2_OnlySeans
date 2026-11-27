package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/catalog/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/catalog/internal/repository"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/releaselimits"
)

const (
	maxReleaseRangeDays = 62
	maxReleaseRange     = maxReleaseRangeDays * 24 * time.Hour
)

type ReleaseUseCase struct {
	repo    repository.ReleaseRepository
	posters PosterURLStorage
	now     func() time.Time
}

func NewReleaseUseCase(repo repository.ReleaseRepository, posters PosterURLStorage) *ReleaseUseCase {
	return &ReleaseUseCase{repo: repo, posters: posters, now: time.Now}
}

func ReleasePagination(limit, offset int) (int, int, error) {
	if limit == 0 {
		limit = releaselimits.DefaultLimit
	}

	if limit < releaselimits.MinLimit || limit > releaselimits.MaxLimit || offset < releaselimits.MinOffset {
		return 0, 0, domain.ErrInvalidRelease
	}

	return limit, offset, nil
}

func (u *ReleaseUseCase) List(ctx context.Context, f domain.ReleaseFilter, user int64) ([]domain.Release, int64, error) {
	var err error

	f.Limit, f.Offset, err = ReleasePagination(f.Limit, f.Offset)
	if err != nil {
		return nil, 0, err
	}

	if f.FilmID < 0 || f.GenreID < 0 || user < 0 {
		return nil, 0, domain.ErrInvalidRelease
	}

	if f.Featured {
		now := u.now().In(domain.ReleaseLocation)
		f.From = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
		f.To = f.From.AddDate(0, 0, maxReleaseRangeDays)
	}

	if f.FilmID == 0 && user == 0 && (f.From.IsZero() || f.To.IsZero()) {
		return nil, 0, domain.ErrInvalidRelease
	}

	if f.From.IsZero() != f.To.IsZero() || (!f.From.IsZero() && (!f.To.After(f.From) || f.To.Sub(f.From) > maxReleaseRange)) {
		return nil, 0, domain.ErrInvalidRelease
	}

	items, total, err := u.repo.List(ctx, f, user)
	if err != nil {
		return nil, 0, fmt.Errorf("list releases: %w", err)
	}

	urls := map[string]string{}

	for i := range items {
		film := &items[i].Film
		if film.PosterURL == nil && film.PosterStorageKey != nil && u.posters != nil {
			key := *film.PosterStorageKey
			url, ok := urls[key]

			if !ok {
				url, err = u.posters.GetURL(ctx, key)
				if err != nil {
					return nil, 0, fmt.Errorf("release poster URL: %w", err)
				}

				urls[key] = url
			}

			film.PosterURL = &url
		}
	}

	return items, total, nil
}

func (u *ReleaseUseCase) Update(ctx context.Context, id int64, input domain.ReleaseInput) error {
	if id <= 0 {
		return domain.ErrInvalidRelease
	}

	if err := input.Validate(); err != nil {
		return err
	}

	return u.repo.Update(ctx, id, input)
}

func (u *ReleaseUseCase) SetSubscription(ctx context.Context, user, id int64, subscribed bool) error {
	if user <= 0 {
		return domain.ErrUnauthorized
	}

	if id <= 0 {
		return domain.ErrInvalidRelease
	}

	return u.repo.SetSubscription(ctx, user, id, subscribed)
}

func (u *ReleaseUseCase) Notifications(ctx context.Context, user int64, limit, offset int) ([]domain.Notification, int64, error) {
	if user <= 0 {
		return nil, 0, domain.ErrUnauthorized
	}

	limit, offset, err := ReleasePagination(limit, offset)
	if err != nil {
		return nil, 0, err
	}

	return u.repo.Notifications(ctx, user, limit, offset)
}

func (u *ReleaseUseCase) ReadNotification(ctx context.Context, user, id int64) error {
	if user <= 0 {
		return domain.ErrUnauthorized
	}

	if id <= 0 {
		return domain.ErrInvalidRelease
	}

	return u.repo.ReadNotification(ctx, user, id)
}
