package ratings

import (
	"context"
	"database/sql"
	"errors"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/domain"
)

func (r *PostgresRepo) GetMyScore(ctx context.Context, user domain.UserID, film domain.FilmID) (_ *int, operationErr error) {
	operationLog := r.operations.Start(ctx, "postgres", "rating.get_my_score", map[string]any{"user_id": user, "film_id": film})
	defer func() { operationLog.Finish(operationErr) }()

	var score int
	operationLog.Stage = "query"
	err := r.db.QueryRowContext(ctx, `SELECT score FROM film_rating WHERE account_id = $1 AND film_id = $2`, user, film).Scan(&score)
	if errors.Is(err, sql.ErrNoRows) {
		operationLog.NotFound = true
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &score, nil
}
