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

type ageGroupOpenStatResponse struct {
	AgeGroup       domain.AgeGroup `json:"age_group"`
	GenreOpenCount int64           `json:"genre_open_count"`
	TotalOpenCount int64           `json:"total_open_count"`
	Share          float64         `json:"share"`
}

type genreAgeResponse struct {
	GenreSlug string                     `json:"genre_slug"`
	AgeGroups []ageGroupOpenStatResponse `json:"age_groups"`
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

func toAgeGroupOpenStatResponses(stats []domain.AgeGroupOpenStat) []ageGroupOpenStatResponse {
	responses := make([]ageGroupOpenStatResponse, len(stats))
	for i, stat := range stats {
		responses[i] = ageGroupOpenStatResponse{
			AgeGroup:       stat.AgeGroup,
			GenreOpenCount: stat.GenreOpenCount,
			TotalOpenCount: stat.TotalOpenCount,
			Share:          stat.Share,
		}
	}

	return responses
}
