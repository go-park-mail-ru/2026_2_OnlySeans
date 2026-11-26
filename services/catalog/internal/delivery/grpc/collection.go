package grpc

import (
	"context"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/catalog/internal/domain"
	pb "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/proto/catalog"
)

func (h *CatalogHandler) ListCollections(ctx context.Context, req *pb.ListCollectionsRequest) (*pb.ListCollectionsResponse, error) {
	if req == nil {
		return nil, missingRequest()
	}

	limit, offset, err := parsePagination(req.GetPagination())
	if err != nil {
		return nil, toGRPCError(err)
	}

	collections, total, err := h.collections.ListCollections(ctx, limit, offset)
	if err != nil {
		return nil, toGRPCError(err)
	}

	result := make([]*pb.Collection, len(collections))
	for i, collection := range collections {
		result[i] = toProtoCollection(collection)
	}

	return &pb.ListCollectionsResponse{
		Collections: result,
		Total:       int64(total),
	}, nil
}

func (h *CatalogHandler) GetCollection(ctx context.Context, req *pb.GetCollectionRequest) (*pb.GetCollectionResponse, error) {
	if req == nil {
		return nil, missingRequest()
	}

	limit, offset, err := parsePagination(req.GetPagination())
	if err != nil {
		return nil, toGRPCError(err)
	}

	result, err := h.collections.GetCollection(ctx, domain.CollectionID(req.GetCollectionId()), limit, offset)
	if err != nil {
		return nil, toGRPCError(err)
	}

	return &pb.GetCollectionResponse{
		Collection: toProtoCollection(result.Collection),
		Films:      toProtoFilms(result.Films),
		Total:      int64(result.Total),
	}, nil
}

func toProtoCollection(collection domain.Collection) *pb.Collection {
	return &pb.Collection{
		Id:          int64(collection.ID),
		Title:       collection.Title,
		Slug:        collection.Slug,
		Description: collection.Description,
		CoverUrl:    collection.CoverURL,
	}
}
