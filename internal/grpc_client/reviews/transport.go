package reviewsclient

import (
	"context"
	"time"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

const (
	EnvAddress      = "REVIEWS_GRPC_ADDRESS"
	EnvToken        = "REVIEWS_GRPC_TOKEN"
	tokenHeader     = "authorization"
	requestIDHeader = "x-request-id"
	rpcTimeout      = 5 * time.Second
)

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
