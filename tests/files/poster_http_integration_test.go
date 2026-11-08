//go:build integration

package files_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/config"
	authdelivery "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/auth"
	filesdelivery "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/files"
	filmsdelivery "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/films"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/delivery/router"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	authservice "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/services/auth"
	filesservice "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/services/files"
	filmsservice "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/services/films"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/storage"
	authusecase "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/auth"
	filesusecase "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/files"
	filmsusecase "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/usecase/films"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
)

func TestRustFSPosterHTTP(t *testing.T) {
	dsn := os.Getenv("POSTERS_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set POSTERS_TEST_DATABASE_URL for the PostgreSQL and RustFS integration test")
	}
	cfg, err := config.LoadS3("../../configs/s3.yaml")
	require.NoError(t, err)
	s3Client, err := storage.NewS3Client(*cfg)
	require.NoError(t, err)
	objects, err := filesservice.NewS3Storage(s3Client, cfg.PostersBucket, cfg.PublicEndpoint, cfg.AvatarURLLifetime)
	require.NoError(t, err)
	db, err := sql.Open("pgx", dsn)
	require.NoError(t, err)
	defer db.Close()
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
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
	appLogger := testFilesLogger(t)
	filmRepo := filmsservice.NewPostgresService(db)
	posterRepo, err := filesservice.NewPostgresPosterRepo(db)
	require.NoError(t, err)
	upload, err := filesusecase.NewUploadPosterUseCase(objects, posterRepo, filmRepo, filesusecase.UploadPosterOptions{
		MaxSize: cfg.MaxPosterSize, MaxDimension: cfg.MaxPosterDimension, CleanupTimeout: cfg.PosterCleanupTimeout,
	}, appLogger)
	require.NoError(t, err)
	uploadHandler, err := filesdelivery.NewPosterHandler(upload, appLogger)
	require.NoError(t, err)
	filmReader, err := filmsservice.NewRepositoryWithPosterURLs(filmRepo, objects, appLogger)
	require.NoError(t, err)
	list, err := filmsusecase.NewListFilmsUseCase(filmReader, appLogger)
	require.NoError(t, err)
	get, err := filmsusecase.NewGetFilmUseCase(filmReader, appLogger)
	require.NoError(t, err)
	create, err := filmsusecase.NewCreateFilmUseCase(filmRepo, appLogger)
	require.NoError(t, err)
	update, err := filmsusecase.NewUpdateFilmUseCase(filmRepo, appLogger)
	require.NoError(t, err)
	deleteFilm, err := filmsusecase.NewDeleteFilmUseCase(filmRepo, appLogger)
	require.NoError(t, err)
	collections, err := filmsusecase.NewListCollectionsUseCase(filmReader, appLogger)
	require.NoError(t, err)
	collection, err := filmsusecase.NewGetCollectionUseCase(filmReader, appLogger)
	require.NoError(t, err)
	readHandler, err := filmsdelivery.NewHandler(filmsdelivery.HandlerUseCases{
		ListFilms: list, GetFilm: get, CreateFilm: create, UpdateFilm: update,
		DeleteFilm: deleteFilm, ListCollections: collections, GetCollection: collection,
	}, appLogger)
	require.NoError(t, err)
	users := authservice.NewInMemoryUserRepo()
	userID, err := users.Create(ctx, "poster-integration@example.com", "poster-integration", "unused-test-hash")
	require.NoError(t, err)
	user, err := users.GetUser(ctx, userID)
	require.NoError(t, err)
	sessions := authservice.NewInMemorySessionStore(time.Hour)
	session, err := sessions.Create(ctx, userID)
	require.NoError(t, err)
	passwords := authservice.BcryptPasswordService{}
	register, err := authusecase.NewRegisterUserUseCase(users, sessions, passwords, appLogger)
	require.NoError(t, err)
	login, err := authusecase.NewLoginUserUseCase(users, sessions, passwords, appLogger)
	require.NoError(t, err)
	logout, err := authusecase.NewLogoutUserUseCase(sessions, appLogger)
	require.NoError(t, err)
	authenticate, err := authusecase.NewAuthenticateUserUseCase(sessions, appLogger)
	require.NoError(t, err)
	getUser, err := authusecase.NewGetUserUseCase(users, appLogger)
	require.NoError(t, err)
	authHandler, err := authdelivery.NewHandler(authdelivery.HandlerUseCases{
		Register: register, Login: login, Logout: logout, Authenticate: authenticate, GetUser: getUser,
	}, appLogger)
	require.NoError(t, err)
	routes := mux.NewRouter()
	routes.HandleFunc("/api/films/{id}/poster", router.WrapMiddlewares(uploadHandler.UploadPoster, []router.Middleware{
		router.RequireAuth(authHandler), authHandler.RequireAdmin,
	})).Methods(http.MethodPost)
	routes.HandleFunc("/api/films/{id}", readHandler.GetFilm).Methods(http.MethodGet)
	server := httptest.NewServer(routes)
	defer server.Close()
	httpClient := &http.Client{Timeout: 10 * time.Second}
	var pngData, jpegData bytes.Buffer
	picture := image.NewRGBA(image.Rect(0, 0, 2, 2))
	require.NoError(t, png.Encode(&pngData, picture))
	require.NoError(t, jpeg.Encode(&jpegData, picture, nil))
	tests := []struct {
		name          string
		authenticated bool
		role          domain.Role
		filmID        string
		data          []byte
		mime          string
		status        int
	}{
		{
			name: "anonymous rejected", filmID: "9", data: pngData.Bytes(), status: http.StatusUnauthorized,
		},
		{
			name: "reader rejected", authenticated: true, role: domain.RoleReader, filmID: "9", data: pngData.Bytes(), status: http.StatusForbidden,
		},
		{
			name: "admin uploads PNG", authenticated: true, role: domain.RoleAdmin, filmID: "9", data: pngData.Bytes(), mime: "image/png", status: http.StatusCreated,
		},
		{
			name: "admin replaces with JPEG", authenticated: true, role: domain.RoleAdmin, filmID: "9", data: jpegData.Bytes(), mime: "image/jpeg", status: http.StatusCreated,
		},
		{
			name: "missing film rejected", authenticated: true, role: domain.RoleAdmin, filmID: "999", data: pngData.Bytes(), status: http.StatusNotFound,
		},
		{
			name: "fake image rejected", authenticated: true, role: domain.RoleAdmin, filmID: "9", data: []byte("fake PNG"), status: http.StatusBadRequest,
		},
	}
	var previousFilename string
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user.Role = tt.role
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
				request.AddCookie(&http.Cookie{Name: authdelivery.SessionCookieName, Value: session.ID})
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
				require.NoError(t, objects.Delete(cleanupCtx, created.Filename), "remove only generated test object")
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
			require.Equal(t, "/"+cfg.PostersBucket+"/"+created.Filename, signed.Path)
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
			var filename, contentType string
			var owner, size int64
			require.NoError(t, db.QueryRowContext(ctx, `SELECT f.storage_key, f.mime_type, f.created_by, f.size_bytes FROM film_file ff JOIN file f ON f.id = ff.file_id WHERE ff.film_id = 9 AND ff.file_role = 'poster' AND ff.sort_order = 1`).Scan(&filename, &contentType, &owner, &size))
			require.Equal(t, created.Filename, filename)
			require.Equal(t, tt.mime, contentType)
			require.Equal(t, int64(userID), owner)
			require.Equal(t, int64(len(tt.data)), size)
		})
	}
	t.Log("HTTP upload -> PostgreSQL binding -> film poster_url -> signed RustFS download; replacement and access checks passed")
}
