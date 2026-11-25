package usecase

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/png"
	"io"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/catalog/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/catalog/internal/repository"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/logger"
)

type filmStub struct {
	repository.FilmWriter
	get    func(context.Context, domain.FilmID) (domain.Film, error)
	list   func(context.Context, int, int) ([]domain.Film, int, error)
	create func(context.Context, domain.CreateFilmInput) (domain.FilmID, error)
}

func (s filmStub) GetFilmByID(ctx context.Context, id domain.FilmID) (domain.Film, error) {
	return s.get(ctx, id)
}

func (s filmStub) ListFilms(ctx context.Context, limit, offset int) ([]domain.Film, int, error) {
	return s.list(ctx, limit, offset)
}

func (s filmStub) CreateFilm(ctx context.Context, input domain.CreateFilmInput) (domain.FilmID, error) {
	return s.create(ctx, input)
}

type collectionStub struct {
	repository.CollectionReader
	get       func(context.Context, domain.CollectionID) (domain.Collection, error)
	listFilms func(context.Context, domain.CollectionID, int, int) ([]domain.Film, int, error)
}

func (s collectionStub) GetCollectionByID(ctx context.Context, id domain.CollectionID) (domain.Collection, error) {
	return s.get(ctx, id)
}

func (s collectionStub) ListCollectionFilms(ctx context.Context, id domain.CollectionID, limit, offset int) ([]domain.Film, int, error) {
	return s.listFilms(ctx, id, limit, offset)
}

type ratingStub struct {
	repository.RatingRepository
	score *int
}

func (s ratingStub) GetMyScore(context.Context, domain.UserID, domain.FilmID) (*int, error) {
	return s.score, nil
}

type preferenceStub struct {
	repository.PreferencesRepository
	saved []domain.PreferenceUpdate
}

func (s *preferenceStub) SavePreference(_ context.Context, _ domain.UserID, update domain.PreferenceUpdate) error {
	s.saved = append(s.saved, update)
	return nil
}

type posterStub struct {
	repository.PosterRepository
	save func(context.Context, domain.FilmID, domain.UserID, domain.PosterFile) error
	get  func(context.Context, domain.FilmID) (*domain.PosterFile, error)
}

func (s posterStub) SavePoster(ctx context.Context, filmID domain.FilmID, userID domain.UserID, file domain.PosterFile) error {
	return s.save(ctx, filmID, userID, file)
}

func (s posterStub) GetPoster(ctx context.Context, id domain.FilmID) (*domain.PosterFile, error) {
	return s.get(ctx, id)
}

type storageStub struct {
	upload func(context.Context, string, string, io.ReadSeeker) error
	delete func(context.Context, string) error
	get    func(context.Context, string) (string, error)
}

func (s storageStub) Upload(ctx context.Context, key, contentType string, body io.ReadSeeker) error {
	return s.upload(ctx, key, contentType, body)
}

func (s storageStub) Delete(ctx context.Context, key string) error {
	return s.delete(ctx, key)
}

func (s storageStub) GetURL(ctx context.Context, key string) (string, error) {
	return s.get(ctx, key)
}

func TestPaginationRejectsInvalidValuesBeforeRepositoryCall(t *testing.T) {
	cases := []struct {
		name   string
		limit  int
		offset int
		want   error
	}{
		{
			name:   "zero limit",
			limit:  0,
			offset: 0,
			want:   ErrInvalidLimit,
		},
		{
			name:   "negative limit",
			limit:  -1,
			offset: 0,
			want:   ErrInvalidLimit,
		},
		{
			name:   "large limit",
			limit:  101,
			offset: 0,
			want:   ErrInvalidLimit,
		},
		{
			name:   "negative offset",
			limit:  20,
			offset: -1,
			want:   ErrInvalidOffset,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := NewFilmUseCase(nil, nil, nil).ListFilms(t.Context(), tc.limit, tc.offset)
			if !errors.Is(err, tc.want) {
				t.Fatalf("film pagination error = %v, want %v", err, tc.want)
			}

			collections := NewCollectionUseCase(nil)
			if _, _, err := collections.ListCollections(t.Context(), tc.limit, tc.offset); !errors.Is(err, tc.want) {
				t.Fatalf("collections pagination error = %v, want %v", err, tc.want)
			}

			if _, err := collections.GetCollection(t.Context(), 1, tc.limit, tc.offset); !errors.Is(err, tc.want) {
				t.Fatalf("collection films pagination error = %v, want %v", err, tc.want)
			}

			ratings := NewRatingUseCase(nil, nil, nil)
			if _, _, err := ratings.GetCollectionRatings(t.Context(), 1, tc.limit, tc.offset); !errors.Is(err, tc.want) {
				t.Fatalf("ratings pagination error = %v, want %v", err, tc.want)
			}
		})
	}
}

func TestFilmPaginationPreservesTotalBeyondLastPage(t *testing.T) {
	cases := []struct {
		name   string
		limit  int
		offset int
		total  int
		films  []domain.Film
	}{
		{
			name:   "first page",
			limit:  1,
			offset: 0,
			total:  37,
			films: []domain.Film{{
				ID: 1,
			}},
		},
		{
			name:   "beyond last page",
			limit:  MaxLimit,
			offset: 500,
			total:  37,
			films:  []domain.Film{},
		},
		{
			name:   "empty catalog",
			limit:  DefaultLimit,
			offset: DefaultOffset,
			total:  0,
			films:  []domain.Film{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := filmStub{
				list: func(_ context.Context, limit, offset int) ([]domain.Film, int, error) {
					if limit != tc.limit || offset != tc.offset {
						t.Fatalf("unexpected pagination: %d, %d", limit, offset)
					}
					return tc.films, tc.total, nil
				},
			}

			films, total, err := NewFilmUseCase(repo, repo, repo).ListFilms(t.Context(), tc.limit, tc.offset)
			if err != nil || !reflect.DeepEqual(films, tc.films) || total != tc.total {
				t.Fatalf("result = %v, %d, %v", films, total, err)
			}
		})
	}
}

func TestCreateFilmNormalizesAndValidatesBeforeWrite(t *testing.T) {
	cases := []struct {
		name      string
		title     string
		year      int16
		wantTitle string
		want      error
	}{
		{
			name:      "trim title",
			title:     "  Film  ",
			year:      2020,
			wantTitle: "Film",
		},
		{
			name:  "empty title",
			title: " ",
			year:  2020,
			want:  domain.ErrInvalidFilmInput,
		},
		{
			name:  "invalid year",
			title: "Film",
			year:  1000,
			want:  domain.ErrInvalidFilmInput,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			repo := filmStub{
				create: func(_ context.Context, input domain.CreateFilmInput) (domain.FilmID, error) {
					calls++
					if input.Title != tc.wantTitle {
						t.Fatalf("title = %q", input.Title)
					}
					return 7, nil
				},
			}
			input := domain.CreateFilmInput{
				Title:       tc.title,
				FilmType:    domain.FilmTypeMovie,
				ReleaseYear: tc.year,
			}

			id, err := NewFilmUseCase(repo, repo, repo).CreateFilm(t.Context(), input)
			if !errors.Is(err, tc.want) {
				t.Fatalf("create error = %v, want %v", err, tc.want)
			}

			if tc.want == nil {
				if calls != 1 || id != 7 {
					t.Fatalf("create id = %d, write calls = %d", id, calls)
				}
			} else if calls != 0 || id != 0 {
				t.Fatalf("invalid film was written: id = %d, calls = %d", id, calls)
			}
		})
	}
}

func TestCollectionKeepsMetadataForEmptyPage(t *testing.T) {
	cases := []struct {
		name      string
		lookupErr error
		pageErr   error
		want      error
	}{
		{
			name: "empty page",
		},
		{
			name:      "missing collection",
			lookupErr: domain.ErrCollectionNotFound,
			want:      domain.ErrCollectionNotFound,
		},
		{
			name:    "page query failed",
			pageErr: context.DeadlineExceeded,
			want:    context.DeadlineExceeded,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := collectionStub{
				get: func(_ context.Context, id domain.CollectionID) (domain.Collection, error) {
					return domain.Collection{
						ID:    id,
						Title: "Collection",
					}, tc.lookupErr
				},
				listFilms: func(_ context.Context, id domain.CollectionID, limit, offset int) ([]domain.Film, int, error) {
					if tc.lookupErr != nil {
						t.Fatal("page requested after failed lookup")
					}
					if id != 5 || limit != 20 || offset != 100 {
						t.Fatalf("unexpected collection page: %d, %d, %d", id, limit, offset)
					}
					return []domain.Film{}, 3, tc.pageErr
				},
			}

			result, err := NewCollectionUseCase(repo).GetCollection(t.Context(), 5, 20, 100)
			if !errors.Is(err, tc.want) {
				t.Fatalf("collection error = %v, want %v", err, tc.want)
			}

			if err == nil && (result.Collection.ID != 5 || result.Total != 3 || len(result.Films) != 0) {
				t.Fatalf("result = %+v", result)
			}
		})
	}
}

func TestRatingDoesNotWriteForMissingFilm(t *testing.T) {
	cases := []struct {
		name string
		want error
	}{
		{
			name: "missing film",
			want: domain.ErrFilmNotFound,
		},
		{
			name: "lookup timeout",
			want: context.DeadlineExceeded,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			films := filmStub{
				get: func(context.Context, domain.FilmID) (domain.Film, error) {
					return domain.Film{}, fmt.Errorf("lookup: %w", tc.want)
				},
			}

			uc := NewRatingUseCase(nil, nil, films)
			if _, err := uc.SetFilmRating(t.Context(), 1, 2, 7); !errors.Is(err, tc.want) {
				t.Fatalf("film lookup error = %v, want %v", err, tc.want)
			}
		})
	}
}

func TestRatingValidationAndMissingPersonalScore(t *testing.T) {
	cases := []struct {
		name     string
		user     domain.UserID
		film     domain.FilmID
		score    int
		personal bool
		want     error
	}{
		{
			name:  "missing user",
			user:  0,
			film:  2,
			score: 7,
			want:  domain.ErrUnauthorized,
		},
		{
			name:  "low score",
			user:  1,
			film:  2,
			score: 0,
			want:  domain.ErrInvalidFilmRating,
		},
		{
			name:  "high score",
			user:  1,
			film:  2,
			score: 11,
			want:  domain.ErrInvalidFilmRating,
		},
		{
			name:     "invalid personal film",
			user:     1,
			film:     0,
			personal: true,
			want:     domain.ErrFilmNotFound,
		},
		{
			name:     "missing personal score",
			user:     1,
			film:     2,
			personal: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			films := filmStub{
				get: func(_ context.Context, id domain.FilmID) (domain.Film, error) {
					return domain.Film{
						ID: id,
					}, nil
				},
			}

			uc := NewRatingUseCase(ratingStub{}, nil, films)
			if tc.personal {
				score, err := uc.GetMyFilmRating(t.Context(), tc.user, tc.film)
				if !errors.Is(err, tc.want) || score != nil {
					t.Fatalf("personal score = %v, error = %v, want %v", score, err, tc.want)
				}

				return
			}

			if _, err := uc.SetFilmRating(t.Context(), tc.user, tc.film, tc.score); !errors.Is(err, tc.want) {
				t.Fatalf("rating error = %v, want %v", err, tc.want)
			}
		})
	}
}

func TestPreferenceUpdatePreservesExplicitFalse(t *testing.T) {
	collapsed := false
	grid := domain.CatalogViewGrid
	invalid := domain.CatalogView("invalid")
	cases := []struct {
		name   string
		user   domain.UserID
		update domain.PreferenceUpdate
		want   error
	}{
		{
			name: "explicit false",
			user: 1,
			update: domain.PreferenceUpdate{
				SidebarCollapsed: &collapsed,
			},
		},
		{
			name: "catalog view",
			user: 1,
			update: domain.PreferenceUpdate{
				CatalogView: &grid,
			},
		},
		{
			name: "missing user",
			user: 0,
			update: domain.PreferenceUpdate{
				CatalogView: &grid,
			},
			want: domain.ErrUnauthorized,
		},
		{
			name: "empty update",
			user: 1,
			want: domain.ErrInvalidPreferences,
		},
		{
			name: "two settings",
			user: 1,
			update: domain.PreferenceUpdate{
				SidebarCollapsed: &collapsed,
				CatalogView:      &grid,
			},
			want: domain.ErrInvalidPreferences,
		},
		{
			name: "invalid view",
			user: 1,
			update: domain.PreferenceUpdate{
				CatalogView: &invalid,
			},
			want: domain.ErrInvalidPreferences,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &preferenceStub{}

			err := NewPreferencesUseCase(repo).SavePreference(t.Context(), tc.user, tc.update)
			if !errors.Is(err, tc.want) {
				t.Fatalf("preference error = %v, want %v", err, tc.want)
			}

			if tc.want == nil {
				if len(repo.saved) != 1 || !reflect.DeepEqual(repo.saved[0], tc.update) {
					t.Fatalf("saved updates = %+v", repo.saved)
				}
			} else if len(repo.saved) != 0 {
				t.Fatalf("invalid update was saved: %+v", repo.saved)
			}
		})
	}
}

func TestLibraryRejectsInvalidInputBeforeRepositoryCall(t *testing.T) {
	cases := []struct {
		name string
		user domain.UserID
		list domain.List
		film domain.FilmID
		want error
	}{
		{
			name: "missing user",
			user: 0,
			list: domain.Favorite,
			film: 2,
			want: domain.ErrUnauthorized,
		},
		{
			name: "invalid list",
			user: 1,
			list: domain.List("invalid"),
			film: 2,
			want: domain.ErrInvalidLibraryData,
		},
		{
			name: "invalid film",
			user: 1,
			list: domain.Watched,
			film: 0,
			want: domain.ErrInvalidLibraryData,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := NewLibraryUseCase(nil).SetSaved(t.Context(), tc.user, tc.list, tc.film, true); !errors.Is(err, tc.want) {
				t.Fatalf("library error = %v, want %v", err, tc.want)
			}
		})
	}
}

func pngData(t *testing.T, width, height int) []byte {
	t.Helper()

	var buffer bytes.Buffer
	if err := png.Encode(&buffer, image.NewRGBA(image.Rect(0, 0, width, height))); err != nil {
		t.Fatal(err)
	}

	return buffer.Bytes()
}

func TestPosterImageValidation(t *testing.T) {
	data := pngData(t, 2, 2)
	cases := []struct {
		name         string
		body         io.ReadSeeker
		size         int64
		maxSize      int64
		maxDimension int
		want         error
	}{
		{
			name:         "missing",
			body:         nil,
			size:         1,
			maxSize:      1024,
			maxDimension: 4,
			want:         ErrPosterMissing,
		},
		{
			name:         "empty",
			body:         bytes.NewReader(nil),
			size:         1,
			maxSize:      1024,
			maxDimension: 4,
			want:         ErrInvalidPosterSize,
		},
		{
			name:         "invalid declared size",
			body:         bytes.NewReader(data),
			size:         0,
			maxSize:      1024,
			maxDimension: 4,
			want:         ErrInvalidPosterSize,
		},
		{
			name:         "declared too large",
			body:         bytes.NewReader(data),
			size:         1025,
			maxSize:      1024,
			maxDimension: 4,
			want:         ErrPosterTooLarge,
		},
		{
			name:         "actual too large",
			body:         bytes.NewReader(data),
			size:         1,
			maxSize:      8,
			maxDimension: 4,
			want:         ErrPosterTooLarge,
		},
		{
			name:         "invalid image",
			body:         strings.NewReader("bad"),
			size:         3,
			maxSize:      1024,
			maxDimension: 4,
			want:         ErrInvalidPosterImage,
		},
		{
			name:         "dimensions",
			body:         bytes.NewReader(data),
			size:         int64(len(data)),
			maxSize:      1024,
			maxDimension: 1,
			want:         ErrPosterDimensionsTooLarge,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := readUploadImage(tc.body, tc.size, tc.maxSize, tc.maxDimension); !errors.Is(err, tc.want) {
				t.Fatalf("image error = %v, want %v", err, tc.want)
			}
		})
	}
}

func TestPosterUploadCleanup(t *testing.T) {
	saveFailure := errors.New("save failed")
	cleanupFailure := errors.New("cleanup failed")
	cases := []struct {
		name       string
		saveErr    error
		cleanupErr error
		wantDelete bool
	}{
		{
			name:       "success",
			saveErr:    nil,
			cleanupErr: nil,
			wantDelete: false,
		},
		{
			name:       "rollback",
			saveErr:    saveFailure,
			cleanupErr: nil,
			wantDelete: true,
		},
		{
			name:       "cleanup failure",
			saveErr:    saveFailure,
			cleanupErr: cleanupFailure,
			wantDelete: true,
		},
		{
			name:       "uncertain commit",
			saveErr:    fmt.Errorf("commit: %w", repository.ErrCommitUnknown),
			cleanupErr: nil,
			wantDelete: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()

			data := pngData(t, 2, 2)
			key := ""
			deleted := false
			storage := storageStub{
				upload: func(_ context.Context, filename, contentType string, body io.ReadSeeker) error {
					key = filename
					actual, err := io.ReadAll(body)
					if err != nil || !bytes.Equal(actual, data) || contentType != "image/png" || !strings.HasSuffix(filename, ".png") {
						t.Fatalf("unexpected upload: %s, %s, %v", filename, contentType, err)
					}
					return nil
				},
				delete: func(cleanupCtx context.Context, filename string) error {
					deleted = true
					if cleanupCtx.Err() != nil || filename != key {
						t.Fatalf("invalid cleanup: %s, %v", filename, cleanupCtx.Err())
					}
					if _, ok := cleanupCtx.Deadline(); !ok {
						t.Fatal("cleanup has no deadline")
					}
					return tc.cleanupErr
				},
			}
			posters := posterStub{
				save: func(_ context.Context, filmID domain.FilmID, userID domain.UserID, file domain.PosterFile) error {
					if filmID != 2 || userID != 1 || file.Filename != key || file.Size != int64(len(data)) {
						t.Fatalf("unexpected metadata: %+v", file)
					}
					cancel()
					return tc.saveErr
				},
			}
			films := filmStub{
				get: func(_ context.Context, id domain.FilmID) (domain.Film, error) {
					return domain.Film{
						ID: id,
					}, nil
				},
			}

			log, err := logger.NewLogger(io.Discard, logger.LogLevelInfo, "json")
			if err != nil {
				t.Fatal(err)
			}

			uc, err := NewPosterUseCase(posters, films, storage, PosterOptions{
				MaxSize:        1024,
				MaxDimension:   4,
				CleanupTimeout: time.Second,
			}, log)
			if err != nil {
				t.Fatal(err)
			}

			filename, err := uc.UploadPoster(ctx, UploadPosterInput{
				UserID: 1,
				FilmID: 2,
				Body:   bytes.NewReader(data),
				Size:   int64(len(data)),
			})
			if tc.saveErr == nil {
				if err != nil || filename != key {
					t.Fatalf("upload = %q, %v", filename, err)
				}
			} else if !errors.Is(err, tc.saveErr) || filename != "" {
				t.Fatalf("failed upload = %q, %v", filename, err)
			}

			if tc.cleanupErr != nil && !errors.Is(err, tc.cleanupErr) {
				t.Fatalf("lost cleanup error: %v", err)
			}

			if deleted != tc.wantDelete {
				t.Fatalf("delete = %v, want %v", deleted, tc.wantDelete)
			}
		})
	}
}

func TestPosterURLReaderDoesNotMutateRepositoryData(t *testing.T) {
	cases := []struct {
		name       string
		storageErr error
		wantURL    string
		want       error
	}{
		{
			name:    "signed URL",
			wantURL: "signed-url",
		},
		{
			name:       "missing object",
			storageErr: domain.ErrFileNotFound,
		},
		{
			name:       "storage failure",
			storageErr: context.DeadlineExceeded,
			want:       context.DeadlineExceeded,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			key := "poster.png"
			external := "https://example.com/poster.png"
			films := []domain.Film{{
				ID:               1,
				PosterStorageKey: &key,
			}, {
				ID:        2,
				PosterURL: &external,
			}}
			repo := filmStub{
				list: func(context.Context, int, int) ([]domain.Film, int, error) {
					return films, 2, nil
				},
			}
			storage := storageStub{
				get: func(_ context.Context, name string) (string, error) {
					if name != key {
						t.Fatalf("key = %q", name)
					}
					return "signed-url", tc.storageErr
				},
			}

			reader, err := NewPosterURLReader(repo, repo, collectionStub{}, storage)
			if err != nil {
				t.Fatal(err)
			}

			result, total, err := reader.ListFilms(t.Context(), 20, 0)
			if !errors.Is(err, tc.want) {
				t.Fatalf("URL error = %v, want %v", err, tc.want)
			}

			if films[0].PosterURL != nil {
				t.Fatal("repository data changed")
			}

			if tc.want != nil {
				return
			}

			if total != 2 || len(result) != 2 || result[1].PosterURL != &external {
				t.Fatalf("films = %+v, total = %d", result, total)
			}

			if tc.wantURL == "" {
				if result[0].PosterURL != nil {
					t.Fatal("missing object received a URL")
				}
			} else if result[0].PosterURL == nil || *result[0].PosterURL != tc.wantURL {
				t.Fatalf("URL = %v, want %q", result[0].PosterURL, tc.wantURL)
			}
		})
	}
}
