package reviewsgrpc

import (
	"context"
	"crypto/subtle"
	"time"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
	reviewspb "github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/reviews"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const (
	EnvAddress      = "REVIEWS_GRPC_ADDRESS"
	EnvToken        = "REVIEWS_GRPC_TOKEN"
	tokenHeader     = "authorization"
	requestIDHeader = "x-request-id"
	rpcTimeout      = 5 * time.Second
)

// проверяем, кто передал user_id
// без токена доступно только чтение списка отзывов
func AuthenticateBackend(token string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if token != "" || info.FullMethod != reviewspb.ReviewsService_ListReviews_FullMethodName {
			values := metadata.ValueFromIncomingContext(ctx, tokenHeader)
			if token == "" || len(values) != 1 || subtle.ConstantTimeCompare([]byte(values[0]), []byte("Bearer "+token)) != 1 {
				return nil, status.Error(codes.Unauthenticated, "reviews backend credentials required")
			}
		}
		requestIDs := metadata.ValueFromIncomingContext(ctx, requestIDHeader)
		if len(requestIDs) == 1 {
			ctx = logger.ContextWithRequestID(ctx, requestIDs[0])
		}
		return handler(ctx, req)
	}
}

func AuthorizeRequest(token string) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any, conn *grpc.ClientConn, invoke grpc.UnaryInvoker, options ...grpc.CallOption) error {
		ctx, cancel := context.WithTimeout(ctx, rpcTimeout)
		defer cancel()

		ctx = metadata.AppendToOutgoingContext(ctx, tokenHeader, "Bearer "+token)
		if requestID := logger.RequestID(ctx); requestID != "" {
			ctx = metadata.AppendToOutgoingContext(ctx, requestIDHeader, requestID)
		}
		return invoke(ctx, method, req, reply, conn, options...)
	}
}
