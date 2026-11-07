package profile

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/storage"
)

var ErrPasswordChanged = errors.New("password changed concurrently")

type Account struct {
	ID        domain.UserID `json:"id"`
	Username  string        `json:"username"`
	Email     string        `json:"email"`
	CreatedAt time.Time     `json:"created_at"`
}

type Film struct {
	ID       int64   `json:"id"`
	Title    string  `json:"title"`
	Year     int     `json:"year"`
	Type     string  `json:"type"`
	AgeLimit int     `json:"age_limit"`
	Poster   string  `json:"poster"`
	Genres   string  `json:"genres"`
	Score    int     `json:"score"`
	Average  float64 `json:"average"`
}

type Overview struct {
	User      Account `json:"user"`
	Favorites []Film  `json:"favorites"`
	Ratings   []Film  `json:"ratings"`
	Reviews   int     `json:"reviews"`
}

type Repo struct{ db *sql.DB }

func NewRepo(db *sql.DB) *Repo { return &Repo{db: db} }

func (r *Repo) Overview(ctx context.Context, id domain.UserID) (Overview, error) {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return Overview{}, err
	}
	defer tx.Rollback()

	var result Overview
	err = tx.QueryRowContext(ctx, `
		SELECT id, username, email, created_at,
		       (SELECT count(*) FROM review WHERE account_id = $1 AND deleted_at IS NULL)
		FROM account WHERE id = $1 AND deleted_at IS NULL`, id).Scan(
		&result.User.ID, &result.User.Username, &result.User.Email, &result.User.CreatedAt, &result.Reviews)
	if err != nil {
		return Overview{}, err
	}

	result.Favorites, err = readFilms(ctx, tx, id, `
		SELECT `+filmColumns+`, 0, `+averageRating+`
		FROM folder JOIN folder_item ON folder.id = folder_item.folder_id
		JOIN film f ON f.id = folder_item.film_id
		WHERE folder.account_id = $1 AND folder.folder_type = 'favorite'
		ORDER BY folder_item.created_at DESC, f.id`)
	if err != nil {
		return Overview{}, err
	}
	result.Ratings, err = readFilms(ctx, tx, id, `
		SELECT `+filmColumns+`, mine.score, `+averageRating+`
		FROM film_rating mine JOIN film f ON f.id = mine.film_id
		WHERE mine.account_id = $1
		ORDER BY mine.updated_at DESC, f.id`)
	if err != nil {
		return Overview{}, err
	}
	return result, tx.Commit()
}

const filmColumns = `f.id, f.title, f.production_year, f.film_type, f.age_limit,
	COALESCE((SELECT COALESCE(fl.external_url, '/static/' || fl.storage_key)
		FROM film_file ff JOIN file fl ON fl.id = ff.file_id
		WHERE ff.film_id = f.id AND ff.file_role = 'poster' ORDER BY ff.sort_order LIMIT 1), ''),
	COALESCE((SELECT string_agg(g.name, ', ' ORDER BY g.name)
		FROM film_genre fg JOIN genre g ON g.id = fg.genre_id WHERE fg.film_id = f.id), '')`

const averageRating = `COALESCE((SELECT ROUND(AVG(score), 1) FROM film_rating WHERE film_id = f.id), 0)::double precision`

func readFilms(ctx context.Context, tx *sql.Tx, id domain.UserID, query string) ([]Film, error) {
	rows, err := tx.QueryContext(ctx, query, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []Film{}
	for rows.Next() {
		var film Film
		if err := rows.Scan(&film.ID, &film.Title, &film.Year, &film.Type, &film.AgeLimit,
			&film.Poster, &film.Genres, &film.Score, &film.Average); err != nil {
			return nil, err
		}
		result = append(result, film)
	}
	return result, rows.Err()
}

func (r *Repo) UpdateAccount(ctx context.Context, id domain.UserID, name, email string) (Account, error) {
	var account Account
	err := r.db.QueryRowContext(ctx, `
		UPDATE account SET username = $2, email = $3 WHERE id = $1 AND deleted_at IS NULL
		RETURNING id, username, email, created_at`, id, name, email).
		Scan(&account.ID, &account.Username, &account.Email, &account.CreatedAt)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == storage.UniqueViolation {
		switch pgErr.ConstraintName {
		case "account_username_key":
			return Account{}, domain.ErrUsernameTaken
		case "account_email_key":
			return Account{}, domain.ErrUserExists
		}
	}
	return account, err
}

func (r *Repo) PasswordHash(ctx context.Context, id domain.UserID) (string, error) {
	var hash string
	err := r.db.QueryRowContext(ctx, "SELECT password_hash FROM account WHERE id = $1 AND deleted_at IS NULL", id).Scan(&hash)
	return hash, err
}

func (r *Repo) ChangePassword(ctx context.Context, id domain.UserID, oldHash, newHash string) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE account SET password_hash = $3 WHERE id = $1 AND password_hash = $2 AND deleted_at IS NULL`, id, oldHash, newHash)
	if err != nil {
		return err
	}
	count, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return ErrPasswordChanged
	}
	return nil
}

func (r *Repo) SetFavorite(ctx context.Context, id domain.UserID, filmID int64, favorite bool) error {
	if !favorite {
		_, err := r.db.ExecContext(ctx, `
			DELETE FROM folder_item item USING folder
			WHERE item.folder_id = folder.id AND folder.account_id = $1
			  AND folder.folder_type = 'favorite' AND item.film_id = $2`, id, filmID)
		return err
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var accountID int64
	if err := tx.QueryRowContext(ctx, "SELECT id FROM account WHERE id = $1 AND deleted_at IS NULL FOR UPDATE", id).Scan(&accountID); err != nil {
		return err
	}
	var folderID int64
	err = tx.QueryRowContext(ctx, "SELECT id FROM folder WHERE account_id = $1 AND folder_type = 'favorite'", id).Scan(&folderID)
	if errors.Is(err, sql.ErrNoRows) {
		err = tx.QueryRowContext(ctx, `
			INSERT INTO folder(account_id, title, folder_type)
			SELECT $1, candidate.title, 'favorite'
			FROM (SELECT CASE WHEN i = 0 THEN 'Избранное' ELSE 'Избранное (' || i || ')' END AS title, i
			      FROM generate_series(0, (SELECT count(*) FROM folder WHERE account_id = $1)) i) candidate
			WHERE NOT EXISTS (SELECT 1 FROM folder WHERE account_id = $1 AND title = candidate.title)
			ORDER BY candidate.i LIMIT 1 RETURNING id`, id).Scan(&folderID)
	}
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO folder_item(folder_id, film_id) VALUES ($1, $2)
		ON CONFLICT (folder_id, film_id) DO NOTHING`, folderID, filmID)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.ConstraintName == "folder_item_film_id_fkey" {
		return domain.ErrFilmNotFound
	}
	if err != nil {
		return err
	}
	return tx.Commit()
}
