package films_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	filesservice "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/services/files"
	filmsservice "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/services/films"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
	"github.com/stretchr/testify/require"
)

type posterFilmsRepo struct {
	domain.FilmsRepository
	films         []domain.Film
	err           error
	limit, offset int
	id            domain.CollectionID
}

func (r *posterFilmsRepo) GetFilmByID(_ context.Context, id domain.FilmID) (domain.Film, error) {
	if r.err != nil {
		return domain.Film{}, r.err
	}
	return r.films[0], nil
}
func (r *posterFilmsRepo) ListFilms(_ context.Context, limit, offset int) ([]domain.Film, int, error) {
	r.limit, r.offset = limit, offset
	return r.films, 12, r.err
}
func (r *posterFilmsRepo) ListCollectionFilms(_ context.Context, id domain.CollectionID, limit, offset int) ([]domain.Film, int, error) {
	r.id = id
	return r.ListFilms(context.Background(), limit, offset)
}

func TestFilmPosterSignedURLs(t *testing.T) {
	tests := []struct {
		name          string
		external, key *string
		status        int
		repoErr       error
		wantRequest   bool
		wantErr       bool
	}{
		{
			name: "private poster signed", key: ptr("posters/film 9.png"), status: http.StatusOK, wantRequest: true,
		},
		{
			name: "external URL preserved", external: ptr("https://images.example.org/poster.png"), key: ptr("ignored.png"),
		},
		{
			name: "no poster",
		},
		{
			name: "missing S3 object omits URL", key: ptr("missing.png"), status: http.StatusNotFound, wantRequest: true,
		},
		{
			name: "S3 failure returned", key: ptr("failed.png"), status: http.StatusForbidden, wantRequest: true, wantErr: true,
		},
		{
			name: "repository failure returned", repoErr: errors.New("database unavailable"), wantErr: true,
		},
	}
	for _, tt := range tests {
		for _, operation := range []string{"get", "list", "collection"} {
			t.Run(tt.name+"/"+operation, func(t *testing.T) {
				requests := 0
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					requests++
					require.Equal(t, http.MethodHead, r.Method)
					require.Equal(t, "/posters/"+*tt.key, r.URL.Path)
					w.WriteHeader(tt.status)
				}))
				defer server.Close()
				client := s3.NewFromConfig(aws.Config{
					Region: "us-east-1", Credentials: credentials.NewStaticCredentialsProvider("test-key", "test-secret", ""),
				}, func(o *s3.Options) {
					o.BaseEndpoint = aws.String(server.URL)
					o.UsePathStyle = true
					o.RetryMaxAttempts = 1
				})
				storage, err := filesservice.NewS3Storage(client, "posters", "https://media.example.org", 15*time.Minute)
				require.NoError(t, err)
				appLogger, err := logger.NewLogger(io.Discard, logger.LogLevelDebug, "json")
				require.NoError(t, err)
				original := domain.Film{ID: 9, PosterURL: tt.external, PosterStorageKey: tt.key}
				repo := &posterFilmsRepo{films: []domain.Film{original}, err: tt.repoErr}
				reader, err := filmsservice.NewRepositoryWithPosterURLs(repo, storage, appLogger)
				require.NoError(t, err)
				var films []domain.Film
				var total int
				switch operation {
				case "get":
					var film domain.Film
					film, err = reader.GetFilmByID(t.Context(), 9)
					films = []domain.Film{film}
				case "list":
					films, total, err = reader.ListFilms(t.Context(), 3, 4)
				case "collection":
					films, total, err = reader.ListCollectionFilms(t.Context(), 5, 3, 4)
				}
				if tt.wantErr {
					require.Error(t, err)
					if tt.repoErr != nil {
						require.ErrorIs(t, err, tt.repoErr)
					}
				} else {
					require.NoError(t, err)
					require.Len(t, films, 1)
					if tt.external != nil {
						require.Equal(t, tt.external, films[0].PosterURL)
					} else if tt.key != nil && tt.status == http.StatusOK {
						require.NotNil(t, films[0].PosterURL)
						signed, err := url.Parse(*films[0].PosterURL)
						require.NoError(t, err)
						require.Equal(t, "https", signed.Scheme)
						require.Equal(t, "media.example.org", signed.Host)
						require.Equal(t, "/posters/"+*tt.key, signed.Path)
						require.NotEmpty(t, signed.Query().Get("X-Amz-Signature"))
						require.Equal(t, "900", signed.Query().Get("X-Amz-Expires"))
					} else {
						require.Nil(t, films[0].PosterURL)
					}
					if operation != "get" {
						require.Equal(t, 12, total)
						require.Equal(t, 3, repo.limit)
						require.Equal(t, 4, repo.offset)
						if operation == "collection" {
							require.Equal(t, domain.CollectionID(5), repo.id)
						}
					}
				}
				if tt.wantRequest {
					require.Equal(t, 1, requests)
				} else {
					require.Zero(t, requests)
				}
				require.Equal(t, original, repo.films[0], "signed URL must not be saved in repository data")
			})
		}
	}
}
