package persons_test

import (
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	personsservice "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/services/persons"
	"github.com/stretchr/testify/require"
)

func TestPostgresPersonRepoCreate(t *testing.T) {
	databaseErr := errors.New("database unavailable")
	tests := []struct {
		name  string
		id    domain.PersonID
		dbErr error
	}{
		{
			name: "returns generated ID without loading person",
			id:   42,
		},
		{
			name:  "database error",
			dbErr: databaseErr,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()
			repo := personsservice.NewPersonRepo(db)
			query := mock.ExpectQuery("INSERT INTO person").WithArgs("Ada", "Lovelace", nil)
			if tt.dbErr != nil {
				query.WillReturnError(tt.dbErr)
			} else {
				query.WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(tt.id))
			}
			id, err := repo.Create(t.Context(), domain.PersonInput{
				FirstName: "Ada",
				LastName:  "Lovelace",
			})
			if tt.dbErr != nil {
				require.ErrorIs(t, err, tt.dbErr)
				require.Zero(t, id)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.id, id)
			}
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
