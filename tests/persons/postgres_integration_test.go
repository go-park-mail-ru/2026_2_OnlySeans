//go:build integration

package persons_test

import (
	"context"
	"database/sql"
	"io"
	"os"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	personsservice "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/services/persons"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/require"
)

func TestPostgresPersonRepoUpdate(t *testing.T) {
	dsn := os.Getenv("PERSONS_TEST_DATABASE_URL")
	require.NotEmpty(t, dsn, "set PERSONS_TEST_DATABASE_URL for integration tests")
	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
	defer cancel()
	db, err := sql.Open("pgx", dsn)
	require.NoError(t, err)
	defer db.Close()
	// Temporary tables and all repository calls use the same connection.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	for _, query := range []string{
		`CREATE TEMP TABLE file (id bigint PRIMARY KEY, external_url text, storage_key text)`,
		`CREATE TEMP TABLE person (
   id bigint PRIMARY KEY, first_name text, last_name text, birth_date date,
   photo_file_id bigint, created_at timestamptz DEFAULT now(), updated_at timestamptz DEFAULT now()
  )`,
		`INSERT INTO file VALUES (1, 'https://example.com/photo.jpg', NULL)`,
		`INSERT INTO person (id, first_name, last_name, photo_file_id) VALUES (1, 'Old', 'Name', 1)`,
	} {
		_, err := db.ExecContext(ctx, query)
		require.NoError(t, err)
	}
	appLogger, err := logger.NewLogger(io.Discard, logger.LogLevelInfo, "text")
	require.NoError(t, err)
	repo := personsservice.NewPersonRepo(db, appLogger)
	birthDate := time.Date(1990, time.January, 2, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name    string
		id      domain.PersonID
		input   domain.PersonInput
		wantErr error
	}{
		{
			name:  "update with birth date and photo",
			id:    1,
			input: domain.PersonInput{FirstName: "Anna", LastName: "Smith", BirthDate: &birthDate},
		},
		{
			name:  "clear birth date",
			id:    1,
			input: domain.PersonInput{FirstName: "Jane", LastName: "Smith"},
		},
		{
			name:    "missing person",
			id:      999,
			input:   domain.PersonInput{FirstName: "Jane", LastName: "Smith"},
			wantErr: domain.ErrPersonNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			person, err := repo.Update(ctx, tt.id, tt.input)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.input.FirstName, person.FirstName)
			require.Equal(t, tt.input.LastName, person.LastName)
			require.NotNil(t, person.PhotoID)
			require.Equal(t, "https://example.com/photo.jpg", *person.PhotoID)
			stored, err := repo.GetByID(ctx, tt.id)
			require.NoError(t, err)
			require.Equal(t, person, stored)
			if tt.input.BirthDate == nil {
				require.Nil(t, person.BirthDate)
			} else {
				require.NotNil(t, person.BirthDate)
				require.True(t, person.BirthDate.Equal(*tt.input.BirthDate))
			}
		})
	}
}
