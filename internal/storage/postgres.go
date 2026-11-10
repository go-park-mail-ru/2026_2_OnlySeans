package storage

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/config"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
)

func Open(
	ctx context.Context,
	cfg config.Database,
	appLogger logger.Logger,
) (_ *sql.DB, err error) {
	db, err := sql.Open("pgx", cfg.URL)
	if err != nil {
		appLogger.ErrorContext(ctx, "open database failed", map[string]any{
			"error": err,
		})
		return nil, fmt.Errorf("open database: %w", err)
	}

	defer func() {
		if err != nil {
			if closeErr := db.Close(); closeErr != nil {
				appLogger.ErrorContext(ctx, "close database after initialization failure", map[string]any{
					"error": closeErr,
				})
			}
		}
	}()

	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	pingCtx, cancel := context.WithTimeout(ctx, cfg.ConnectTimeout)
	defer cancel()

	if err := db.PingContext(pingCtx); err != nil {
		appLogger.ErrorContext(ctx, "database ping failed", map[string]any{
			"error": err,
		})
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return db, nil
}
