package filmdto

import pb "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/proto/catalog"

type Genre struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type Film struct {
	ID            int64   `json:"id"`
	Title         string  `json:"title"`
	OriginalTitle *string `json:"original_title,omitempty"`
	FilmType      string  `json:"film_type"`
	ReleaseYear   int32   `json:"release_year"`
	DurationMin   *int32  `json:"duration_min,omitempty"`
	AgeLimit      int32   `json:"age_limit"`
	Description   *string `json:"description,omitempty"`
	PosterURL     *string `json:"poster_url,omitempty"`
	TrailerURL    *string `json:"trailer_url,omitempty"`
	Genres        []Genre `json:"genres"`
}

func FromFilm(film *pb.Film) Film {
	return Film{
		ID:            film.Id,
		Title:         film.Title,
		OriginalTitle: film.OriginalTitle,
		FilmType:      film.FilmType,
		ReleaseYear:   film.ReleaseYear,
		DurationMin:   film.DurationMin,
		AgeLimit:      film.AgeLimit,
		Description:   film.Description,
		PosterURL:     film.PosterUrl,
		TrailerURL:    film.TrailerUrl,
		Genres:        genres(film.Genres),
	}
}

func genres(genres []*pb.Genre) []Genre {
	result := make([]Genre, len(genres))
	for i, genre := range genres {
		result[i] = Genre{
			ID:   genre.Id,
			Name: genre.Name,
			Slug: genre.Slug,
		}
	}

	return result
}
