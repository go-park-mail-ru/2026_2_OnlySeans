package films_test

import (
	"context"
	"database/sql/driver"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	films "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/services/films"
)

var (
	filmColumns       = []string{"id", "title", "original_title", "film_type", "production_year", "duration_min", "age_limit", "description", "poster_url", "trailer_url"}
	genreColumns      = []string{"film_id", "id", "name", "slug"}
	collectionColumns = []string{"id", "title", "slug", "description", "cover_url"}
	countColumns      = []string{"count"}
)

// int64SliceConverter пропускает срез номеров фильмов в запрос жанров как есть.
// Настоящий драйвер pgx превращает []int64 в массив Postgres, а sqlmock
// по умолчанию срезы не принимает.
type int64SliceConverter struct{}

func (int64SliceConverter) ConvertValue(v any) (driver.Value, error) {
	if ids, ok := v.([]int64); ok {
		return ids, nil
	}
	return driver.DefaultParameterConverter.ConvertValue(v)
}

func newPostgresService(t *testing.T) (*films.PostgresService, sqlmock.Sqlmock) {
	t.Helper()

	db, mock, err := sqlmock.New(sqlmock.ValueConverterOption(int64SliceConverter{}))
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, mock.ExpectationsWereMet())
		db.Close()
	})

	return films.NewPostgresService(db), mock
}

var (
	dramaGenre = domain.Genre{ID: 1, Name: "драма", Slug: "drama"}
	crimeGenre = domain.Genre{ID: 3, Name: "криминал", Slug: "crime"}
)

// shawshank — фильм со всеми полями, ivan — без оригинального названия и трейлера.
func shawshank() domain.Film {
	return domain.Film{
		ID:            1,
		Title:         "Побег из Шоушенка",
		OriginalTitle: ptr("The Shawshank Redemption"),
		FilmType:      domain.FilmTypeMovie,
		ReleaseYear:   1994,
		DurationMin:   ptr[int16](142),
		AgeLimit:      16,
		Description:   ptr("Банкир Энди Дюфрейн получает пожизненный срок."),
		PosterURL:     ptr("https://avatars.mds.yandex.net/get-kinopoisk-image/1/2/600x900"),
		TrailerURL:    ptr("https://video.example.com/shawshank-trailer-1"),
	}
}

func ivan() domain.Film {
	return domain.Film{
		ID:          11,
		Title:       "Иван Васильевич меняет профессию",
		FilmType:    domain.FilmTypeMovie,
		ReleaseYear: 1973,
		DurationMin: ptr[int16](88),
		AgeLimit:    6,
		PosterURL:   ptr("/static/posters/ivan-vasilievich.svg"),
	}
}

func ptr[T any](v T) *T {
	return &v
}

// value превращает пустой указатель в NULL, а заполненный — в значение.
func value[T any](p *T) driver.Value {
	if p == nil {
		return nil
	}
	return *p
}

func filmRows(filmsToReturn ...domain.Film) *sqlmock.Rows {
	rows := sqlmock.NewRows(filmColumns)
	for _, f := range filmsToReturn {
		rows.AddRow(f.ID, f.Title, value(f.OriginalTitle), string(f.FilmType), f.ReleaseYear,
			value(f.DurationMin), f.AgeLimit, value(f.Description), value(f.PosterURL), value(f.TrailerURL))
	}
	return rows
}

func withGenres(film domain.Film, genres ...domain.Genre) domain.Film {
	film.Genres = append([]domain.Genre{}, genres...)
	return film
}

func countRows(n int) *sqlmock.Rows {
	return sqlmock.NewRows(countColumns).AddRow(n)
}

func TestPostgresService_GetFilmByID(t *testing.T) {
	service, mock := newPostgresService(t)

	mock.ExpectQuery(`FROM film AS f\s+WHERE f.id = \$1`).
		WithArgs(domain.FilmID(1)).
		WillReturnRows(filmRows(shawshank()))
	mock.ExpectQuery(`FROM film_genre`).
		WithArgs([]int64{1}).
		WillReturnRows(sqlmock.NewRows(genreColumns).
			AddRow(1, dramaGenre.ID, dramaGenre.Name, dramaGenre.Slug).
			AddRow(1, crimeGenre.ID, crimeGenre.Name, crimeGenre.Slug))

	film, err := service.GetFilmByID(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, withGenres(shawshank(), dramaGenre, crimeGenre), film)
}

func TestPostgresService_GetFilmByIDNullFields(t *testing.T) {
	service, mock := newPostgresService(t)

	mock.ExpectQuery(`WHERE f.id = \$1`).WillReturnRows(filmRows(ivan()))
	mock.ExpectQuery(`FROM film_genre`).WillReturnRows(sqlmock.NewRows(genreColumns))

	film, err := service.GetFilmByID(context.Background(), 11)
	require.NoError(t, err)
	assert.Nil(t, film.OriginalTitle, "NULL из базы становится nil")
	assert.Nil(t, film.TrailerURL)
	assert.NotNil(t, film.Genres, "фильм без жанров получает [], а не null")
	assert.Empty(t, film.Genres)
}

func TestPostgresService_GetFilmByIDErrors(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(sqlmock.Sqlmock)
		wantErr error
	}{
		{"фильма нет", func(m sqlmock.Sqlmock) {
			m.ExpectQuery(`WHERE f.id = \$1`).WillReturnRows(filmRows())
		}, films.ErrFilmNotFound},
		{"база недоступна", func(m sqlmock.Sqlmock) {
			m.ExpectQuery(`WHERE f.id = \$1`).WillReturnError(errStorage)
		}, errStorage},
		{"не загрузились жанры", func(m sqlmock.Sqlmock) {
			m.ExpectQuery(`WHERE f.id = \$1`).WillReturnRows(filmRows(shawshank()))
			m.ExpectQuery(`FROM film_genre`).WillReturnError(errStorage)
		}, errStorage},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, mock := newPostgresService(t)
			tt.setup(mock)

			_, err := service.GetFilmByID(context.Background(), 1)
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestPostgresService_ListFilms(t *testing.T) {
	service, mock := newPostgresService(t)

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT count\(\*\) FROM film`).WillReturnRows(countRows(13))
	mock.ExpectQuery(`FROM film AS f\s+ORDER BY f.id\s+LIMIT \$1 OFFSET \$2`).
		WithArgs(2, 0).
		WillReturnRows(filmRows(shawshank(), ivan()))
	mock.ExpectQuery(`FROM film_genre`).
		WithArgs([]int64{1, 11}).
		WillReturnRows(sqlmock.NewRows(genreColumns).
			AddRow(11, dramaGenre.ID, dramaGenre.Name, dramaGenre.Slug))
	mock.ExpectCommit()

	list, total, err := service.ListFilms(context.Background(), 2, 0)
	require.NoError(t, err)
	assert.Equal(t, 13, total)
	assert.Equal(t, []domain.Film{withGenres(shawshank()), withGenres(ivan(), dramaGenre)}, list,
		"жанр достаётся своему фильму, порядок фильмов сохраняется")
}

func TestPostgresService_ListFilmsEmptyPage(t *testing.T) {
	service, mock := newPostgresService(t)

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT count`).WillReturnRows(countRows(13))
	mock.ExpectQuery(`ORDER BY f.id`).WillReturnRows(filmRows())
	mock.ExpectCommit()

	list, total, err := service.ListFilms(context.Background(), 20, 100)
	require.NoError(t, err)
	assert.Equal(t, 13, total)
	assert.NotNil(t, list)
	assert.Empty(t, list)
}

func TestPostgresService_ListFilmsErrors(t *testing.T) {
	interrupted := filmRows(shawshank(), ivan()).RowError(1, errStorage)

	tests := []struct {
		name  string
		setup func(sqlmock.Sqlmock)
	}{
		{"не удалось посчитать", func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectQuery(`SELECT count`).WillReturnError(errStorage)
			m.ExpectRollback()
		}},
		{"не удалось получить страницу", func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectQuery(`SELECT count`).WillReturnRows(countRows(13))
			m.ExpectQuery(`ORDER BY f.id`).WillReturnError(errStorage)
			m.ExpectRollback()
		}},
		{"чтение оборвалось на середине", func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectQuery(`SELECT count`).WillReturnRows(countRows(13))
			m.ExpectQuery(`ORDER BY f.id`).WillReturnRows(interrupted)
			m.ExpectRollback()
		}},
		{"не загрузились жанры", func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectQuery(`SELECT count`).WillReturnRows(countRows(13))
			m.ExpectQuery(`ORDER BY f.id`).WillReturnRows(filmRows(shawshank()))
			m.ExpectQuery(`FROM film_genre`).WillReturnError(errStorage)
			m.ExpectRollback()
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, mock := newPostgresService(t)
			tt.setup(mock)

			list, total, err := service.ListFilms(context.Background(), 20, 0)
			assert.ErrorIs(t, err, errStorage)
			assert.Nil(t, list)
			assert.Zero(t, total)
		})
	}
}

func TestPostgresService_GetCollectionByID(t *testing.T) {
	service, mock := newPostgresService(t)

	mock.ExpectQuery(`FROM collection AS c WHERE c.id = \$1`).
		WithArgs(domain.CollectionID(2)).
		WillReturnRows(sqlmock.NewRows(collectionColumns).
			AddRow(2, "Лучшие фильмы всех времён", "best-of-all-time", "Фильмы, которые стоит посмотреть.", nil))

	collection, err := service.GetCollectionByID(context.Background(), 2)
	require.NoError(t, err)
	assert.Equal(t, domain.Collection{
		ID:          2,
		Title:       "Лучшие фильмы всех времён",
		Slug:        "best-of-all-time",
		Description: ptr("Фильмы, которые стоит посмотреть."),
	}, collection, "у подборки без обложки CoverURL — nil")
}

func TestPostgresService_GetCollectionByIDErrors(t *testing.T) {
	tests := []struct {
		name    string
		rows    *sqlmock.Rows
		dbErr   error
		wantErr error
	}{
		{"подборки нет", sqlmock.NewRows(collectionColumns), nil, films.ErrCollectionNotFound},
		{"база недоступна", nil, errStorage, errStorage},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, mock := newPostgresService(t)
			expect := mock.ExpectQuery(`FROM collection AS c WHERE`)
			if tt.dbErr != nil {
				expect.WillReturnError(tt.dbErr)
			} else {
				expect.WillReturnRows(tt.rows)
			}

			_, err := service.GetCollectionByID(context.Background(), 2)
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestPostgresService_ListCollections(t *testing.T) {
	service, mock := newPostgresService(t)

	mock.ExpectBegin()
	mock.ExpectQuery(`count\(\*\) FROM collection\s*$`).WillReturnRows(countRows(5))
	mock.ExpectQuery(`FROM collection AS c\s+ORDER BY c.id LIMIT \$1 OFFSET \$2`).
		WithArgs(2, 1).
		WillReturnRows(sqlmock.NewRows(collectionColumns).
			AddRow(2, "Лучшие фильмы всех времён", "best-of-all-time", nil, nil).
			AddRow(3, "Смотрим всей семьёй", "family", nil, "/static/cover/family.jpg"))
	mock.ExpectCommit()

	list, total, err := service.ListCollections(context.Background(), 2, 1)
	require.NoError(t, err)
	assert.Equal(t, 5, total)
	require.Len(t, list, 2)
	assert.Equal(t, "best-of-all-time", list[0].Slug)
	assert.Equal(t, ptr("/static/cover/family.jpg"), list[1].CoverURL)
}

func TestPostgresService_ListCollectionsErrors(t *testing.T) {
	tests := []struct {
		name  string
		setup func(sqlmock.Sqlmock)
	}{
		{"не удалось посчитать", func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectQuery(`FROM collection`).WillReturnError(errStorage)
			m.ExpectRollback()
		}},
		{"не удалось получить страницу", func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectQuery(`FROM collection`).WillReturnRows(countRows(5))
			m.ExpectQuery(`ORDER BY c.id`).WillReturnError(errStorage)
			m.ExpectRollback()
		}},
		{"чтение оборвалось на середине", func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectQuery(`FROM collection`).WillReturnRows(countRows(5))
			m.ExpectQuery(`ORDER BY c.id`).WillReturnRows(sqlmock.NewRows(collectionColumns).
				AddRow(1, "Подборка", "slug", nil, nil).RowError(0, errStorage))
			m.ExpectRollback()
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, mock := newPostgresService(t)
			tt.setup(mock)

			list, total, err := service.ListCollections(context.Background(), 20, 0)
			assert.ErrorIs(t, err, errStorage)
			assert.Nil(t, list)
			assert.Zero(t, total)
		})
	}
}

func expectCollectionExists(mock sqlmock.Sqlmock, id domain.CollectionID) {
	mock.ExpectQuery(`FROM collection AS c WHERE c.id = \$1`).
		WithArgs(id).
		WillReturnRows(sqlmock.NewRows(collectionColumns).AddRow(id, "Подборка", "slug", nil, nil))
}

func TestPostgresService_ListCollectionFilms(t *testing.T) {
	service, mock := newPostgresService(t)

	expectCollectionExists(mock, 2)
	mock.ExpectBegin()
	mock.ExpectQuery(`count\(\*\) FROM collection_film`).
		WithArgs(domain.CollectionID(2)).
		WillReturnRows(countRows(6))
	mock.ExpectQuery(`FROM collection_film AS cf\s+JOIN film AS f.+ORDER BY cf.position\s+LIMIT \$2 OFFSET \$3`).
		WithArgs(domain.CollectionID(2), 2, 0).
		WillReturnRows(filmRows(ivan(), shawshank()))
	mock.ExpectQuery(`FROM film_genre`).
		WithArgs([]int64{11, 1}).
		WillReturnRows(sqlmock.NewRows(genreColumns).
			AddRow(1, crimeGenre.ID, crimeGenre.Name, crimeGenre.Slug))
	mock.ExpectCommit()

	list, total, err := service.ListCollectionFilms(context.Background(), 2, 2, 0)
	require.NoError(t, err)
	assert.Equal(t, 6, total)
	assert.Equal(t, []domain.Film{withGenres(ivan()), withGenres(shawshank(), crimeGenre)}, list,
		"фильмы идут в порядке подборки, а не по номеру")
}

func TestPostgresService_ListCollectionFilmsMissingCollection(t *testing.T) {
	service, mock := newPostgresService(t)
	mock.ExpectQuery(`FROM collection AS c WHERE c.id = \$1`).WillReturnRows(sqlmock.NewRows(collectionColumns))

	list, total, err := service.ListCollectionFilms(context.Background(), 999, 10, 0)
	assert.ErrorIs(t, err, films.ErrCollectionNotFound)
	assert.Nil(t, list)
	assert.Zero(t, total)
}

func TestPostgresService_ListCollectionFilmsErrors(t *testing.T) {
	tests := []struct {
		name  string
		setup func(sqlmock.Sqlmock)
	}{
		{"не удалось посчитать", func(m sqlmock.Sqlmock) {
			expectCollectionExists(m, 2)
			m.ExpectBegin()
			m.ExpectQuery(`FROM collection_film`).WillReturnError(errStorage)
			m.ExpectRollback()
		}},
		{"не удалось получить страницу", func(m sqlmock.Sqlmock) {
			expectCollectionExists(m, 2)
			m.ExpectBegin()
			m.ExpectQuery(`FROM collection_film`).WillReturnRows(countRows(6))
			m.ExpectQuery(`ORDER BY cf.position`).WillReturnError(errStorage)
			m.ExpectRollback()
		}},
		{"чтение оборвалось на середине", func(m sqlmock.Sqlmock) {
			expectCollectionExists(m, 2)
			m.ExpectBegin()
			m.ExpectQuery(`FROM collection_film`).WillReturnRows(countRows(6))
			m.ExpectQuery(`ORDER BY cf.position`).WillReturnRows(filmRows(shawshank()).RowError(0, errStorage))
			m.ExpectRollback()
		}},
		{"не загрузились жанры", func(m sqlmock.Sqlmock) {
			expectCollectionExists(m, 2)
			m.ExpectBegin()
			m.ExpectQuery(`FROM collection_film`).WillReturnRows(countRows(6))
			m.ExpectQuery(`ORDER BY cf.position`).WillReturnRows(filmRows(shawshank()))
			m.ExpectQuery(`FROM film_genre`).WillReturnError(errStorage)
			m.ExpectRollback()
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, mock := newPostgresService(t)
			tt.setup(mock)

			list, total, err := service.ListCollectionFilms(context.Background(), 2, 2, 0)
			assert.ErrorIs(t, err, errStorage)
			assert.Nil(t, list)
			assert.Zero(t, total)
		})
	}
}
