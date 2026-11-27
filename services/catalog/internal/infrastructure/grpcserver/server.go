package grpcserver

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	grpcgo "google.golang.org/grpc"
)

func Serve(ctx context.Context, server *grpcgo.Server, listener net.Listener, shutdownTimeout time.Duration) error {
	serverErr := make(chan error, 1)
	go func() { serverErr <- server.Serve(listener) }()
	select {
	case err := <-serverErr:
		if err != nil && !errors.Is(err, grpcgo.ErrServerStopped) {
			return fmt.Errorf("serve catalog gRPC: %w", err)
		}

		return nil
	case <-ctx.Done():
	}

	stopped := make(chan struct{})

	go func() {
		server.GracefulStop()
		close(stopped)
	}()

	timer := time.NewTimer(shutdownTimeout)
	defer timer.Stop()
	select {
	case <-stopped:
	case <-timer.C:
		server.Stop()
	}

	if err := <-serverErr; err != nil && !errors.Is(err, grpcgo.ErrServerStopped) {
		return fmt.Errorf("serve catalog gRPC: %w", err)
	}

	return nil
}
