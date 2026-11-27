package grpc

import (
	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/catalog/internal/usecase"
	pb "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/proto/catalog"
)

func parsePagination(input *pb.Pagination) (int, int, error) {
	limit := int64(usecase.DefaultLimit)
	if input != nil && input.Limit != nil {
		limit = input.GetLimit()
	}

	offset := input.GetOffset()

	if limit < int64(usecase.MinLimit) || limit > int64(usecase.MaxLimit) {
		return 0, 0, usecase.ErrInvalidLimit
	}

	if offset < 0 || int64(int(offset)) != offset {
		return 0, 0, usecase.ErrInvalidOffset
	}

	return int(limit), int(offset), nil
}
