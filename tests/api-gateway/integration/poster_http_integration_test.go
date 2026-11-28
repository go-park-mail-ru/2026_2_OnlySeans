//go:build integration

package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	authdelivery "github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/delivery/http/auth"
	filesdelivery "github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/delivery/http/files"
	filmsdelivery "github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/delivery/http/films"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/delivery/router"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/grpc_client/catalogclient"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/api-gateway/internal/middleware"
	catalogfixture "github.com/go-park-mail-ru/2026_2_OnlySeans/services/catalog/tests/integrationfixture"
	pb "github.com/go-park-mail-ru/2026_2_OnlySeans/shared/proto/catalog"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/tests/testutil"
	"github.com/stretchr/testify/require"
)

func TestRustFSPosterHTTP(t *testing.T) {
	db := testutil.IntegrationDB(t, "POSTERS_TEST_DATABASE_URL")
	objects, bucket := testutil.IntegrationS3(t)
	ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
	defer cancel()
	for _, query := range []string{
		`CREATE TEMP TABLE account (id bigint PRIMARY KEY)`,
		`INSERT INTO account VALUES (1)`,
		`CREATE TEMP TABLE film (id bigint PRIMARY KEY, title text, original_title text, film_type text, production_year smallint, duration_min smallint, age_limit smallint, description text)`,
		`INSERT INTO film VALUES (9, 'Poster integration film', NULL, 'movie', 2020, NULL, 0, NULL)`,
		`CREATE TEMP TABLE file (
   id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
   storage_key text UNIQUE, external_url text, mime_type text NOT NULL,
   size_bytes bigint CHECK (size_bytes > 0), created_by bigint REFERENCES account(id),
   CHECK (num_nonnulls(storage_key, external_url) = 1)
  )`,
		`CREATE TEMP TABLE film_file (
   film_id bigint REFERENCES film(id), file_id bigint REFERENCES file(id),
   file_role text NOT NULL, sort_order smallint NOT NULL CHECK (sort_order > 0),
   created_at timestamptz NOT NULL DEFAULT now(),
   PRIMARY KEY (film_id, file_id), UNIQUE (film_id, file_role, sort_order)
  )`,
		`CREATE TEMP TABLE genre (id bigint PRIMARY KEY, name text, slug text)`,
		`CREATE TEMP TABLE film_genre (film_id bigint, genre_id bigint)`,
	} {
		_, err := db.ExecContext(ctx, query)
		require.NoError(t, err)
	}
	appLogger := testutil.DiscardLogger()
	client := catalogclient.NewCatalogClient(pb.NewCatalogServiceClient(catalogfixture.New(t, db, objects)))
	uploadHandler, err := filesdelivery.NewPosterHandler(client, 5<<20, appLogger)
	require.NoError(t, err)
	readHandler, err := filmsdelivery.NewHandler(client, appLogger)
	require.NoError(t, err)
	users := &integrationUsers{role: "reader"}
	sessions := integrationSessions{}
	handler, err := router.NewRouter(router.Handlers{
		Auth: authdelivery.NewAuthHandler(sessions, users, appLogger), Films: readHandler, Posters: uploadHandler,
	}, middleware.NewAuthMiddleware(sessions, users, appLogger), nil, appLogger)
	require.NoError(t, err)
	server := httptest.NewServer(handler)
	defer server.Close()
	httpClient := &http.Client{Timeout: 10 * time.Second}
	var pngData, jpegData bytes.Buffer
	picture := image.NewRGBA(image.Rect(0, 0, 2, 2))
	require.NoError(t, png.Encode(&pngData, picture))
	require.NoError(t, jpeg.Encode(&jpegData, picture, nil))
	tests := []struct {
		name          string
		authenticated bool
		role          string
		filmID        string
		data          []byte
		mime          string
		status        int
	}{
		{
			name: "anonymous rejected", filmID: "9", data: pngData.Bytes(), status: http.StatusUnauthorized,
		},
		{
			name: "reader rejected", authenticated: true, role: "reader", filmID: "9", data: pngData.Bytes(), status: http.StatusForbidden,
		},
		{
			name: "admin uploads PNG", authenticated: true, role: "admin", filmID: "9", data: pngData.Bytes(), mime: "image/png", status: http.StatusCreated,
		},
		{
			name: "admin replaces with JPEG", authenticated: true, role: "admin", filmID: "9", data: jpegData.Bytes(), mime: "image/jpeg", status: http.StatusCreated,
		},
		{
			name: "missing film rejected", authenticated: true, role: "admin", filmID: "999", data: pngData.Bytes(), status: http.StatusNotFound,
		},
		{
			name: "fake image rejected", authenticated: true, role: "admin", filmID: "9", data: []byte("fake PNG"), status: http.StatusBadRequest,
		},
	}
	var previousFilename string
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			users.role = tt.role
			var body bytes.Buffer
			writer := multipart.NewWriter(&body)
			part, err := writer.CreateFormFile("poster", "integration.png")
			require.NoError(t, err)
			_, err = part.Write(tt.data)
			require.NoError(t, err)
			require.NoError(t, writer.Close())
			request, err := http.NewRequestWithContext(ctx, http.MethodPost, server.URL+"/api/films/"+tt.filmID+"/poster", &body)
			require.NoError(t, err)
			request.Header.Set("Content-Type", writer.FormDataContentType())
			if tt.authenticated {
				request.AddCookie(&http.Cookie{
					Name:  authdelivery.SessionCookieName,
					Value: "integration-session",
				})
			}
			response, err := httpClient.Do(request)
			require.NoError(t, err)
			defer response.Body.Close()
			require.Equal(t, tt.status, response.StatusCode)
			if tt.status != http.StatusCreated {
				return
			}
			var created struct {
				Filename string `json:"filename"`
			}
			require.NoError(t, json.NewDecoder(response.Body).Decode(&created))
			require.Regexp(t, `^[0-9a-f]{32}\.(png|jpg)$`, created.Filename)
			t.Cleanup(func() {
				cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cleanupCancel()
				require.NoError(t, objects.DeleteFile(cleanupCtx, created.Filename), "remove only generated test object")
			})
			require.NotEqual(t, previousFilename, created.Filename)
			previousFilename = created.Filename
			readResponse, err := httpClient.Get(server.URL + "/api/films/9")
			require.NoError(t, err)
			defer readResponse.Body.Close()
			require.Equal(t, http.StatusOK, readResponse.StatusCode)
			var result struct {
				Film struct {
					PosterURL string `json:"poster_url"`
				} `json:"film"`
			}
			require.NoError(t, json.NewDecoder(readResponse.Body).Decode(&result))
			signed, err := url.Parse(result.Film.PosterURL)
			require.NoError(t, err)
			require.Equal(t, "/"+bucket+"/"+created.Filename, signed.Path)
			require.NotEmpty(t, signed.Query().Get("X-Amz-Signature"))
			download, err := httpClient.Get(result.Film.PosterURL)
			if err != nil {
				t.Fatal("download using signed poster URL failed")
			}
			defer download.Body.Close()
			require.Equal(t, http.StatusOK, download.StatusCode)
			require.Equal(t, tt.mime, download.Header.Get("Content-Type"))
			data, err := io.ReadAll(download.Body)
			require.NoError(t, err)
			require.Equal(t, tt.data, data)
			getPoster, err := httpClient.Get(server.URL + "/api/films/9/poster")
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, getPoster.StatusCode)
			var posterURL struct {
				URL string `json:"url"`
			}
			require.NoError(t, json.NewDecoder(getPoster.Body).Decode(&posterURL))
			getPoster.Body.Close()
			require.Contains(t, posterURL.URL, created.Filename)
			var filename, contentType string
			var owner, size int64
			require.NoError(t, db.QueryRowContext(ctx, `SELECT f.storage_key, f.mime_type, f.created_by, f.size_bytes FROM film_file ff JOIN file f ON f.id = ff.file_id WHERE ff.film_id = 9 AND ff.file_role = 'poster' AND ff.sort_order = 1`).Scan(&filename, &contentType, &owner, &size))
			require.Equal(t, created.Filename, filename)
			require.Equal(t, tt.mime, contentType)
			require.Equal(t, int64(1), owner)
			require.Equal(t, int64(len(tt.data)), size)
		})
	}
	for _, role := range []string{"reader", "admin"} {
		users.role = role
		request, err := http.NewRequestWithContext(ctx, http.MethodDelete, server.URL+"/api/films/9/poster", nil)
		require.NoError(t, err)
		request.AddCookie(&http.Cookie{
			Name:  authdelivery.SessionCookieName,
			Value: "integration-session",
		})
		response, err := httpClient.Do(request)
		require.NoError(t, err)
		response.Body.Close()
		if role == "reader" {
			require.Equal(t, http.StatusForbidden, response.StatusCode)
		} else {
			require.Equal(t, http.StatusNoContent, response.StatusCode)
		}
	}
	missingPoster, err := httpClient.Get(server.URL + "/api/films/9/poster")
	require.NoError(t, err)
	missingPoster.Body.Close()
	require.Equal(t, http.StatusNotFound, missingPoster.StatusCode)
	var bindings int
	require.NoError(t, db.QueryRowContext(ctx, `SELECT count(*) FROM film_file WHERE film_id = 9 AND file_role = 'poster'`).Scan(&bindings))
	require.Zero(t, bindings)

	t.Log("HTTP upload -> PostgreSQL binding -> film poster_url -> signed RustFS download; replacement and access checks passed")
}
