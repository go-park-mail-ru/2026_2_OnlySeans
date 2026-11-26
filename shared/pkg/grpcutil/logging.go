package grpcutil

import (
	"context"
	"crypto/rand"
	"fmt"
	"time"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/logger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const RequestIDMetadataKey = "x-request-id"

func validRequestID(id string) bool {
	if len(id) == 0 || len(id) > 128 {
		return false
	}
	for _, c := range id {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-' || c == '_' || c == '.') {
			return false
		}
	}
	return true
}

func UnaryClientRequestID(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	id := logger.RequestID(ctx)
	if !validRequestID(id) {
		id = rand.Text()
	}
	md, _ := metadata.FromOutgoingContext(ctx)
	md = md.Copy()
	md.Set(RequestIDMetadataKey, id)
	return invoker(metadata.NewOutgoingContext(ctx, md), method, req, reply, cc, opts...)
}

func UnaryServerRequestID(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	values := md.Get(RequestIDMetadataKey)
	id := ""
	if len(values) == 1 && validRequestID(values[0]) {
		id = values[0]
	}
	if id == "" {
		id = rand.Text()
	}
	return handler(logger.ContextWithRequestID(ctx, id), req)
}

func ErrorFields(err error) map[string]any {
	fields := map[string]any{"error_type": fmt.Sprintf("%T", err)}
	if st, ok := status.FromError(err); ok {
		fields["grpc_code"] = st.Code().String()
		fields["error"] = "RPC failed: " + st.Code().String()
	} else if err != nil {
		fields["error"] = err.Error()
	}
	return fields
}

func UnaryServerLogging(service string, appLogger logger.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		started := time.Now()
		response, err := handler(ctx, req)
		if appLogger == nil {
			return response, err
		}
		code := status.Code(err)
		fields := map[string]any{
			"service": service, "method": info.FullMethod, "grpc_code": code.String(),
			"duration_ms": float64(time.Since(started).Microseconds()) / 1000,
		}
		switch code {
		case codes.OK:
			appLogger.InfoContext(ctx, "gRPC request handled", fields)
		case codes.NotFound, codes.Canceled:
			appLogger.DebugContext(ctx, "gRPC request handled", fields)
		case codes.InvalidArgument, codes.AlreadyExists, codes.PermissionDenied, codes.Unauthenticated, codes.FailedPrecondition, codes.OutOfRange:
			appLogger.WarnContext(ctx, "gRPC request handled", fields)
		default:
			appLogger.ErrorContext(ctx, "gRPC request handled", fields)
		}
		return response, err
	}
}
