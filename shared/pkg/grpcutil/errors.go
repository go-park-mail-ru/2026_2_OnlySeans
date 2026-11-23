package grpcutil

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type StatusCoder interface {
	GRPCStatus() codes.Code
}

type Slugger interface {
	Slug() string
}

func ToGRPCError(err error) error {
	if err == nil {
		return nil
	}

	if _, ok := status.FromError(err); ok {
		return err
	}

	if s, ok := err.(StatusCoder); ok {
		message := err.Error()

		if slugErr, ok := err.(Slugger); ok {
			message = slugErr.Slug()
		}

		return status.Error(s.GRPCStatus(), message)
	}

	return status.Error(codes.Internal, err.Error())
}
