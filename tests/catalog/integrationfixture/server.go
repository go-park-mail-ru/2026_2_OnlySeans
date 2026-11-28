//go:build integration

package integrationfixture

import (
	"database/sql"
	delivery "github.com/go-park-mail-ru/2026_2_OnlySeans/services/catalog/internal/delivery/grpc"
	postgres "github.com/go-park-mail-ru/2026_2_OnlySeans/services/catalog/internal/repository/postgres"
	posters3 "github.com/go-park-mail-ru/2026_2_OnlySeans/services/catalog/internal/repository/s3"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/catalog/internal/usecase"
	shareds3 "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/s3"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/tests/testutil"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"testing"
	"time"
)

// New starts the real catalog service on an in-memory gRPC transport over temporary SQL tables.
func New(t *testing.T, db *sql.DB, storage *shareds3.S3Storage) *grpc.ClientConn {
	t.Helper()
	log := testutil.DiscardLogger()
	films := postgres.NewFilmRepo(db, log)
	reader, err := usecase.NewPosterURLReader(films, films, films, storage)
	require.NoError(t, err)
	repo, err := postgres.NewPosterRepo(db, log)
	require.NoError(t, err)
	objects, err := posters3.NewPosterStorage(storage)
	require.NoError(t, err)
	posters, err := usecase.NewPosterUseCase(repo, films, objects, usecase.PosterOptions{
		MaxSize:        5 << 20,
		MaxDimension:   4096,
		CleanupTimeout: 5 * time.Second,
	}, log)
	require.NoError(t, err)
	handler := delivery.NewCatalogHandler(usecase.NewFilmUseCase(reader, reader, films), usecase.NewCollectionUseCase(reader), nil, nil, nil, posters)
	server, err := delivery.NewServer(handler, 5<<20)
	require.NoError(t, err)
	return testutil.IntegrationGRPC(t, server)
}
