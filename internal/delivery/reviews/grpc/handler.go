package reviewsgrpc

import (
	"context"
	"errors"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	reviewsusecase "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/reviews"
	reviewspb "github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/reviews"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	defaultLimit = 20
	minLimit     = 1
	maxLimit     = 100
)

type Handler struct {
	reviewspb.UnimplementedReviewsServiceServer
	operations reviewsusecase.Operations
}

func NewHandler(operations reviewsusecase.Operations) (*Handler, error) {
	if operations.List == nil || operations.Create == nil || operations.Update == nil || operations.Delete == nil {
		return nil, errors.New("reviews use cases are required")
	}
	return &Handler{operations: operations}, nil
}

func (h *Handler) ListReviews(ctx context.Context, req *reviewspb.ListReviewsRequest) (*reviewspb.ListReviewsResponse, error) {
	filmID := req.GetFilmId()
	if filmID <= 0 {
		return nil, status.Error(codes.InvalidArgument, "film_id must be positive")
	}
	limit := req.GetLimit()
	// в proto3 неприсланное поле limit имеет нулевое значение
	if limit == 0 {
		limit = defaultLimit
	}
	if limit < minLimit || limit > maxLimit {
		return nil, status.Errorf(codes.InvalidArgument, "limit must be between %d and %d", minLimit, maxLimit)
	}
	offset := req.GetOffset()
	if offset < 0 || int64(int(offset)) != offset {
		return nil, status.Error(codes.InvalidArgument, "offset is out of range")
	}
	input := reviewsusecase.NewListReviewsInput(domain.FilmID(filmID), int(limit), int(offset))
	output, err := h.operations.List(ctx, input)
	if err != nil {
		return nil, rpcError(err)
	}
	reviews := make([]*reviewspb.Review, 0, len(output.Reviews))
	for _, review := range output.Reviews {
		reviews = append(reviews, toProtoReview(review))
	}
	return &reviewspb.ListReviewsResponse{Reviews: reviews, Total: int64(output.Total)}, nil
}

func (h *Handler) CreateReview(ctx context.Context, req *reviewspb.CreateReviewRequest) (*reviewspb.CreateReviewResponse, error) {
	if req.GetFilmId() <= 0 || req.GetReview() == nil {
		return nil, status.Error(codes.InvalidArgument, "film_id and review are required")
	}
	if req.GetUserId() <= 0 {
		return nil, rpcError(domain.ErrUnauthorized)
	}
	input := reviewsusecase.NewCreateReviewInput(domain.UserID(req.GetUserId()), domain.FilmID(req.GetFilmId()), fromProtoInput(req.GetReview()))
	result, err := h.operations.Create(ctx, input)
	if err != nil {
		return nil, rpcError(err)
	}
	return &reviewspb.CreateReviewResponse{Id: int64(result.ID)}, nil
}

func (h *Handler) UpdateReview(ctx context.Context, req *reviewspb.UpdateReviewRequest) (*reviewspb.UpdateReviewResponse, error) {
	if req.GetReviewId() <= 0 || req.GetReview() == nil {
		return nil, status.Error(codes.InvalidArgument, "review_id and review are required")
	}
	if req.GetUserId() <= 0 {
		return nil, rpcError(domain.ErrUnauthorized)
	}
	version := req.GetVersion()
	if int64(int(version)) != version {
		return nil, rpcError(domain.ErrInvalidVersion)
	}
	// проверку авторства и версии выполняет существующий юзкейс
	input := reviewsusecase.NewUpdateReviewInput(domain.UserID(req.GetUserId()), domain.ReviewID(req.GetReviewId()), fromProtoInput(req.GetReview()), int(version))
	result, err := h.operations.Update(ctx, input)
	if err != nil {
		return nil, rpcError(err)
	}
	return &reviewspb.UpdateReviewResponse{Review: toProtoReview(result.Review)}, nil
}

func (h *Handler) DeleteReview(ctx context.Context, req *reviewspb.DeleteReviewRequest) (*emptypb.Empty, error) {
	if req.GetReviewId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "review_id must be positive")
	}
	if req.GetUserId() <= 0 {
		return nil, rpcError(domain.ErrUnauthorized)
	}
	input := reviewsusecase.NewDeleteReviewInput(domain.UserID(req.GetUserId()), domain.ReviewID(req.GetReviewId()))
	if err := h.operations.Delete(ctx, input); err != nil {
		return nil, rpcError(err)
	}
	return &emptypb.Empty{}, nil
}

func fromProtoInput(input *reviewspb.ReviewInput) domain.ReviewInput {
	return domain.ReviewInput{Title: input.GetTitle(), Content: input.GetContent(), ContainsSpoilers: input.GetContainsSpoilers()}
}

// toProtoReview переводит доменный отзыв в сообщение для gRPC-ответа
func toProtoReview(review domain.Review) *reviewspb.Review {
	return &reviewspb.Review{
		Id: int64(review.ID), AuthorId: int64(review.AuthorID), FilmId: int64(review.FilmID),
		Title: review.Title, Content: review.Content, ContainsSpoilers: review.ContainsSpoilers,
		Version: int32(review.Version), CreatedAt: timestamppb.New(review.CreatedAt), UpdatedAt: timestamppb.New(review.UpdatedAt),
	}
}
