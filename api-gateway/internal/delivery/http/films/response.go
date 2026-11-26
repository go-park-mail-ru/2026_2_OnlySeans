package films

import (
	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/delivery/http/filmdto"
	pb "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/proto/catalog"
)

type genreResponseDTO = filmdto.GenreResponseDTO
type filmResponseDTO = filmdto.FilmResponseDTO

type filmCardResponseDTO struct {
	ID          int64              `json:"id"`
	Title       string             `json:"title"`
	ReleaseYear int32              `json:"release_year"`
	PosterURL   *string            `json:"poster_url,omitempty"`
	Genres      []genreResponseDTO `json:"genres"`
}

type collectionResponseDTO struct {
	ID          int64   `json:"id"`
	Title       string  `json:"title"`
	Slug        string  `json:"slug"`
	Description *string `json:"description,omitempty"`
	CoverURL    *string `json:"cover_url,omitempty"`
}

type collectionWithFilmsResponseDTO struct {
	ID          int64                 `json:"id"`
	Title       string                `json:"title"`
	Slug        string                `json:"slug"`
	Description *string               `json:"description,omitempty"`
	CoverURL    *string               `json:"cover_url,omitempty"`
	Films       []filmCardResponseDTO `json:"films"`
}

type createFilmResponse struct {
	ID int64 `json:"id"`
}

func toFilmResponseDTO(film *pb.Film) filmResponseDTO {
	return filmdto.FromFilm(film)
}

func toFilmCardResponseDTO(film *pb.Film) filmCardResponseDTO {
	return filmCardResponseDTO{
		ID:          film.Id,
		Title:       film.Title,
		ReleaseYear: film.ReleaseYear,
		PosterURL:   film.PosterUrl,
		Genres:      toGenreResponseDTOs(film.Genres),
	}
}

func toGenreResponseDTOs(genres []*pb.Genre) []genreResponseDTO {
	result := make([]genreResponseDTO, len(genres))

	for i, genre := range genres {
		result[i] = genreResponseDTO{
			ID:   genre.Id,
			Name: genre.Name,
			Slug: genre.Slug,
		}
	}

	return result
}

func toFilmCardResponseDTOs(films []*pb.Film) []filmCardResponseDTO {
	result := make([]filmCardResponseDTO, len(films))

	for i, film := range films {
		result[i] = toFilmCardResponseDTO(film)
	}

	return result
}

func toCollectionResponseDTO(collection *pb.Collection) collectionResponseDTO {
	return collectionResponseDTO{
		ID:          collection.Id,
		Title:       collection.Title,
		Slug:        collection.Slug,
		Description: collection.Description,
		CoverURL:    collection.CoverUrl,
	}
}

func toCollectionResponseDTOs(collections []*pb.Collection) []collectionResponseDTO {
	result := make([]collectionResponseDTO, len(collections))

	for i, collection := range collections {
		result[i] = toCollectionResponseDTO(collection)
	}

	return result
}

func toCollectionWithFilmsResponseDTO(collection *pb.Collection, films []*pb.Film) collectionWithFilmsResponseDTO {
	dto := toCollectionResponseDTO(collection)

	return collectionWithFilmsResponseDTO{
		ID:          dto.ID,
		Title:       dto.Title,
		Slug:        dto.Slug,
		Description: dto.Description,
		CoverURL:    dto.CoverURL,
		Films:       toFilmCardResponseDTOs(films),
	}
}
