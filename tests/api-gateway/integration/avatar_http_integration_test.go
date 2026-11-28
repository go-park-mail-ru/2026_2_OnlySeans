//go:build integration

package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"image"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	authdelivery "github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/delivery/http/auth"
	userdelivery "github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/delivery/http/user"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/delivery/router"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/grpc_client/userclient"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/middleware"
	userfixture "github.com/go-park-mail-ru/2026_2_OnlySeans/services/user/tests/integrationfixture"
	pb "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/proto/user"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/tests/testutil"
	"github.com/stretchr/testify/require"
)

func TestRustFSAvatarHTTP(t *testing.T) {
	db := testutil.IntegrationDB(t, "AVATAR_TEST_DATABASE_URL")
	objectStorage, _ := testutil.IntegrationS3(t)
	setupCtx, setupCancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer setupCancel()
	previousFilename := strings.ReplaceAll(uuid.NewString(), "-", "") + ".png"
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		require.NoError(t, objectStorage.DeleteFile(cleanupCtx, previousFilename))
	})
	require.NoError(t, objectStorage.UploadFile(setupCtx, previousFilename, "image/png", []byte("previous test avatar")))
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
	} {
		_, err := db.ExecContext(setupCtx, query)
		require.NoError(t, err)
	}
	_, err := db.ExecContext(setupCtx, `INSERT INTO file (storage_key, mime_type, size_bytes, created_by) VALUES ($1, 'image/png', 20, 1)`, previousFilename)
	require.NoError(t, err)
	_, err = db.ExecContext(setupCtx, `UPDATE account SET avatar_file_id = (SELECT id FROM file WHERE storage_key = $1) WHERE id = 1`, previousFilename)
	require.NoError(t, err)
	appLogger := testutil.DiscardLogger()
	rpc := pb.NewUserServiceClient(userfixture.New(t, db, objectStorage))
	users := userclient.NewUserClient(rpc)
	sessions := integrationSessions{}
	handler, err := router.NewRouter(router.Handlers{
		Auth: authdelivery.NewAuthHandler(sessions, users, appLogger), Files: userdelivery.NewAvatarHandler(users, appLogger),
	}, middleware.NewAuthMiddleware(sessions, users, appLogger), nil, appLogger)
	require.NoError(t, err)
	server := httptest.NewServer(handler)
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
	req.AddCookie(&http.Cookie{
		Name:  authdelivery.SessionCookieName,
		Value: "integration-session",
	})
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
		err := objectStorage.DeleteFile(cleanupCtx, created.Filename)
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
	profileRequest.AddCookie(&http.Cookie{
		Name:  authdelivery.SessionCookieName,
		Value: "integration-session",
	})
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
	other, err := users.GetByID(ctx, 2)
	require.NoError(t, err)
	require.Nil(t, other.AvatarFilename)
	var owner int64
	var size int64
	require.NoError(t, db.QueryRowContext(ctx, `SELECT created_by, size_bytes FROM file WHERE storage_key = $1`, created.Filename).Scan(&owner, &size))
	require.Equal(t, int64(1), owner)
	require.Equal(t, int64(imageData.Len()), size)
	t.Log("POST 201 -> PostgreSQL avatar assigned -> profile 200 -> signed download 200; other account unchanged")
}
