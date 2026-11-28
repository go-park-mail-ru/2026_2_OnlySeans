package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/services/catalog/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/shared/pkg/logger"
)

const (
	releaseReminderBatchSize = 500

	releaseFilmExistsQuery = `SELECT id FROM film WHERE id=$1`

	releaseDateQuery = `(CASE WHEN $6::boolean THEN f.world_premiere_date ELSE f.russia_premiere_date END)`

	releaseFilterQuery = ` FROM film f
	WHERE ($1::date IS NULL OR ` + releaseDateQuery + ` >= $1)
	AND ($2::date IS NULL OR ` + releaseDateQuery + ` < $2)
	AND ($3::bigint=0 OR f.id=$3)
	AND ($4::bigint=0 OR EXISTS (SELECT 1 FROM film_genre g WHERE g.film_id=f.id AND g.genre_id=$4))
	AND ($5::bigint=0 OR EXISTS (SELECT 1 FROM release_subscription s WHERE s.film_id=f.id AND s.account_id=$5))
	`

	countReleasesQuery = `SELECT count(*)` + releaseFilterQuery

	listReleasesQuery = `SELECT f.world_premiere_date,f.russia_premiere_date,` + filmColumns + releaseFilterQuery +
		` ORDER BY ` + releaseDateQuery + ` NULLS LAST,f.id LIMIT $7 OFFSET $8`

	updateReleaseDatesQuery = `UPDATE film SET
	world_premiere_date=CASE WHEN $2 THEN $3::date ELSE world_premiere_date END,
	russia_premiere_date=CASE WHEN $4 THEN $5::date ELSE russia_premiere_date END
	WHERE id=$1
	`

	deleteReleaseSubscriptionQuery = `DELETE FROM release_subscription WHERE account_id=$1 AND film_id=$2`

	checkReleaseSubscriptionQuery = `SELECT COALESCE(russia_premiere_date >= (now() AT TIME ZONE 'Europe/Moscow')::date,false) FROM film WHERE id=$1 FOR SHARE`

	releaseSubscriptionExistsQuery = `SELECT EXISTS (SELECT 1 FROM release_subscription WHERE account_id=$1 AND film_id=$2)`

	insertReleaseSubscriptionQuery = `INSERT INTO release_subscription (account_id,film_id) VALUES ($1,$2) ON CONFLICT DO NOTHING`

	countReleaseNotificationsQuery = `SELECT count(*) FROM notification WHERE account_id=$1`

	listReleaseNotificationsQuery = `SELECT id,message,is_read,created_at,film_id,notification_type FROM notification WHERE account_id=$1 ORDER BY created_at DESC,id DESC LIMIT $2 OFFSET $3`

	markReleaseNotificationReadQuery = `UPDATE notification SET is_read=true WHERE id=$1 AND account_id=$2`

	deliverReleaseRemindersQuery = `WITH due AS MATERIALIZED (
	SELECT s.account_id,f.id AS film_id,f.title,f.russia_premiere_date
	FROM release_subscription s JOIN film f ON f.id=s.film_id
	WHERE (f.russia_premiere_date + TIME '09:00') AT TIME ZONE 'Europe/Moscow' <= $1
	AND f.russia_premiere_date >= (s.created_at AT TIME ZONE 'Europe/Moscow')::date
	AND NOT EXISTS (SELECT 1 FROM notification n WHERE n.account_id=s.account_id AND n.film_id=f.id AND n.notification_type='release_reminder')
	ORDER BY f.russia_premiere_date,f.id,s.account_id LIMIT $2 FOR SHARE OF f,s
	) INSERT INTO notification (account_id,message,film_id,notification_type)
	SELECT account_id,'Премьера «' || title || '» в России — ' || to_char(russia_premiere_date,'DD.MM.YYYY') || '.',film_id,'release_reminder' FROM due
	ON CONFLICT (account_id,film_id,notification_type) WHERE film_id IS NOT NULL DO NOTHING
	`
)

type ReleaseRepo struct {
	db        *sql.DB
	appLogger logger.Logger
}

func NewReleaseRepo(db *sql.DB, appLogger logger.Logger) *ReleaseRepo {
	return &ReleaseRepo{db: db, appLogger: appLogger}
}

func releaseDBError(err error) error {
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		switch pg.Code {
		case "23505", "23503", "23514", "23502":
			return domain.ErrInvalidRelease
		}
	}

	return err
}

func dateArgument(t time.Time) any {
	if t.IsZero() {
		return nil
	}

	return t.Format(time.DateOnly)
}

func (r *ReleaseRepo) List(ctx context.Context, f domain.ReleaseFilter, user int64) ([]domain.Release, int64, error) {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return nil, 0, err
	}

	defer func() { _ = tx.Rollback() }()

	if f.FilmID > 0 {
		var id int64
		if err = tx.QueryRowContext(ctx, releaseFilmExistsQuery, f.FilmID).Scan(&id); errors.Is(err, sql.ErrNoRows) {
			return nil, 0, domain.ErrFilmNotFound
		} else if err != nil {
			return nil, 0, err
		}
	}

	args := []any{
		dateArgument(f.From),
		dateArgument(f.To),
		f.FilmID,
		f.GenreID,
		user,
		f.Region == domain.ReleaseRegionWorld,
	}
	var total int64

	if err = tx.QueryRowContext(ctx, countReleasesQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	args = append(args, f.Limit, f.Offset)

	rows, err := tx.QueryContext(ctx, listReleasesQuery, args...)
	if err != nil {
		return nil, 0, err
	}

	items := []domain.Release{}

	for rows.Next() {
		var event domain.Release
		film := &event.Film

		err = rows.Scan(&event.WorldPremiereDate, &event.RussiaPremiereDate,
			&film.ID, &film.Title, &film.OriginalTitle, &film.FilmType, &film.ReleaseYear, &film.DurationMin, &film.AgeLimit, &film.Description, &film.PosterURL, &film.TrailerURL, &film.PosterStorageKey)
		if err != nil {
			_ = rows.Close()
			return nil, 0, err
		}

		items = append(items, event)
	}

	err = rows.Err()
	_ = rows.Close()

	if err != nil {
		return nil, 0, err
	}

	films := []domain.Film{}

	positions := map[domain.FilmID]int{}
	for _, event := range items {
		if _, ok := positions[event.Film.ID]; !ok {
			positions[event.Film.ID] = len(films)
			films = append(films, event.Film)
		}
	}

	if err = attachGenresTx(ctx, tx, films, r.appLogger); err != nil {
		return nil, 0, err
	}

	for i := range items {
		items[i].Film = films[positions[items[i].Film.ID]]
	}

	if err = tx.Commit(); err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

func (r *ReleaseRepo) Update(ctx context.Context, id int64, in domain.ReleaseInput) error {
	result, err := r.db.ExecContext(ctx, updateReleaseDatesQuery, id, in.WorldPremiereDateSet, in.WorldPremiereDate, in.RussiaPremiereDateSet, in.RussiaPremiereDate)
	if err != nil {
		return releaseDBError(err)
	}

	n, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if n == 0 {
		return domain.ErrFilmNotFound
	}

	return nil
}

func (r *ReleaseRepo) SetSubscription(ctx context.Context, user, id int64, on bool) error {
	if !on {
		_, err := r.db.ExecContext(ctx, deleteReleaseSubscriptionQuery, user, id)
		return err
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() { _ = tx.Rollback() }()
	var available bool

	err = tx.QueryRowContext(ctx, checkReleaseSubscriptionQuery, id).Scan(&available)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ErrFilmNotFound
	}

	if err != nil {
		return err
	}

	var subscribed bool
	if err = tx.QueryRowContext(ctx, releaseSubscriptionExistsQuery, user, id).Scan(&subscribed); err != nil {
		return err
	}

	if subscribed {
		return tx.Commit()
	}

	if !available {
		return domain.ErrReleaseUnavailable
	}

	_, err = tx.ExecContext(ctx, insertReleaseSubscriptionQuery, user, id)
	if err != nil {
		return releaseDBError(err)
	}

	return tx.Commit()
}

func (r *ReleaseRepo) Notifications(ctx context.Context, user int64, limit, offset int) ([]domain.Notification, int64, error) {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return nil, 0, err
	}

	defer func() { _ = tx.Rollback() }()

	var total int64
	if err = tx.QueryRowContext(ctx, countReleaseNotificationsQuery, user).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := tx.QueryContext(ctx, listReleaseNotificationsQuery, user, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	items := []domain.Notification{}

	for rows.Next() {
		var n domain.Notification
		if err = rows.Scan(&n.ID, &n.Message, &n.IsRead, &n.CreatedAt, &n.FilmID, &n.Type); err != nil {
			_ = rows.Close()
			return nil, 0, err
		}

		items = append(items, n)
	}

	err = rows.Err()
	_ = rows.Close()

	if err != nil {
		return nil, 0, err
	}

	if err = tx.Commit(); err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

func (r *ReleaseRepo) ReadNotification(ctx context.Context, user, id int64) error {
	result, err := r.db.ExecContext(ctx, markReleaseNotificationReadQuery, id, user)
	if err != nil {
		return err
	}

	n, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if n == 0 {
		return domain.ErrNotificationNotFound
	}

	return nil
}

func (r *ReleaseRepo) DeliverReminders(ctx context.Context, now time.Time) (int64, error) {
	result, err := r.db.ExecContext(ctx, deliverReleaseRemindersQuery, now, releaseReminderBatchSize)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}
