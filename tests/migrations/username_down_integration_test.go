//go:build integration

package migrations_test

import (
	"database/sql"
	"errors"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/require"
)

func TestUsernameRollbackPreservesNames(t *testing.T) {
	dsn := os.Getenv("MIGRATIONS_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set MIGRATIONS_TEST_DATABASE_URL for the PostgreSQL integration test")
	}
	migration, err := os.ReadFile("../../db/migrations/000012_account_username.down.sql")
	require.NoError(t, err)
	for _, tc := range []struct {
		name, username string
		compatible     bool
	}{
		{"legacy name can roll back", "anna_17", true},
		{"Unicode name is preserved", "Анна", false},
		{"two-character name is preserved", "ab", false},
		{"capital letters are preserved", "Anna", false},
		{"spaces and hyphens are preserved", "Anna Maria-Smith", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db, err := sql.Open("pgx", dsn)
			require.NoError(t, err)
			defer db.Close()
			db.SetMaxOpenConns(1)
			db.SetMaxIdleConns(1)
			_, err = db.ExecContext(t.Context(), `CREATE TEMP TABLE account (
    username text NOT NULL,
    CONSTRAINT account_username_check CHECK (length(username) BETWEEN 2 AND 32 AND username = btrim(username))
   )`)
			require.NoError(t, err)
			_, err = db.ExecContext(t.Context(), `INSERT INTO account (username) VALUES ($1)`, tc.username)
			require.NoError(t, err)
			_, err = db.ExecContext(t.Context(), string(migration))
			if tc.compatible {
				require.NoError(t, err)
			} else {
				var pgErr *pgconn.PgError
				require.True(t, errors.As(err, &pgErr))
				require.Equal(t, "23514", pgErr.Code)
				require.Contains(t, pgErr.Message, "without changing existing usernames")
			}
			var username string
			require.NoError(t, db.QueryRowContext(t.Context(), `SELECT username FROM account`).Scan(&username))
			require.Equal(t, tc.username, username)
			_, err = db.ExecContext(t.Context(), `INSERT INTO account (username) VALUES ('Новый')`)
			if tc.compatible {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
