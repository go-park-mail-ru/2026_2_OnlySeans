package domain

import (
	"errors"
	"time"
)

const (
	ReleaseRegionWorld  = "world"
	ReleaseRegionRussia = "russia"

	minReleaseYear        = 1888
	maxReleaseYear        = 2200
	releaseTimezoneName   = "Europe/Moscow"
	releaseTimezoneOffset = 3 * 60 * 60
)

var (
	ErrInvalidRelease       = errors.New("invalid release data")
	ErrReleaseUnavailable   = errors.New("russian premiere date is unknown or already passed")
	ErrNotificationNotFound = errors.New("notification not found")
)

var ReleaseLocation = time.FixedZone(releaseTimezoneName, releaseTimezoneOffset)

type Release struct {
	WorldPremiereDate  *time.Time
	RussiaPremiereDate *time.Time
	Film               Film
}

type ReleaseFilter struct {
	Region   string
	From     time.Time
	To       time.Time
	FilmID   FilmID
	GenreID  GenreID
	Limit    int
	Offset   int
	Featured bool
}

type ReleaseInput struct {
	WorldPremiereDate     *string
	RussiaPremiereDate    *string
	WorldPremiereDateSet  bool
	RussiaPremiereDateSet bool
}

func ParseReleaseDate(value string) (time.Time, error) {
	date, err := time.Parse(time.DateOnly, value)
	if err != nil || date.Year() < minReleaseYear || date.After(time.Date(maxReleaseYear, time.January, 1, 0, 0, 0, 0, time.UTC)) {
		return time.Time{}, ErrInvalidRelease
	}

	return date, nil
}

func (in ReleaseInput) Validate() error {
	if !in.WorldPremiereDateSet && !in.RussiaPremiereDateSet {
		return ErrInvalidRelease
	}

	if in.WorldPremiereDate != nil {
		if !in.WorldPremiereDateSet {
			return ErrInvalidRelease
		}

		if _, err := ParseReleaseDate(*in.WorldPremiereDate); err != nil {
			return err
		}
	}

	if in.RussiaPremiereDate != nil {
		if !in.RussiaPremiereDateSet {
			return ErrInvalidRelease
		}

		if _, err := ParseReleaseDate(*in.RussiaPremiereDate); err != nil {
			return err
		}
	}

	return nil
}

type Notification struct {
	ID        int64
	Message   string
	IsRead    bool
	CreatedAt time.Time
	FilmID    *int64
	Type      *string
}
