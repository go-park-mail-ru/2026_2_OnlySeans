package ratings

import pb "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/proto/catalog"

func toFilmRatingResponseDTO(rating *pb.FilmRating) filmRatingResponseDTO {
	return filmRatingResponseDTO{
		FilmID:    rating.FilmId,
		Score:     rating.Score,
		CreatedAt: rating.GetCreatedAt().AsTime(),
		UpdatedAt: rating.GetUpdatedAt().AsTime(),
	}
}

func toFilmRatingSummaryResponseDTO(summary *pb.FilmRatingSummary) filmRatingSummaryResponseDTO {
	return filmRatingSummaryResponseDTO{
		FilmID:       summary.FilmId,
		AverageScore: summary.AverageScore,
		Votes:        summary.Votes,
	}
}

func toFilmRatingSummaryResponseDTOs(summaries []*pb.FilmRatingSummary) []filmRatingSummaryResponseDTO {
	result := make([]filmRatingSummaryResponseDTO, len(summaries))
	for i, summary := range summaries {
		result[i] = toFilmRatingSummaryResponseDTO(summary)
	}

	return result
}
