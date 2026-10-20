package films

import "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/entities"

type genreResponseDTO struct {
	ID   entities.GenreID `json:"id"`
	Name string           `json:"name"`
	Slug string           `json:"slug"`
}

type filmResponseDTO struct {
	ID            entities.FilmID    `json:"id"`
	Title         string             `json:"title"`
	OriginalTitle *string            `json:"original_title,omitempty"`
	FilmType      entities.FilmType  `json:"film_type"`
	ReleaseYear   int16              `json:"release_year"`
	DurationMin   *int16             `json:"duration_min,omitempty"`
	AgeLimit      int16              `json:"age_limit"`
	Description   *string            `json:"description,omitempty"`
	PosterURL     *string            `json:"poster_url,omitempty"`
	TrailerURL    *string            `json:"trailer_url,omitempty"`
	Genres        []genreResponseDTO `json:"genres"`
}

type collectionResponseDTO struct {
	ID          entities.CollectionID `json:"id"`
	Title       string                `json:"title"`
	Slug        string                `json:"slug"`
	Description *string               `json:"description,omitempty"`
	CoverURL    *string               `json:"cover_url,omitempty"`
}

type collectionWithFilmsResponseDTO struct {
	ID          entities.CollectionID `json:"id"`
	Title       string                `json:"title"`
	Slug        string                `json:"slug"`
	Description *string               `json:"description,omitempty"`
	CoverURL    *string               `json:"cover_url,omitempty"`
	Films       []filmResponseDTO     `json:"films"`
}

func toFilmResponseDTO(film entities.Film) filmResponseDTO {
	result := filmResponseDTO{
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
	}
	if film.Genres != nil {
		result.Genres = make([]genreResponseDTO, len(film.Genres))
		for i, genre := range film.Genres {
			result.Genres[i] = genreResponseDTO{
				ID:   genre.ID,
				Name: genre.Name,
				Slug: genre.Slug,
			}
		}
	}
	return result
}

func toFilmResponseDTOs(films []entities.Film) []filmResponseDTO {
	if films == nil {
		return nil
	}
	result := make([]filmResponseDTO, len(films))
	for i, film := range films {
		result[i] = toFilmResponseDTO(film)
	}
	return result
}

func toCollectionResponseDTO(collection entities.Collection) collectionResponseDTO {
	return collectionResponseDTO{
		ID:          collection.ID,
		Title:       collection.Title,
		Slug:        collection.Slug,
		Description: collection.Description,
		CoverURL:    collection.CoverURL,
	}
}

func toCollectionResponseDTOs(collections []entities.Collection) []collectionResponseDTO {
	if collections == nil {
		return nil
	}
	result := make([]collectionResponseDTO, len(collections))
	for i, collection := range collections {
		result[i] = toCollectionResponseDTO(collection)
	}
	return result
}

func toCollectionWithFilmsResponseDTO(collection entities.Collection, films []entities.Film) collectionWithFilmsResponseDTO {
	dto := toCollectionResponseDTO(collection)
	return collectionWithFilmsResponseDTO{
		ID:          dto.ID,
		Title:       dto.Title,
		Slug:        dto.Slug,
		Description: dto.Description,
		CoverURL:    dto.CoverURL,
		Films:       toFilmResponseDTOs(films),
	}
}
