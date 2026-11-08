package ratings

import (
	"context"
	"database/sql"
	"errors"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
)

func (r *PostgresRepo) GetMyScore(ctx context.Context, user domain.UserID, film domain.FilmID) (*int, error) {
	var score int
	err := r.db.QueryRowContext(ctx, `SELECT score FROM film_rating WHERE account_id = $1 AND film_id = $2`, user, film).Scan(&score)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &score, nil
}
