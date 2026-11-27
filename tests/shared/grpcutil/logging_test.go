package grpcutil_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/grpcutil"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/logger"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	health "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

type healthServer struct {
	health.UnimplementedHealthServer
	check func(context.Context) (*health.HealthCheckResponse, error)
}

func (s healthServer) Check(ctx context.Context, _ *health.HealthCheckRequest) (*health.HealthCheckResponse, error) {
	return s.check(ctx)
}

func startServer(t *testing.T, name string, output *bytes.Buffer, check func(context.Context) (*health.HealthCheckResponse, error)) health.HealthClient {
	t.Helper()
	l, err := logger.NewLogger(output, logger.LogLevelDebug, "json")
	require.NoError(t, err)
	listener := bufconn.Listen(1 << 20)
	server := grpc.NewServer(grpc.ChainUnaryInterceptor(grpcutil.UnaryServerRequestID, grpcutil.UnaryServerLogging(name, l)))
	health.RegisterHealthServer(server, healthServer{check: check})
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() { server.Stop(); _ = listener.Close() })
	conn, err := grpc.NewClient("passthrough:///bufnet", grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }), grpc.WithChainUnaryInterceptor(grpcutil.UnaryClientRequestID))
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
	return health.NewHealthClient(conn)
}

func TestRequestIDAcrossTwoGRPCHops(t *testing.T) {
	var authLogs, userLogs bytes.Buffer
	ids := make(chan string, 2)
	metadataValues := make(chan []string, 1)
	users := startServer(t, "user", &userLogs, func(ctx context.Context) (*health.HealthCheckResponse, error) {
		ids <- logger.RequestID(ctx)
		md, _ := metadata.FromIncomingContext(ctx)
		metadataValues <- md.Get("test-metadata")
		return &health.HealthCheckResponse{Status: health.HealthCheckResponse_SERVING}, nil
	})
	auth := startServer(t, "auth", &authLogs, func(ctx context.Context) (*health.HealthCheckResponse, error) {
		ids <- logger.RequestID(ctx)
		return users.Check(metadata.AppendToOutgoingContext(ctx, "test-metadata", "preserved"), &health.HealthCheckRequest{})
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ctx = logger.ContextWithRequestID(ctx, "http-request-42")
	original := metadata.Pairs(grpcutil.RequestIDMetadataKey, "stale-id", "keep", "unchanged")
	ctx = metadata.NewOutgoingContext(ctx, original)
	_, err := auth.Check(ctx, &health.HealthCheckRequest{})
	require.NoError(t, err)
	require.Equal(t, "http-request-42", <-ids)
	require.Equal(t, "http-request-42", <-ids)
	require.Equal(t, []string{"preserved"}, <-metadataValues)
	require.Equal(t, []string{"stale-id"}, original.Get(grpcutil.RequestIDMetadataKey))
	for name, output := range map[string]*bytes.Buffer{"auth": &authLogs, "user": &userLogs} {
		var record map[string]any
		require.NoError(t, json.Unmarshal(output.Bytes(), &record))
		require.Equal(t, name, record["service"])
		require.Equal(t, "http-request-42", record["request_id"])
		require.Equal(t, "OK", record["grpc_code"])
		require.Equal(t, "/grpc.health.v1.Health/Check", record["method"])
		require.GreaterOrEqual(t, record["duration_ms"], float64(0))
	}
}

func TestIncomingRequestIDValidation(t *testing.T) {
	for _, values := range [][]string{nil, {"valid-ID_42.1"}, {""}, {"bad\nid"}, {strings.Repeat("x", 129)}, {"one", "two"}} {
		t.Run(fmt.Sprint(values), func(t *testing.T) {
			ctx := metadata.NewIncomingContext(context.Background(), metadata.MD{grpcutil.RequestIDMetadataKey: values})
			_, err := grpcutil.UnaryServerRequestID(ctx, nil, &grpc.UnaryServerInfo{}, func(ctx context.Context, _ any) (any, error) {
				id := logger.RequestID(ctx)
				require.Regexp(t, `^[A-Za-z0-9_.-]{1,128}$`, id)
				if len(values) == 1 && values[0] == "valid-ID_42.1" {
					require.Equal(t, values[0], id)
				}
				return nil, nil
			})
			require.NoError(t, err)
		})
	}
}

func TestGRPCLogLevelsAndPrivacy(t *testing.T) {
	for _, tc := range []struct {
		code  codes.Code
		level string
	}{
		{codes.OK, "INFO"}, {codes.NotFound, "DEBUG"}, {codes.Canceled, "DEBUG"},
		{codes.InvalidArgument, "WARN"}, {codes.Unauthenticated, "WARN"}, {codes.PermissionDenied, "WARN"},
		{codes.Internal, "ERROR"}, {codes.Unavailable, "ERROR"}, {codes.DeadlineExceeded, "ERROR"},
	} {
		t.Run(tc.code.String(), func(t *testing.T) {
			var output bytes.Buffer
			l, err := logger.NewLogger(&output, logger.LogLevelDebug, "json")
			require.NoError(t, err)
			failure := status.Error(tc.code, "secret-from-request")
			ctx := logger.ContextWithRequestID(context.Background(), "id-42")
			response, actualErr := grpcutil.UnaryServerLogging("auth", l)(ctx, "secret-request-body", &grpc.UnaryServerInfo{FullMethod: "/auth/Login"}, func(context.Context, any) (any, error) { return "secret-response-body", failure })
			require.Equal(t, "secret-response-body", response)
			require.Equal(t, failure, actualErr)
			var record map[string]any
			require.NoError(t, json.Unmarshal(output.Bytes(), &record))
			require.Equal(t, tc.level, record["level"])
			require.Equal(t, tc.code.String(), record["grpc_code"])
			require.Equal(t, "id-42", record["request_id"])
			require.NotContains(t, output.String(), "secret")
		})
	}
	fields := grpcutil.ErrorFields(fmt.Errorf("wrapped: %w", status.Error(codes.Unavailable, "secret-token")))
	require.Equal(t, "Unavailable", fields["grpc_code"])
	require.NotContains(t, fmt.Sprint(fields), "secret-token")
}
