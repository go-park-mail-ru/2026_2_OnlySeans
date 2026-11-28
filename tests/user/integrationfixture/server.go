//go:build integration

package integrationfixture

import (
	"database/sql"
	delivery "github.com/go-park-mail-ru/2026_2_OnlySeans/services/user/internal/delivery/grpc"
	postgres "github.com/go-park-mail-ru/2026_2_OnlySeans/services/user/internal/repository/postgres"
	users3 "github.com/go-park-mail-ru/2026_2_OnlySeans/services/user/internal/repository/s3"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/user/internal/usecase"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/password"
	shareds3 "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/s3"
	pb "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/proto/user"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/tests/testutil"
	"google.golang.org/grpc"
	"testing"
)

func New(t *testing.T, db *sql.DB, storage *shareds3.S3Storage) *grpc.ClientConn {
	t.Helper()
	files := users3.NewFileStorage(storage)
	users := usecase.NewUserUseCase(postgres.NewPostgresUserRepo(db, testutil.DiscardLogger()), postgres.NewPostgresAvatarRepo(db), files, &password.BcryptPasswordService{}, testutil.DiscardLogger())
	server := grpc.NewServer(grpc.MaxRecvMsgSize(6 << 20))
	pb.RegisterUserServiceServer(server, delivery.NewUserHandler(users, nil))
	return testutil.IntegrationGRPC(t, server)
}
