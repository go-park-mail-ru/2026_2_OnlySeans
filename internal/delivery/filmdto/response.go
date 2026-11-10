package filmdto

import entities "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"

type Genre struct {
	ID   entities.GenreID `json:"id"`
	Name string           `json:"name"`
	Slug string           `json:"slug"`
}

type Film struct {
	ID            entities.FilmID   `json:"id"`
	Title         string            `json:"title"`
	OriginalTitle *string           `json:"original_title,omitempty"`
	FilmType      entities.FilmType `json:"film_type"`
	ReleaseYear   int16             `json:"release_year"`
	DurationMin   *int16            `json:"duration_min,omitempty"`
	AgeLimit      int16             `json:"age_limit"`
	Description   *string           `json:"description,omitempty"`
	PosterURL     *string           `json:"poster_url,omitempty"`
	TrailerURL    *string           `json:"trailer_url,omitempty"`
	Genres        []Genre           `json:"genres"`
}

func FromFilm(film entities.Film) Film {
	return Film{
		ID:            film.ID,
		Title:         film.Title,
		OriginalTitle: film.OriginalTitle,
		FilmType:      film.FilmType,
		ReleaseYear:   film.ReleaseYear,
		DurationMin:   film.DurationMin,
		AgeLimit:      film.AgeLimit,
		Description:   film.Description,
		PosterURL:     film.PosterURL,
		TrailerURL:    film.TrailerURL,
		Genres:        genres(film.Genres),
	}
}

func genres(genres []entities.Genre) []Genre {
	if genres == nil {
		return nil
	}
	result := make([]Genre, len(genres))
	for i, genre := range genres {
		result[i] = Genre{
			ID:   genre.ID,
			Name: genre.Name,
			Slug: genre.Slug,
		}
	}
	return result
}
