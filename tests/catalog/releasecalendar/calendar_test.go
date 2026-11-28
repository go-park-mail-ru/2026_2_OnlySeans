package releasecalendar_test

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/require"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/catalog/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/catalog/internal/repository/postgres"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/catalog/internal/usecase"
)

func ptr[T any](v T) *T { return &v }
func day(t *testing.T, value string) time.Time {
	t.Helper()

	date, err := domain.ParseReleaseDate(value)
	require.NoError(t, err)

	return date
}

func TestReleaseInputValidation(t *testing.T) {
	for _, date := range []string{"2026-02-29", "2026-13-01", "2026-12-01T00:00:00Z", "1800-01-01", "2200-01-02"} {
		_, err := domain.ParseReleaseDate(date)
		require.ErrorIs(t, err, domain.ErrInvalidRelease, date)
	}

	require.NoError(t, (domain.ReleaseInput{
		WorldPremiereDate:    ptr("2028-02-29"),
		WorldPremiereDateSet: true,
	}).Validate())
	require.NoError(t, (domain.ReleaseInput{RussiaPremiereDateSet: true}).Validate())
	require.NoError(t, (domain.ReleaseInput{
		WorldPremiereDateSet:  true,
		RussiaPremiereDateSet: true,
	}).Validate())
	require.ErrorIs(t, (domain.ReleaseInput{}).Validate(), domain.ErrInvalidRelease)
	require.ErrorIs(t, (domain.ReleaseInput{RussiaPremiereDate: ptr("2028-02-29")}).Validate(), domain.ErrInvalidRelease)
	require.ErrorIs(t, (domain.ReleaseInput{
		RussiaPremiereDate:    ptr("2026-02-29"),
		RussiaPremiereDateSet: true,
	}).Validate(), domain.ErrInvalidRelease)
}

func TestCalendarPostgresLifecycle(t *testing.T) {
	dsn := os.Getenv("RELEASE_CALENDAR_TEST_DSN")
	if dsn == "" {
		t.Skip("set RELEASE_CALENDAR_TEST_DSN for PostgreSQL integration")
	}

	db := isolatedCalendarDatabase(t, dsn)
	var err error

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	repo := postgres.NewReleaseRepo(db, nil)
	uc := usecase.NewReleaseUseCase(repo, nil)
	var user, other, film int64
	suffix := fmt.Sprint(time.Now().UnixNano())
	require.NoError(t, db.QueryRowContext(ctx, `INSERT INTO account (role_id,email,username,password_hash) SELECT id,$1,$2,repeat('x',60) FROM role ORDER BY id LIMIT 1 RETURNING id`, suffix+"@calendar.test", "c"+suffix).Scan(&user))
	t.Cleanup(func() {
		_, err := db.Exec(`DELETE FROM account_role_history WHERE account_id=$1`, user)
		require.NoError(t, err)
		_, err = db.Exec(`DELETE FROM account WHERE id=$1`, user)
		require.NoError(t, err)
	})
	require.NoError(t, db.QueryRowContext(ctx, `INSERT INTO account (role_id,email,username,password_hash) SELECT id,$1,$2,repeat('x',60) FROM role ORDER BY id LIMIT 1 RETURNING id`, suffix+"b@calendar.test", "d"+suffix).Scan(&other))
	t.Cleanup(func() {
		_, err := db.Exec(`DELETE FROM account_role_history WHERE account_id=$1`, other)
		require.NoError(t, err)
		_, err = db.Exec(`DELETE FROM account WHERE id=$1`, other)
		require.NoError(t, err)
	})
	require.NoError(t, db.QueryRowContext(ctx, `INSERT INTO film (title,film_type,production_year,age_limit) VALUES ($1,'movie',2026,12) RETURNING id`, "Calendar test "+suffix).Scan(&film))
	t.Cleanup(func() { _, err := db.Exec(`DELETE FROM film WHERE id=$1`, film); require.NoError(t, err) })
	var genre int64
	require.NoError(t, db.QueryRowContext(ctx, `SELECT id FROM genre ORDER BY id LIMIT 1`).Scan(&genre))
	_, err = db.ExecContext(ctx, `INSERT INTO film_genre (film_id,genre_id) VALUES ($1,$2)`, film, genre)
	require.NoError(t, err)

	today := time.Now().In(domain.ReleaseLocation)
	tomorrow := today.AddDate(0, 0, 1).Format(time.DateOnly)
	next := today.AddDate(0, 0, 2).Format(time.DateOnly)
	items, total, err := uc.List(ctx, domain.ReleaseFilter{FilmID: domain.FilmID(film)}, 0)
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Nil(t, items[0].WorldPremiereDate)
	require.Nil(t, items[0].RussiaPremiereDate)
	require.ErrorIs(t, uc.SetSubscription(ctx, user, film, true), domain.ErrReleaseUnavailable)
	require.NoError(t, uc.Update(ctx, film, domain.ReleaseInput{
		WorldPremiereDate:    &tomorrow,
		WorldPremiereDateSet: true,
	}))

	f := domain.ReleaseFilter{
		From:    day(t, tomorrow),
		To:      day(t, next),
		GenreID: domain.GenreID(genre),
	}
	items, _, err = uc.List(ctx, f, 0)
	require.NoError(t, err)

	for _, item := range items {
		require.NotEqualValues(t, film, item.Film.ID)
	}

	worldFilter := f
	worldFilter.Region = domain.ReleaseRegionWorld
	worldFilter.FilmID = domain.FilmID(film)
	items, total, err = uc.List(ctx, worldFilter, 0)
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.EqualValues(t, film, items[0].Film.ID)
	worldFilter.From = day(t, next)
	worldFilter.To = worldFilter.From.AddDate(0, 0, 1)
	items, total, err = uc.List(ctx, worldFilter, 0)
	require.NoError(t, err)
	require.Zero(t, total)
	require.Empty(t, items)

	require.NoError(t, uc.Update(ctx, film, domain.ReleaseInput{
		RussiaPremiereDate:    &tomorrow,
		RussiaPremiereDateSet: true,
	}))
	items, _, err = uc.List(ctx, f, 0)
	require.NoError(t, err)

	found := false

	for _, item := range items {
		if item.Film.ID == domain.FilmID(film) {
			found = true

			require.Equal(t, tomorrow, item.WorldPremiereDate.Format(time.DateOnly))
			require.Equal(t, tomorrow, item.RussiaPremiereDate.Format(time.DateOnly))
			require.NotEmpty(t, item.Film.Genres)
		}
	}

	require.True(t, found)
	require.NoError(t, uc.SetSubscription(ctx, user, film, true))
	require.NoError(t, uc.SetSubscription(ctx, user, film, true))
	require.NoError(t, uc.Update(ctx, film, domain.ReleaseInput{
		RussiaPremiereDate:    &next,
		RussiaPremiereDateSet: true,
	}))
	items, total, err = uc.List(ctx, domain.ReleaseFilter{}, user)
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.EqualValues(t, film, items[0].Film.ID)
	require.Equal(t, tomorrow, items[0].WorldPremiereDate.Format(time.DateOnly))
	require.Equal(t, next, items[0].RussiaPremiereDate.Format(time.DateOnly))
	items, _, err = uc.List(ctx, f, 0)
	require.NoError(t, err)

	for _, item := range items {
		require.NotEqualValues(t, film, item.Film.ID)
	}

	f.To = f.From
	_, _, err = uc.List(ctx, f, 0)
	require.ErrorIs(t, err, domain.ErrInvalidRelease)

	f.To = f.From.AddDate(0, 0, 63)
	_, _, err = uc.List(ctx, f, 0)
	require.ErrorIs(t, err, domain.ErrInvalidRelease)
	require.NoError(t, uc.Update(ctx, film, domain.ReleaseInput{RussiaPremiereDateSet: true}))
	items, total, err = uc.List(ctx, domain.ReleaseFilter{}, user)
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Nil(t, items[0].RussiaPremiereDate)
	require.NoError(t, uc.SetSubscription(ctx, user, film, true))
	require.ErrorIs(t, uc.SetSubscription(ctx, other, film, true), domain.ErrReleaseUnavailable)

	after := time.Date(today.Year(), today.Month(), today.Day(), 9, 0, 0, 0, domain.ReleaseLocation).AddDate(0, 0, 2)
	count, err := repo.DeliverReminders(ctx, after)
	require.NoError(t, err)
	require.Zero(t, count)
	require.NoError(t, uc.Update(ctx, film, domain.ReleaseInput{
		RussiaPremiereDate:    &next,
		RussiaPremiereDateSet: true,
	}))
	require.NoError(t, uc.SetSubscription(ctx, user, film, false))
	require.NoError(t, uc.SetSubscription(ctx, user, film, false))
	count, err = repo.DeliverReminders(ctx, after)
	require.NoError(t, err)
	require.Zero(t, count)
	require.NoError(t, uc.SetSubscription(ctx, user, film, true))
	count, err = repo.DeliverReminders(ctx, after.Add(-time.Minute))
	require.NoError(t, err)
	require.Zero(t, count)
	var wg sync.WaitGroup
	errs := make(chan error, 2)

	for range 2 {
		wg.Go(func() { _, err := repo.DeliverReminders(ctx, after); errs <- err })
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		require.NoError(t, err)
	}

	notes, n, err := uc.Notifications(ctx, user, 20, 0)
	require.NoError(t, err)
	require.EqualValues(t, 1, n)
	require.Equal(t, film, *notes[0].FilmID)

	count, err = repo.DeliverReminders(ctx, after.Add(48*time.Hour))
	require.NoError(t, err)
	require.Zero(t, count)
	require.ErrorIs(t, uc.ReadNotification(ctx, other, notes[0].ID), domain.ErrNotificationNotFound)
	require.NoError(t, uc.ReadNotification(ctx, user, notes[0].ID))
	require.NoError(t, uc.ReadNotification(ctx, user, notes[0].ID))
	notes, _, err = uc.Notifications(ctx, user, 20, 0)
	require.NoError(t, err)
	require.True(t, notes[0].IsRead)
	require.NoError(t, uc.Update(ctx, film, domain.ReleaseInput{WorldPremiereDateSet: true}))
	items, _, err = uc.List(ctx, domain.ReleaseFilter{}, user)
	require.NoError(t, err)
	require.Nil(t, items[0].WorldPremiereDate)
	require.Equal(t, next, items[0].RussiaPremiereDate.Format(time.DateOnly))
	require.NoError(t, uc.SetSubscription(ctx, user, film, false))
	_, err = db.ExecContext(ctx, `DELETE FROM notification WHERE account_id=$1`, user)
	require.NoError(t, err)
}
