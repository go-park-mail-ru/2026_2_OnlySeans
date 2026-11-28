package releasecalendar_test

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/stretchr/testify/require"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/filecleanup"
)

type cleanupStorage struct {
	keys []string
	err  error
}

func (s *cleanupStorage) DeleteFile(_ context.Context, key string) error {
	s.keys = append(s.keys, key)
	return s.err
}

func cleanupDB(t *testing.T) *sql.DB {
	t.Helper()

	dsn := os.Getenv("RELEASE_CALENDAR_TEST_DSN")
	if dsn == "" {
		t.Skip("set RELEASE_CALENDAR_TEST_DSN for PostgreSQL integration")
	}

	return isolatedCalendarDatabase(t, dsn)
}
func cleanupFile(t *testing.T, db *sql.DB, key string) int64 {
	t.Helper()
	var id int64
	require.NoError(t, db.QueryRow(`INSERT INTO file(storage_key,mime_type,size_bytes) VALUES($1,'image/png',10) RETURNING id`, key).Scan(&id))

	return id
}
func execCleanup(t *testing.T, db *sql.DB, q string, args ...any) {
	t.Helper()

	_, err := db.Exec(q, args...)
	require.NoError(t, err)
}

func TestFileCleanupPreservesSharedFileAndRetriesStorageFailure(t *testing.T) {
	db := cleanupDB(t)
	id := cleanupFile(t, db, "shared-avatar.png")
	execCleanup(t, db, `UPDATE account SET avatar_file_id=$1 WHERE id=1`, id)
	execCleanup(t, db, `UPDATE person SET photo_file_id=$1 WHERE id=1`, id)
	execCleanup(t, db, `UPDATE account SET avatar_file_id=NULL WHERE id=1`)

	storage := &cleanupStorage{}
	worker := filecleanup.New(db, storage)
	done, err := worker.ProcessOne(t.Context())
	require.True(t, done)
	require.NoError(t, err)
	require.Empty(t, storage.keys)
	var count int
	require.NoError(t, db.QueryRow(`SELECT count(*) FROM file WHERE id=$1`, id).Scan(&count))
	require.Equal(t, 1, count)
	execCleanup(t, db, `UPDATE person SET photo_file_id=NULL WHERE id=1`)

	storage.err = errors.New("S3 temporarily unavailable")
	done, err = worker.ProcessOne(t.Context())
	require.True(t, done)
	require.ErrorIs(t, err, storage.err)
	var attempts int
	require.NoError(t, db.QueryRow(`SELECT attempts FROM file_cleanup_queue WHERE file_id=$1`, id).Scan(&attempts))
	require.Equal(t, 1, attempts)
	require.NoError(t, db.QueryRow(`SELECT count(*) FROM file WHERE id=$1`, id).Scan(&count))
	require.Equal(t, 1, count)

	storage.err = nil

	execCleanup(t, db, `UPDATE file_cleanup_queue SET next_attempt_at=now() WHERE file_id=$1`, id)
	done, err = worker.ProcessOne(t.Context())
	require.True(t, done)
	require.NoError(t, err)
	require.Equal(t, []string{"shared-avatar.png", "shared-avatar.png"}, storage.keys)
	require.NoError(t, db.QueryRow(`SELECT count(*) FROM file WHERE id=$1`, id).Scan(&count))
	require.Zero(t, count)
	require.NoError(t, db.QueryRow(`SELECT count(*) FROM file_cleanup_queue WHERE file_id=$1`, id).Scan(&count))
	require.Zero(t, count)
}

func TestPosterReplacementQueuesOnlyDetachedFilesAndRollbackKeepsBinding(t *testing.T) {
	db := cleanupDB(t)
	first := cleanupFile(t, db, "first.png")
	second := cleanupFile(t, db, "second.png")
	execCleanup(t, db, `DELETE FROM film_file WHERE film_id=1 AND file_role='poster'`)
	execCleanup(t, db, `DELETE FROM file_cleanup_queue`)
	execCleanup(t, db, `INSERT INTO film_file(film_id,file_id,file_role,sort_order) VALUES(1,$1,'poster',1)`, first)
	tx, err := db.BeginTx(t.Context(), nil)
	require.NoError(t, err)
	_, err = tx.Exec(`UPDATE film_file SET file_id=$1 WHERE film_id=1 AND file_role='poster'`, second)
	require.NoError(t, err)
	require.NoError(t, tx.Rollback())
	var count int
	require.NoError(t, db.QueryRow(`SELECT count(*) FROM file_cleanup_queue`).Scan(&count))
	require.Zero(t, count)
	execCleanup(t, db, `UPDATE film_file SET file_id=$1 WHERE film_id=1 AND file_role='poster'`, second)

	storage := &cleanupStorage{}
	worker := filecleanup.New(db, storage)
	done, err := worker.ProcessOne(t.Context())
	require.True(t, done)
	require.NoError(t, err)
	require.Equal(t, []string{"first.png"}, storage.keys)
	execCleanup(t, db, `DELETE FROM film_file WHERE film_id=1 AND file_role='poster'`)
	done, err = worker.ProcessOne(t.Context())
	require.True(t, done)
	require.NoError(t, err)
	require.Equal(t, []string{"first.png", "second.png"}, storage.keys)
}

func TestRetiredPremiereTriggerPreservesRerunsAndCountry(t *testing.T) {
	db := cleanupDB(t)
	execCleanup(t, db, `DELETE FROM film_release WHERE film_id=1`)
	execCleanup(t, db, `INSERT INTO film_release(film_id,country_id,release_type,release_date)
 SELECT 1,id,'world_premiere','2020-01-01' FROM country WHERE iso_code='US'`)
	execCleanup(t, db, `INSERT INTO film_release(film_id,country_id,release_type,release_date)
 SELECT 1,id,'cinema',d::date FROM country CROSS JOIN unnest(ARRAY['2020-02-01','2025-02-01']) d WHERE iso_code='RU'`)

	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)

	root := filepath.Join(filepath.Dir(file), "../../../services/catalog/db/migrations")
	for _, name := range []string{"000021_preserve_release_history.up.sql"} {
		data, err := os.ReadFile(filepath.Join(root, name))
		require.NoError(t, err)
		execCleanup(t, db, string(data))
	}

	execCleanup(t, db, `UPDATE film SET world_premiere_date='2027-01-01',russia_premiere_date=NULL WHERE id=1`)
	var count int
	require.NoError(t, db.QueryRow(`SELECT count(*) FROM film_release WHERE film_id=1`).Scan(&count))
	require.Equal(t, 3, count)
	var country string
	require.NoError(t, db.QueryRow(`SELECT c.iso_code FROM film_release r JOIN country c ON c.id=r.country_id WHERE film_id=1 AND release_type='world_premiere'`).Scan(&country))
	require.Equal(t, "US", country)
}

type blockingCleanupStorage struct{ started, proceed chan struct{} }

func (s blockingCleanupStorage) DeleteFile(ctx context.Context, _ string) error {
	close(s.started)
	select {
	case <-s.proceed:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func TestCleanupLocksOutConcurrentAttachment(t *testing.T) {
	db := cleanupDB(t)
	id := cleanupFile(t, db, "racing.png")
	execCleanup(t, db, `UPDATE account SET avatar_file_id=$1 WHERE id=1`, id)
	execCleanup(t, db, `UPDATE account SET avatar_file_id=NULL WHERE id=1`)

	storage := blockingCleanupStorage{
		started: make(chan struct{}),
		proceed: make(chan struct{}),
	}

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	cleanDone := make(chan error, 1)

	go func() { _, err := filecleanup.New(db, storage).ProcessOne(ctx); cleanDone <- err }()
	select {
	case <-storage.started:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}

	attached := make(chan error, 1)

	go func() {
		_, err := db.ExecContext(ctx, `UPDATE person SET photo_file_id=$1 WHERE id=1`, id)
		attached <- err
	}()
	select {
	case err := <-attached:
		t.Fatalf("attachment must wait while object is being deleted: %v", err)
	case <-time.After(100 * time.Millisecond):
	}
	close(storage.proceed)
	require.NoError(t, <-cleanDone)

	err := <-attached
	var pgErr *pgconn.PgError
	require.ErrorAs(t, err, &pgErr)
	require.Equal(t, "23503", pgErr.Code, "concurrent attachment must fail its FK rather than create a broken reference")
}
