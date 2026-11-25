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

type AgeGroup string

const (
	AgeGroupUnder18 AgeGroup = "under_18"
	AgeGroup18To24  AgeGroup = "18_24"
	AgeGroup25To34  AgeGroup = "25_34"
	AgeGroup35To44  AgeGroup = "35_44"
	AgeGroup45To54  AgeGroup = "45_54"
	AgeGroup55Plus  AgeGroup = "55_plus"
	AgeGroupUnknown AgeGroup = "unknown"
)

type AgeGroupOpenStat struct {
	AgeGroup       AgeGroup
	GenreOpenCount int64
	TotalOpenCount int64
	Share          float64
}

type FilmOpenEvent struct {
	VisitorPublicID string
	AccountID       *UserID
	FilmID          FilmID
	OccurredAt      time.Time
}

type AnalyticsRepository interface {
	TopGenresByFilmOpens(ctx context.Context, from time.Time, to time.Time, limit int) ([]GenreOpenStat, error)
	GenreOpensByAge(ctx context.Context, genreSlug string, from, to time.Time) ([]AgeGroupOpenStat, error)
}

type AnalyticsEventRepository interface {
	RecordFilmOpen(ctx context.Context, event FilmOpenEvent) error
}
