package domain

import (
	"context"
	"time"
)

type GenreOpenStat struct {
	GenreID   GenreID
	Name      string
	Slug      string
	OpenCount int64
}

type AnalyticsRepository interface {
	TopGenresByFilmOpens(ctx context.Context, from time.Time, to time.Time, limit int) ([]GenreOpenStat, error)
}
