package reviewsgrpc

import (
	"context"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	reviewsusecase "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/reviews"
	reviewspb "github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/reviews"
	"google.golang.org/grpc"
)

func ClientOperations(client reviewspb.ReviewsServiceClient) reviewsusecase.Operations {
	return reviewsusecase.Operations{
		List: func(ctx context.Context, in reviewsusecase.ListReviewsInput) (*reviewsusecase.ListReviewsOutput, error) {
			// страница из 100 отзывов по 20 000 символов UTF-8 может превышать стандартные 4 МиБ
			const maxPageSize = 10 << 20
			response, err := client.ListReviews(ctx, &reviewspb.ListReviewsRequest{
				FilmId: int64(in.FilmID), Limit: int32(in.Limit), Offset: int64(in.Offset),
			}, grpc.MaxCallRecvMsgSize(maxPageSize))
			if err != nil {
				return nil, domainError(err)
			}

			reviews := make([]domain.Review, 0, len(response.GetReviews()))
			for _, review := range response.GetReviews() {
				reviews = append(reviews, fromProtoReview(review))
			}
			return reviewsusecase.NewListReviewsOutput(reviews, int(response.GetTotal())), nil
		},
		Create: func(ctx context.Context, in reviewsusecase.CreateReviewInput) (*reviewsusecase.CreateReviewOutput, error) {
			response, err := client.CreateReview(ctx, &reviewspb.CreateReviewRequest{
				FilmId: int64(in.FilmID), UserId: int64(in.AuthorID), Review: toProtoInput(in.Review),
			})
			if err != nil {
				return nil, domainError(err)
			}
			return &reviewsusecase.CreateReviewOutput{ID: domain.ReviewID(response.GetId())}, nil
		},
		Update: func(ctx context.Context, in reviewsusecase.UpdateReviewInput) (*reviewsusecase.ReviewOutput, error) {
			response, err := client.UpdateReview(ctx, &reviewspb.UpdateReviewRequest{
				ReviewId: int64(in.ReviewID), UserId: int64(in.UserID), Review: toProtoInput(in.Review), Version: int64(in.Version),
			})
			if err != nil {
				return nil, domainError(err)
			}
			review := fromProtoReview(response.GetReview())
			return reviewsusecase.NewReviewOutput(&review), nil
		},
		Delete: func(ctx context.Context, in reviewsusecase.DeleteReviewInput) error {
			_, err := client.DeleteReview(ctx, &reviewspb.DeleteReviewRequest{
				ReviewId: int64(in.ReviewID), UserId: int64(in.UserID),
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
