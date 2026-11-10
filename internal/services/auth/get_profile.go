package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/storage"
)

const (
	getProfileUserQuery = `
        SELECT a.id, a.email, a.username, a.created_at,
            (SELECT storage_key FROM file WHERE id = a.avatar_file_id),
            (SELECT count(*) FROM review WHERE account_id = a.id AND deleted_at IS NULL)
        FROM account AS a WHERE a.id = $1 AND a.deleted_at IS NULL`

	profileFilmColumns = `
        f.id, f.title, f.production_year, f.film_type, f.age_limit,
        COALESCE((SELECT string_agg(g.name, ', ' ORDER BY g.name)
            FROM film_genre AS fg JOIN genre AS g ON g.id = fg.genre_id WHERE fg.film_id = f.id), ''),
        (SELECT fl.external_url FROM film_file AS ff JOIN file AS fl ON fl.id = ff.file_id
            WHERE ff.film_id = f.id AND ff.file_role = 'poster' ORDER BY ff.sort_order LIMIT 1),
        (SELECT CASE WHEN fl.external_url IS NULL THEN fl.storage_key END
            FROM film_file AS ff JOIN file AS fl ON fl.id = ff.file_id
            WHERE ff.film_id = f.id AND ff.file_role = 'poster' ORDER BY ff.sort_order LIMIT 1),
        COALESCE((SELECT avg(score) FROM film_rating WHERE film_id = f.id), 0)`

	getProfileFavoritesQuery = `SELECT ` + profileFilmColumns + `
        FROM folder AS folder
        JOIN folder_item AS item ON item.folder_id = folder.id
        JOIN film AS f ON f.id = item.film_id
        WHERE folder.account_id = $1 AND folder.folder_type = 'favorite'
        ORDER BY item.created_at DESC, f.id`

	getProfileRatingsQuery = `SELECT ` + profileFilmColumns + `, rating.score
        FROM film_rating AS rating JOIN film AS f ON f.id = rating.film_id
        WHERE rating.account_id = $1
        ORDER BY rating.updated_at DESC, f.id`
)

func (r *ProfileRepo) GetProfile(ctx context.Context, id domain.UserID) (_ *domain.Profile, operationErr error) {
	operation := storage.StartOperation(ctx, r.log, "postgres", "profile.get_profile", map[string]any{"user_id": id})
	defer func() { operation.Finish(operationErr) }()
	operation.Stage = "begin"
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return nil, fmt.Errorf("begin profile read: %w", err)
	}
	defer tx.Rollback()
	profile := &domain.Profile{}
	operation.Stage = "query"
	err = tx.QueryRowContext(ctx, getProfileUserQuery, id).Scan(&profile.User.ID, &profile.User.Email, &profile.User.Username, &profile.User.CreatedAt, &profile.User.AvatarFilename, &profile.Reviews)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("read profile user: %w", err)
	}
	profile.Favorites, err = readProfileFilms(ctx, tx, getProfileFavoritesQuery, id, false)
	if err != nil {
		return nil, fmt.Errorf("read profile favorites: %w", err)
	}
	profile.Ratings, err = readProfileFilms(ctx, tx, getProfileRatingsQuery, id, true)
	if err != nil {
		return nil, fmt.Errorf("read profile ratings: %w", err)
	}
	operation.Stage = "commit"
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit profile read: %w", err)
	}
	return profile, nil
}

func readProfileFilms(ctx context.Context, tx *sql.Tx, query string, id domain.UserID, withScore bool) ([]domain.ProfileFilm, error) {
	rows, err := tx.QueryContext(ctx, query, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	films := make([]domain.ProfileFilm, 0)
	for rows.Next() {
		var film domain.ProfileFilm
		values := []any{&film.ID, &film.Title, &film.Year, &film.Type, &film.AgeLimit, &film.Genres, &film.PosterURL, &film.PosterStorageKey, &film.Average}
		var score int
		if withScore {
			values = append(values, &score)
		}
		if err := rows.Scan(values...); err != nil {
			return nil, err
		}
		if withScore {
			film.Score = &score
		}
		films = append(films, film)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return films, nil
}
