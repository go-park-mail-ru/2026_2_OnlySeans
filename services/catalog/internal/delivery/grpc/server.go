package grpc

import (
	"errors"

	"google.golang.org/grpc"

	pb "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/proto/catalog"
)

const messageSizeOverhead = 1 << 20

func NewServer(handler *CatalogHandler, maxPosterSize int64, opts ...grpc.ServerOption) (*grpc.Server, error) {
	if handler == nil {
		return nil, errors.New("catalog handler is required")
	}

	maxInt := int64(int(^uint(0) >> 1))
	if maxPosterSize <= 0 || maxPosterSize > maxInt-messageSizeOverhead {
		return nil, errors.New("poster size must be positive and leave room for gRPC message overhead")
	}

	opts = append(opts, grpc.MaxRecvMsgSize(int(maxPosterSize)+messageSizeOverhead))
	server := grpc.NewServer(opts...)
	pb.RegisterCatalogServiceServer(server, handler)

	return server, nil
}
