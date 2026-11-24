package analytics

import "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"

type genreOpenStatResponse struct {
	GenreID   domain.GenreID `json:"genre_id"`
	Name      string         `json:"name"`
	Slug      string         `json:"slug"`
	OpenCount int64          `json:"open_count"`
}

type topGenresResponse struct {
	Genres []genreOpenStatResponse `json:"genres"`
}

func toGenreOpenStatResponses(stats []domain.GenreOpenStat) []genreOpenStatResponse {
	responses := make([]genreOpenStatResponse, len(stats))
	for i, stat := range stats {
		responses[i] = genreOpenStatResponse{
			GenreID:   stat.GenreID,
			Name:      stat.Name,
			Slug:      stat.Slug,
			OpenCount: stat.OpenCount,
		}
	}

	return responses
}
