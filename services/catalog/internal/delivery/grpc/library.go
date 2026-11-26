package grpc

import (
	"context"

	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/catalog/internal/domain"
	pb "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/proto/catalog"
)

func (h *CatalogHandler) GetLibrary(ctx context.Context, req *pb.GetLibraryRequest) (*pb.GetLibraryResponse, error) {
	if req == nil {
		return nil, missingRequest()
	}

	library, err := h.library.GetLibrary(ctx, domain.UserID(req.GetUserId()))
	if err != nil {
		return nil, toGRPCError(err)
	}

	return &pb.GetLibraryResponse{
		Library: &pb.Library{
			Favorite:   filmIDs(library.Favorite),
			Watchlater: filmIDs(library.WatchLater),
			Watched:    filmIDs(library.Watched),
		},
	}, nil
}

func (h *CatalogHandler) SetSaved(ctx context.Context, req *pb.SetSavedRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, missingRequest()
	}

	if err := h.library.SetSaved(ctx, domain.UserID(req.GetUserId()), domain.List(req.GetList()), domain.FilmID(req.GetFilmId()), req.GetSaved()); err != nil {
		return nil, toGRPCError(err)
	}

	return &emptypb.Empty{}, nil
}

func filmIDs(ids []domain.FilmID) []int64 {
	result := make([]int64, len(ids))
	for i, id := range ids {
		result[i] = int64(id)
	}

	return result
}
