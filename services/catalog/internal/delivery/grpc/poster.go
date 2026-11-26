package grpc

import (
	"bytes"
	"context"

	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/catalog/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/catalog/internal/usecase"
	pb "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/proto/catalog"
)

func (h *CatalogHandler) UploadPoster(ctx context.Context, req *pb.UploadPosterRequest) (*pb.UploadPosterResponse, error) {
	if req == nil {
		return nil, missingRequest()
	}

	data := req.GetImageData()

	filename, err := h.posters.UploadPoster(ctx, usecase.UploadPosterInput{
		UserID: domain.UserID(req.GetUserId()),
		FilmID: domain.FilmID(req.GetFilmId()),
		Body:   bytes.NewReader(data),
		Size:   int64(len(data)),
	})
	if err != nil {
		return nil, toGRPCError(err)
	}

	return &pb.UploadPosterResponse{
		Filename: filename,
	}, nil
}

func (h *CatalogHandler) GetPosterURL(ctx context.Context, req *pb.GetPosterURLRequest) (*pb.GetPosterURLResponse, error) {
	if req == nil {
		return nil, missingRequest()
	}

	url, err := h.posters.GetPosterURL(ctx, domain.FilmID(req.GetFilmId()))
	if err != nil {
		return nil, toGRPCError(err)
	}

	return &pb.GetPosterURLResponse{
		Url: url,
	}, nil
}

func (h *CatalogHandler) DeletePoster(ctx context.Context, req *pb.DeletePosterRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, missingRequest()
	}

	if err := h.posters.DeletePoster(ctx, domain.FilmID(req.GetFilmId())); err != nil {
		return nil, toGRPCError(err)
	}

	return &emptypb.Empty{}, nil
}
