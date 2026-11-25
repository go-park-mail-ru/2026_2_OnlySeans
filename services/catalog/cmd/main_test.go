package main

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	cataloggrpc "github.com/go-park-mail-ru/2026_2_OnlySeans/services/catalog/internal/delivery/grpc"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/catalog/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/catalog/internal/usecase"
	pb "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/proto/catalog"
)

type blockingFilms struct {
	usecase.FilmUseCase
	entered            chan struct{}
	release            chan struct{}
	ignoreCancellation bool
}

func (f *blockingFilms) GetFilmByID(ctx context.Context, _ domain.FilmID) (domain.Film, error) {
	close(f.entered)

	if f.ignoreCancellation {
		<-f.release
		return domain.Film{ID: 42}, nil
	}

	select {
	case <-f.release:
		return domain.Film{
			ID: 42,
		}, nil
	case <-ctx.Done():
		return domain.Film{}, ctx.Err()
	}
}

func TestGracefulShutdown(t *testing.T) {
	cases := []struct {
		name  string
		force bool
	}{
		{
			name: "drain active RPC",
		},
		{
			name:  "force stop after timeout",
			force: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			listener := bufconn.Listen(1 << 20)
			films := &blockingFilms{
				entered:            make(chan struct{}),
				release:            make(chan struct{}),
				ignoreCancellation: tc.force,
			}
			server, err := cataloggrpc.NewServer(cataloggrpc.NewCatalogHandler(films, nil, nil, nil, nil, nil), 5<<20)
			require.NoError(t, err)

			defer server.Stop()

			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()

			finished := make(chan error, 1)

			go func() { finished <- serve(ctx, server, listener, 50*time.Millisecond) }()

			conn, err := grpc.NewClient("passthrough:///catalog",
				grpc.WithTransportCredentials(insecure.NewCredentials()),
				grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) { return listener.DialContext(ctx) }),
			)
			require.NoError(t, err)

			defer func() { _ = conn.Close() }()

			rpcCtx, rpcCancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer rpcCancel()

			rpcDone := make(chan error, 1)

			go func() {
				_, err := pb.NewCatalogServiceClient(conn).GetFilm(rpcCtx, &pb.GetFilmRequest{
					FilmId: 42,
				})
				rpcDone <- err
			}()
			select {
			case <-films.entered:
			case <-rpcCtx.Done():
				t.Fatal("RPC did not start")
			}
			cancel()

			if !tc.force {
				close(films.release)
			}
			select {
			case err := <-finished:
				require.NoError(t, err)
			case <-rpcCtx.Done():
				t.Fatal("shutdown did not finish")
			}

			if tc.force {
				close(films.release)
				require.Error(t, <-rpcDone)
			} else {
				require.NoError(t, <-rpcDone)
			}
		})
	}
}
