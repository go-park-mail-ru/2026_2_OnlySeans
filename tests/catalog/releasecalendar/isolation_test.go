package releasecalendar_test

import (
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/catalog/internal/domain"
)

func isolatedCalendarDatabase(t *testing.T, dsn string) *sql.DB {
	t.Helper()

	admin, err := sql.Open("pgx", dsn)
	require.NoError(t, err)

	schema := fmt.Sprintf("calendar_test_%d", time.Now().UnixNano())
	_, err = admin.Exec("CREATE SCHEMA " + schema)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, err := admin.Exec("DROP SCHEMA " + schema + " CASCADE")
		require.NoError(t, err)
		require.NoError(t, admin.Close())
	})

	parsed, err := url.Parse(dsn)
	require.NoError(t, err)

	query := parsed.Query()
	query.Set("search_path", schema+",public")
	parsed.RawQuery = query.Encode()
	db, err := sql.Open("pgx", parsed.String())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })

	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)

	root := filepath.Join(filepath.Dir(file), "../../..")
	migrations, err := os.ReadFile(filepath.Join(root, "db/apply_all.sql"))
	require.NoError(t, err)

	for _, line := range strings.Split(string(migrations), "\n") {
		if !strings.HasPrefix(line, `\ir `) {
			continue
		}

		path := filepath.Join(root, "db", strings.TrimPrefix(line, `\ir `))
		source, err := os.ReadFile(path)
		require.NoError(t, err)
		_, err = db.Exec(string(source))
		require.NoError(t, err, path)
	}

	return db
}

func TestPremiereDatesSurviveMigrationRoundTrip(t *testing.T) {
	dsn := os.Getenv("RELEASE_CALENDAR_TEST_DSN")
	if dsn == "" {
		t.Skip("set RELEASE_CALENDAR_TEST_DSN for PostgreSQL integration")
	}

	db := isolatedCalendarDatabase(t, dsn)
	_, err := db.Exec(`CREATE TABLE calendar_schema_migrations (version integer PRIMARY KEY)`)
	require.NoError(t, err)

	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)

	root := filepath.Join(filepath.Dir(file), "../../../services/catalog/db/migrations")

	for _, date := range []any{"2027-01-15", nil} {
		_, err = db.Exec(`UPDATE film SET world_premiere_date=$1,russia_premiere_date=$1 WHERE id=2`, date)
		require.NoError(t, err)

		for _, migration := range []string{"000020_release_date_sync.down.sql", "000019_release_calendar.down.sql", "000019_release_calendar.up.sql", "000020_release_date_sync.up.sql"} {
			source, err := os.ReadFile(filepath.Join(root, migration))
			require.NoError(t, err)
			_, err = db.Exec(string(source))
			require.NoError(t, err, migration)
		}
		var world, russia sql.NullString
		require.NoError(t, db.QueryRow(`SELECT world_premiere_date::text,russia_premiere_date::text FROM film WHERE id=2`).Scan(&world, &russia))

		if date == nil {
			require.False(t, world.Valid)
			require.False(t, russia.Valid)
		} else {
			require.Equal(t, date, world.String)
			require.Equal(t, date, russia.String)
		}
	}
}

func TestReleaseRangeIncludesLastSupportedDate(t *testing.T) {
	end, err := domain.ParseReleaseRangeEnd("2200-01-02")
	require.NoError(t, err)
	require.Equal(t, "2200-01-02", end.Format(time.DateOnly))

	_, err = domain.ParseReleaseDate("2200-01-02")
	require.ErrorIs(t, err, domain.ErrInvalidRelease)
}
