package domain

import "time"

const (
	MinFilmRating int = 1
	MaxFilmRating int = 10
)

type FilmRating struct {
	AccountID UserID
	FilmID    FilmID
	Score     int
	CreatedAt time.Time
	UpdatedAt time.Time
}

type FilmRatingSummary struct {
	FilmID       FilmID
	AverageScore float64
	Votes        int64
}
