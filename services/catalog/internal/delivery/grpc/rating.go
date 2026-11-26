package grpc

import (
	"context"

	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/catalog/internal/domain"
	pb "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/proto/catalog"
)

func (h *CatalogHandler) SetFilmRating(ctx context.Context, req *pb.SetFilmRatingRequest) (*pb.SetFilmRatingResponse, error) {
	if req == nil {
		return nil, missingRequest()
	}

	rating, err := h.ratings.SetFilmRating(ctx, domain.UserID(req.GetUserId()), domain.FilmID(req.GetFilmId()), int(req.GetScore()))
	if err != nil {
		return nil, toGRPCError(err)
	}

	return &pb.SetFilmRatingResponse{
		Rating: &pb.FilmRating{
			UserId:    int64(rating.AccountID),
			FilmId:    int64(rating.FilmID),
			Score:     int32(rating.Score),
			CreatedAt: timestamppb.New(rating.CreatedAt),
			UpdatedAt: timestamppb.New(rating.UpdatedAt),
		},
	}, nil
}

func (h *CatalogHandler) GetFilmRating(ctx context.Context, req *pb.GetFilmRatingRequest) (*pb.GetFilmRatingResponse, error) {
	if req == nil {
		return nil, missingRequest()
	}

	summary, err := h.ratings.GetFilmRating(ctx, domain.FilmID(req.GetFilmId()))
	if err != nil {
		return nil, toGRPCError(err)
	}

	return &pb.GetFilmRatingResponse{
		Rating: toProtoRatingSummary(summary),
	}, nil
}

func (h *CatalogHandler) GetMyFilmRating(ctx context.Context, req *pb.GetMyFilmRatingRequest) (*pb.GetMyFilmRatingResponse, error) {
	if req == nil {
		return nil, missingRequest()
	}

	score, err := h.ratings.GetMyFilmRating(ctx, domain.UserID(req.GetUserId()), domain.FilmID(req.GetFilmId()))
	if err != nil {
		return nil, toGRPCError(err)
	}

	var result *int32

	if score != nil {
		value := int32(*score)
		result = &value
	}

	return &pb.GetMyFilmRatingResponse{
		Score: result,
	}, nil
}

func (h *CatalogHandler) DeleteFilmRating(ctx context.Context, req *pb.DeleteFilmRatingRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, missingRequest()
	}

	if err := h.ratings.DeleteFilmRating(ctx, domain.UserID(req.GetUserId()), domain.FilmID(req.GetFilmId())); err != nil {
		return nil, toGRPCError(err)
	}

	return &emptypb.Empty{}, nil
}

func (h *CatalogHandler) GetCollectionRatings(ctx context.Context, req *pb.GetCollectionRatingsRequest) (*pb.GetCollectionRatingsResponse, error) {
	if req == nil {
		return nil, missingRequest()
	}

	limit, offset, err := parsePagination(req.GetPagination())
	if err != nil {
		return nil, toGRPCError(err)
	}

	ratings, total, err := h.ratings.GetCollectionRatings(ctx, domain.CollectionID(req.GetCollectionId()), limit, offset)
	if err != nil {
		return nil, toGRPCError(err)
	}

	result := make([]*pb.FilmRatingSummary, len(ratings))
	for i, rating := range ratings {
		result[i] = toProtoRatingSummary(rating)
	}

	return &pb.GetCollectionRatingsResponse{
		Ratings: result,
		Total:   int64(total),
	}, nil
}

func toProtoRatingSummary(rating domain.FilmRatingSummary) *pb.FilmRatingSummary {
	return &pb.FilmRatingSummary{
		FilmId:       int64(rating.FilmID),
		AverageScore: rating.AverageScore,
		Votes:        rating.Votes,
	}
}
