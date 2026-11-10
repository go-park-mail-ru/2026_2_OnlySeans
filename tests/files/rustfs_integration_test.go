//go:build integration

package files_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"image"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/config"
	authdelivery "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/auth"
	filesdelivery "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/files"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/router"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	authservice "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/services/auth"
	filesservice "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/services/files"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/storage"
	authusecase "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/auth"
	filesusecase "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/files"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
)

type integrationAuthenticator struct{}

func (integrationAuthenticator) Authenticate(_ context.Context, session string) (domain.UserID, error) {
	if session != "integration-session" {
		return 0, domain.ErrUnauthorized
	}
	return 1, nil
}

// This test sends real HTTP requests through the handlers into RustFS.
// It uses a test session and removes only the object it creates.
func TestRustFSAvatarHTTP(t *testing.T) {
	cfg, err := config.LoadS3("../../configs/s3.yaml")
	require.NoError(t, err)
	client, err := storage.NewS3Client(*cfg)
	require.NoError(t, err)
	objectStorage, err := filesservice.NewS3Storage(client, cfg.Bucket, cfg.PublicEndpoint, cfg.AvatarURLLifetime)
	require.NoError(t, err)
	db, err := sql.Open("pgx", os.Getenv("AVATAR_TEST_DATABASE_URL"))
	require.NoError(t, err)
	defer db.Close()
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	setupCtx, setupCancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer setupCancel()
	for _, query := range []string{
		`CREATE TEMP TABLE role (id bigint PRIMARY KEY, code text NOT NULL)`,
		`INSERT INTO role VALUES (1, 'reader')`,
		`CREATE TEMP TABLE account (
   id bigint PRIMARY KEY, role_id bigint REFERENCES role(id), email text, username text,
   password_hash text, deleted_at timestamptz, avatar_file_id bigint
  )`,
		`INSERT INTO account (id, role_id, email, username, password_hash) VALUES
   (1, 1, 'avatar@example.com', 'avataruser', 'hash'),
   (2, 1, 'other@example.com', 'otheruser', 'hash')`,
		`CREATE TEMP TABLE file (
   id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
   storage_key text UNIQUE, external_url text, mime_type text NOT NULL,
   size_bytes bigint CHECK (size_bytes > 0), created_by bigint REFERENCES account(id),
   CHECK (num_nonnulls(storage_key, external_url) = 1),
   CHECK (storage_key IS NULL OR size_bytes IS NOT NULL)
  )`,
		`ALTER TABLE account ADD FOREIGN KEY (avatar_file_id) REFERENCES file(id)`,
		`INSERT INTO file (storage_key, mime_type, size_bytes, created_by) VALUES ('previous.png', 'image/png', 1, 1)`,
		`UPDATE account SET avatar_file_id = (SELECT id FROM file WHERE storage_key = 'previous.png') WHERE id = 1`,
	} {
		_, err = db.ExecContext(setupCtx, query)
		require.NoError(t, err)
	}
	avatarRepo, err := filesservice.NewPostgresAvatarRepo(db, nil)
	require.NoError(t, err)
	upload, err := filesusecase.NewUploadAvatarUseCase(objectStorage, avatarRepo, filesusecase.UploadAvatarOptions{MaxSize: cfg.MaxAvatarSize, MaxDimension: cfg.MaxAvatarDimension, CleanupTimeout: cfg.AvatarCleanupTimeout}, testFilesLogger(t))
	require.NoError(t, err)
	getURL, err := filesusecase.NewGetAvatarURLUseCase(objectStorage, testFilesLogger(t))
	require.NoError(t, err)
	appLogger, err := logger.NewLogger(io.Discard, logger.LogLevelInfo, "text")
	require.NoError(t, err)
	accountAvatar, err := filesusecase.NewAccountAvatarUseCase(avatarRepo, objectStorage, testFilesLogger(t))
	require.NoError(t, err)
	handler, err := filesdelivery.NewHandler(upload, getURL, accountAvatar, appLogger)
	require.NoError(t, err)
	routes := mux.NewRouter()
	routes.HandleFunc("/api/avatars", router.RequireAuth(integrationAuthenticator{})(handler.UploadAvatar)).Methods(http.MethodPost)
	routes.HandleFunc("/api/avatars/{filename}", handler.GetAvatarURL).Methods(http.MethodGet)
	userRepo := authservice.NewPostgresUserRepo(db, nil)
	sessions := authservice.NewInMemorySessionStore(time.Hour)
	passwords := authservice.BcryptPasswordService{}
	registerUC, err := authusecase.NewRegisterUserUseCase(userRepo, sessions, passwords, appLogger)
	require.NoError(t, err)
	loginUC, err := authusecase.NewLoginUserUseCase(userRepo, sessions, passwords, appLogger)
	require.NoError(t, err)
	logoutUC, err := authusecase.NewLogoutUserUseCase(sessions, appLogger)
	require.NoError(t, err)
	authenticateUC, err := authusecase.NewAuthenticateUserUseCase(sessions, appLogger)
	require.NoError(t, err)
	getUserUC, err := authusecase.NewGetUserUseCase(userRepo, appLogger)
	require.NoError(t, err)
	authHandler, err := authdelivery.NewHandler(authdelivery.HandlerUseCases{
		Register: registerUC, Login: loginUC, Logout: logoutUC, Authenticate: authenticateUC, GetUser: getUserUC,
	}, appLogger)
	require.NoError(t, err)
	routes.HandleFunc("/api/authorised", router.RequireAuth(integrationAuthenticator{})(authHandler.Authorised)).Methods(http.MethodGet)
	server := httptest.NewServer(routes)
	defer server.Close()
	httpClient := &http.Client{Timeout: 10 * time.Second}

	var imageData bytes.Buffer
	require.NoError(t, png.Encode(&imageData, image.NewRGBA(image.Rect(0, 0, 2, 2))))
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("avatar", "integration.png")
	require.NoError(t, err)
	_, err = part.Write(imageData.Bytes())
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, server.URL+"/api/avatars", &body)
	require.NoError(t, err)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.AddCookie(&http.Cookie{Name: authdelivery.SessionCookieName, Value: "integration-session"})
	response, err := httpClient.Do(req)
	require.NoError(t, err)
	defer response.Body.Close()
	var created struct {
		Filename string `json:"filename"`
	}
	require.NoError(t, json.NewDecoder(response.Body).Decode(&created))
	require.Equal(t, http.StatusCreated, response.StatusCode)
	require.Regexp(t, `^[0-9a-f]{32}\.png$`, created.Filename)
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		_, err := client.DeleteObject(cleanupCtx, &s3.DeleteObjectInput{Bucket: aws.String(cfg.Bucket), Key: aws.String(created.Filename)})
		require.NoError(t, err, "remove test object")
	})

	response, err = httpClient.Get(server.URL + "/api/avatars/" + created.Filename)
	require.NoError(t, err)
	defer response.Body.Close()
	require.Equal(t, http.StatusOK, response.StatusCode)
	var link struct {
		URL string `json:"url"`
	}
	require.NoError(t, json.NewDecoder(response.Body).Decode(&link))
	download, err := httpClient.Get(link.URL)
	// Avoid printing the signed URL on an error.
	if err != nil {
		t.Fatal("download using signed URL failed")
	}
	defer download.Body.Close()
	require.Equal(t, http.StatusOK, download.StatusCode)
	require.Equal(t, "image/png", download.Header.Get("Content-Type"))
	actual, err := io.ReadAll(download.Body)
	require.NoError(t, err)
	require.Equal(t, imageData.Bytes(), actual)
	profileRequest, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/api/authorised", nil)
	require.NoError(t, err)
	profileRequest.AddCookie(&http.Cookie{Name: authdelivery.SessionCookieName, Value: "integration-session"})
	profileResponse, err := httpClient.Do(profileRequest)
	require.NoError(t, err)
	defer profileResponse.Body.Close()
	require.Equal(t, http.StatusOK, profileResponse.StatusCode)
	var profile struct {
		User struct {
			AvatarFilename string `json:"avatar_filename"`
		} `json:"user"`
	}
	require.NoError(t, json.NewDecoder(profileResponse.Body).Decode(&profile))
	require.Equal(t, created.Filename, profile.User.AvatarFilename)
	other, err := userRepo.GetUser(ctx, 2)
	require.NoError(t, err)
	require.Nil(t, other.AvatarFilename)
	var owner domain.UserID
	var size int64
	require.NoError(t, db.QueryRowContext(ctx, `SELECT created_by, size_bytes FROM file WHERE storage_key = $1`, created.Filename).Scan(&owner, &size))
	require.Equal(t, domain.UserID(1), owner)
	require.Equal(t, int64(imageData.Len()), size)
	t.Log("POST 201 -> PostgreSQL avatar assigned -> profile 200 -> signed download 200; other account unchanged")
}
