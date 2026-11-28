//go:build integration

package testutil

import (
	"context"
	"database/sql"
	"net"
	"os"
	"testing"
	"time"

	shareds3 "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/s3"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

// IntegrationDB pins temporary tables to one connection and never modifies application tables.
func IntegrationDB(t *testing.T, env string) *sql.DB {
	t.Helper()
	dsn := os.Getenv(env)
	if dsn == "" {
		t.Skip("set " + env + " for PostgreSQL integration")
	}
	db, err := sql.Open("pgx", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	return db
}

func IntegrationS3(t *testing.T) (*shareds3.S3Storage, string) {
	t.Helper()
	endpoint := os.Getenv("INTEGRATION_S3_ENDPOINT")
	bucket := os.Getenv("INTEGRATION_S3_BUCKET")
	access := os.Getenv("RUSTFS_ACCESS_KEY")
	secret := os.Getenv("RUSTFS_SECRET_KEY")
	if endpoint == "" || bucket == "" || access == "" || secret == "" {
		t.Skip("set INTEGRATION_S3_ENDPOINT, INTEGRATION_S3_BUCKET, RUSTFS_ACCESS_KEY and RUSTFS_SECRET_KEY for object storage integration")
	}
	client, err := shareds3.NewS3Client(shareds3.ClientConfig{Endpoint: endpoint, Region: "us-east-1", AccessKey: access, SecretKey: secret})
	require.NoError(t, err)
	storage, err := shareds3.NewS3Storage(client, bucket, endpoint, 15*time.Minute, DiscardLogger())
	require.NoError(t, err)
	return storage, bucket
}

func IntegrationGRPC(t *testing.T, server *grpc.Server) *grpc.ClientConn {
	t.Helper()
	listener := bufconn.Listen(1 << 20)
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() { server.Stop(); require.NoError(t, listener.Close()) })
	conn, err := grpc.NewClient("passthrough:///integration", grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, conn.Close()) })
	return conn
}
