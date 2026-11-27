package grpc

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/catalog/internal/domain"
	pb "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/proto/catalog"
)

func (h *CatalogHandler) TopGenres(ctx context.Context, req *pb.TopGenresRequest) (*pb.TopGenresResponse, error) {
	if req == nil || req.GetFrom() == nil || req.GetTo() == nil {
		return nil, missingRequest()
	}

	stats, err := h.analytics.TopGenres(ctx, req.GetFrom().AsTime(), req.GetTo().AsTime(), int(req.GetLimit()))
	if err != nil {
		return nil, analyticsGRPCError(err)
	}

	result := make([]*pb.GenreOpenStat, len(stats))
	for i, stat := range stats {
		result[i] = &pb.GenreOpenStat{GenreId: int64(stat.GenreID), Name: stat.Name, Slug: stat.Slug, OpenCount: stat.OpenCount}
	}

	return &pb.TopGenresResponse{Genres: result}, nil
}

func (h *CatalogHandler) GenreAge(ctx context.Context, req *pb.GenreAgeRequest) (*pb.GenreAgeResponse, error) {
	if req == nil || req.GetFrom() == nil || req.GetTo() == nil {
		return nil, missingRequest()
	}

	stats, err := h.analytics.GenreAge(ctx, req.GetGenreSlug(), req.GetFrom().AsTime(), req.GetTo().AsTime())
	if err != nil {
		return nil, analyticsGRPCError(err)
	}

	result := make([]*pb.AgeGroupOpenStat, len(stats))
	for i, stat := range stats {
		result[i] = &pb.AgeGroupOpenStat{
			AgeGroup:       string(stat.AgeGroup),
			GenreOpenCount: stat.GenreOpenCount,
			TotalOpenCount: stat.TotalOpenCount,
			Share:          stat.Share,
		}
	}

	return &pb.GenreAgeResponse{GenreSlug: req.GetGenreSlug(), AgeGroups: result}, nil
}

func (h *CatalogHandler) RecordFilmOpen(ctx context.Context, req *pb.RecordFilmOpenRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, missingRequest()
	}

	var accountID *domain.UserID
	if req.AccountId != nil {
		value := domain.UserID(req.GetAccountId())
		accountID = &value
	}

	if err := h.analytics.RecordFilmOpen(ctx, req.GetVisitorPublicId(), accountID, domain.FilmID(req.GetFilmId())); err != nil {
		return nil, analyticsGRPCError(err)
	}

	return &emptypb.Empty{}, nil
}

func analyticsGRPCError(err error) error {
	if errors.Is(err, domain.ErrInvalidAnalyticsPeriod) ||
		errors.Is(err, domain.ErrInvalidAnalyticsLimit) ||
		errors.Is(err, domain.ErrInvalidAnalyticsGenreSlug) ||
		errors.Is(err, domain.ErrInvalidAnalyticsEvent) {
		return status.Error(codes.InvalidArgument, err.Error())
	}

	return toGRPCError(err)
}
