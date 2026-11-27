package grpc

import (
	"context"
	"errors"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/catalog/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/catalog/internal/usecase"
	pb "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/proto/catalog"
)

type ReleaseHandler struct {
	pb.UnimplementedReleaseServiceServer
	releases *usecase.ReleaseUseCase
}

func NewReleaseHandler(releases *usecase.ReleaseUseCase) *ReleaseHandler {
	return &ReleaseHandler{releases: releases}
}

func releaseError(err error) error {
	switch {
	case errors.Is(err, domain.ErrInvalidRelease):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, domain.ErrNotificationNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, domain.ErrReleaseUnavailable):
		return status.Error(codes.FailedPrecondition, err.Error())
	default:
		return toGRPCError(err)
	}
}

func (h *ReleaseHandler) ListReleases(ctx context.Context, req *pb.ListReleasesRequest) (*pb.ReleaseList, error) {
	if req == nil {
		return nil, missingRequest()
	}

	f := domain.ReleaseFilter{FilmID: domain.FilmID(req.FilmId), GenreID: domain.GenreID(req.GenreId), Limit: int(req.Limit), Offset: int(req.Offset), Featured: req.Featured}
	var err error

	if req.From != "" {
		f.From, err = domain.ParseReleaseDate(req.From)
		if err != nil {
			return nil, releaseError(err)
		}
	}

	if req.To != "" {
		f.To, err = domain.ParseReleaseDate(req.To)
		if err != nil {
			return nil, releaseError(err)
		}
	}

	items, total, err := h.releases.List(ctx, f, 0)
	if err != nil {
		return nil, releaseError(err)
	}

	return protoReleases(items, total), nil
}

func (h *ReleaseHandler) UpdateRelease(ctx context.Context, req *pb.UpdateReleaseRequest) (*emptypb.Empty, error) {
	if req == nil || req.Input == nil {
		return nil, missingRequest()
	}

	if err := h.releases.Update(ctx, req.FilmId, releaseInput(req.Input)); err != nil {
		return nil, releaseError(err)
	}

	return &emptypb.Empty{}, nil
}

func (h *ReleaseHandler) SetReleaseSubscription(ctx context.Context, req *pb.SetReleaseSubscriptionRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, missingRequest()
	}

	if err := h.releases.SetSubscription(ctx, req.UserId, req.FilmId, req.Subscribed); err != nil {
		return nil, releaseError(err)
	}

	return &emptypb.Empty{}, nil
}

func (h *ReleaseHandler) ListReleaseSubscriptions(ctx context.Context, req *pb.ReleaseUserRequest) (*pb.ReleaseList, error) {
	if req == nil {
		return nil, missingRequest()
	}

	if req.UserId <= 0 {
		return nil, releaseError(domain.ErrUnauthorized)
	}

	items, total, err := h.releases.List(ctx, domain.ReleaseFilter{Limit: int(req.Limit), Offset: int(req.Offset)}, req.UserId)
	if err != nil {
		return nil, releaseError(err)
	}

	return protoReleases(items, total), nil
}

func (h *ReleaseHandler) ListNotifications(ctx context.Context, req *pb.ReleaseUserRequest) (*pb.NotificationList, error) {
	if req == nil {
		return nil, missingRequest()
	}

	items, total, err := h.releases.Notifications(ctx, req.UserId, int(req.Limit), int(req.Offset))
	if err != nil {
		return nil, releaseError(err)
	}

	result := &pb.NotificationList{Total: total, Items: make([]*pb.Notification, len(items))}
	for i, n := range items {
		result.Items[i] = &pb.Notification{Id: n.ID, Message: n.Message, IsRead: n.IsRead, CreatedAt: timestamppb.New(n.CreatedAt), FilmId: n.FilmID, NotificationType: n.Type}
	}

	return result, nil
}

func (h *ReleaseHandler) ReadNotification(ctx context.Context, req *pb.ReadNotificationRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, missingRequest()
	}

	if err := h.releases.ReadNotification(ctx, req.UserId, req.NotificationId); err != nil {
		return nil, releaseError(err)
	}

	return &emptypb.Empty{}, nil
}

func releaseInput(in *pb.ReleaseInput) domain.ReleaseInput {
	return domain.ReleaseInput{
		WorldPremiereDate: in.WorldPremiereDate, RussiaPremiereDate: in.RussiaPremiereDate,
		WorldPremiereDateSet: in.WorldPremiereDateSet, RussiaPremiereDateSet: in.RussiaPremiereDateSet,
	}
}

func premiereDate(date *time.Time) *string {
	if date == nil {
		return nil
	}

	value := date.Format(time.DateOnly)

	return &value
}

func protoReleases(items []domain.Release, total int64) *pb.ReleaseList {
	result := &pb.ReleaseList{Total: total, Items: make([]*pb.Release, len(items))}
	for i, event := range items {
		result.Items[i] = &pb.Release{Film: toProtoFilm(event.Film),
			WorldPremiereDate: premiereDate(event.WorldPremiereDate), RussiaPremiereDate: premiereDate(event.RussiaPremiereDate)}
	}

	return result
}
