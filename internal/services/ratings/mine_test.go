package ratings

import (
	"context"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
)

func TestPersonalScoreIsScopedToAccountAndFilm(t *testing.T) {
	for _, tc := range []struct {
		name    string
		present bool
		fail    bool
	}{
		{"saved", true, false}, {"not yet rated", false, false}, {"database failure", false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			expected := mock.ExpectQuery(regexp.QuoteMeta("SELECT score FROM film_rating WHERE account_id = $1 AND film_id = $2")).WithArgs(domain.UserID(7), domain.FilmID(42))
			storageErr := errors.New("unavailable")
			if tc.fail {
				expected.WillReturnError(storageErr)
			} else {
				rows := sqlmock.NewRows([]string{"score"})
				if tc.present {
					rows.AddRow(8)
				}
				expected.WillReturnRows(rows)
			}
			score, err := (&PostgresRepo{db: db}).GetMyScore(context.Background(), 7, 42)
			if tc.fail {
				if !errors.Is(err, storageErr) {
					t.Fatalf("error=%v", err)
				}
			} else if err != nil || (score != nil) != tc.present {
				t.Fatalf("score=%v, error=%v", score, err)
			}
			if tc.present && *score != 8 {
				t.Fatalf("score=%d", *score)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
