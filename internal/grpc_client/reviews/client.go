package reviewsclient

import (
	"context"

	reviewsdelivery "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/reviews"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	reviewspb "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/proto/reviews"
	"google.golang.org/grpc"
)

func Operations(client reviewspb.ReviewsServiceClient) reviewsdelivery.Operations {
	return reviewsdelivery.Operations{
		List: func(ctx context.Context, filmID domain.FilmID, limit, offset int) ([]domain.Review, int, error) {
			// A page of 100 reviews with 20,000 UTF-8 characters can exceed the default 4 MiB limit.
			const maxPageSize = 10 << 20
			response, err := client.ListReviews(ctx, &reviewspb.ListReviewsRequest{
				FilmId: int64(filmID), Limit: int32(limit), Offset: int64(offset),
			}, grpc.MaxCallRecvMsgSize(maxPageSize))
			if err != nil {
				return nil, 0, domainError(err)
			}

			reviews := make([]domain.Review, 0, len(response.GetReviews()))
			for _, review := range response.GetReviews() {
				reviews = append(reviews, fromProtoReview(review))
			}
			return reviews, int(response.GetTotal()), nil
		},
		Create: func(ctx context.Context, authorID domain.UserID, filmID domain.FilmID, input domain.ReviewInput) (domain.ReviewID, error) {
			response, err := client.CreateReview(ctx, &reviewspb.CreateReviewRequest{
				FilmId: int64(filmID), UserId: int64(authorID), Review: toProtoInput(input),
			})
			if err != nil {
				return 0, domainError(err)
			}
			return domain.ReviewID(response.GetId()), nil
		},
		Update: func(ctx context.Context, userID domain.UserID, reviewID domain.ReviewID, input domain.ReviewInput, version int) (domain.Review, error) {
			response, err := client.UpdateReview(ctx, &reviewspb.UpdateReviewRequest{
				ReviewId: int64(reviewID), UserId: int64(userID), Review: toProtoInput(input), Version: int64(version),
			})
			if err != nil {
				return domain.Review{}, domainError(err)
			}
			return fromProtoReview(response.GetReview()), nil
		},
		Delete: func(ctx context.Context, userID domain.UserID, reviewID domain.ReviewID) error {
			_, err := client.DeleteReview(ctx, &reviewspb.DeleteReviewRequest{
				ReviewId: int64(reviewID), UserId: int64(userID),
			})
			return domainError(err)
		},
	}
}

func toProtoInput(input domain.ReviewInput) *reviewspb.ReviewInput {
	return &reviewspb.ReviewInput{Title: input.Title, Content: input.Content, ContainsSpoilers: input.ContainsSpoilers}
}

func fromProtoReview(review *reviewspb.Review) domain.Review {
	return domain.Review{
		ID: domain.ReviewID(review.GetId()), AuthorID: domain.UserID(review.GetAuthorId()), FilmID: domain.FilmID(review.GetFilmId()),
		Title: review.GetTitle(), Content: review.GetContent(), ContainsSpoilers: review.GetContainsSpoilers(),
		Version: int(review.GetVersion()), CreatedAt: review.GetCreatedAt().AsTime(), UpdatedAt: review.GetUpdatedAt().AsTime(),
	}
}
