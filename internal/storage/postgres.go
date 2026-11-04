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
		appLogger.Error("open database failed", map[string]any{
			"error": err,
		})
		return nil, fmt.Errorf("open database: %w", err)
	}

	// При ошибке закрываем пул, чтобы соединения не утекли.
	defer func() {
		if err != nil {
			if closeErr := db.Close(); closeErr != nil {
				appLogger.Error("close database after initialization failure", map[string]any{
					"error": closeErr,
				})
			}
		}
	}()

	// Настраиваем пул.
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	// Ограничиваем время подключения.
	pingCtx, cancel := context.WithTimeout(ctx, cfg.ConnectTimeout)
	defer cancel()

	// Проверяем, что база отвечает.
	if err := db.PingContext(pingCtx); err != nil {
		appLogger.Error("database ping failed", map[string]any{
			"error": err,
		})
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return db, nil
}
