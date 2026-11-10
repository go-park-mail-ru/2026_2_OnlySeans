package files_test

import (
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	files "github.com/go-park-mail-ru/2026_2_OnlySeans/internal/services/files"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
	"github.com/stretchr/testify/require"
)

func TestFileRepositoriesRejectNilDependencies(t *testing.T) {
	constructors := []struct {
		name   string
		create func(*sql.DB, logger.Logger) error
	}{
		{"avatar", func(db *sql.DB, appLogger logger.Logger) error {
			repo, err := files.NewPostgresAvatarRepo(db, appLogger)
			require.Nil(t, repo)
			return err
		}},
		{"poster", func(db *sql.DB, appLogger logger.Logger) error {
			repo, err := files.NewPostgresPosterRepo(db, appLogger)
			require.Nil(t, repo)
			return err
		}},
	}
	for _, constructor := range constructors {
		t.Run(constructor.name, func(t *testing.T) {
			require.Error(t, constructor.create(nil, testFilesLogger(t)))
			db, _, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()
			require.ErrorIs(t, constructor.create(db, nil), domain.ErrInvalidAppLogger)
		})
	}
}
