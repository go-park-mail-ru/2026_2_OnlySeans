package repository

import (
	"context"
	"time"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/catalog/internal/domain"
)

type AnalyticsRepository interface {
	TopGenresByFilmOpens(ctx context.Context, from, to time.Time, limit int) ([]domain.GenreOpenStat, error)
	GenreOpensByAge(ctx context.Context, genreSlug string, from, to time.Time) ([]domain.AgeGroupOpenStat, error)
	RecordFilmOpen(ctx context.Context, event domain.FilmOpenEvent) error
}
