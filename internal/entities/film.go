package entities

import "time"

type FilmID int64

type GenreID int64

type FilmType string

const (
	FilmTypeMovie  FilmType = "movie"
	FilmTypeSeries FilmType = "series"
)

type Genre struct {
	ID        GenreID   `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	Slug      string    `json:"slug" db:"slug"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type Film struct {
	ID             FilmID    `json:"id" db:"id"`
	Title          string    `json:"title" db:"title"`
	OriginalTitle  *string   `json:"original_title,omitempty" db:"original_title"`
	FilmType       FilmType  `json:"film_type" db:"film_type"`
	ProductionYear int16     `json:"production_year" db:"production_year"`
	DurationMin    *int16    `json:"duration_min,omitempty" db:"duration_min"`
	AgeLimit       int16     `json:"age_limit" db:"age_limit"`
	Description    *string   `json:"description,omitempty" db:"description"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" db:"updated_at"`
	Genres         []Genre   `json:"genres" db:"-"`
}
