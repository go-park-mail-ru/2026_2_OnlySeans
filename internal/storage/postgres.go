package storage

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/go-park-mail-ru/2026_2_OnlySeans/internal/config"
)

func Open(ctx context.Context, cfg config.Database) (_ *sql.DB, err error) {
	db, err := sql.Open("pgx", cfg.URL) //открыть пул
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// при ошибке закрываем пул, чтобы соединения не утекли
	defer func() {
		if err != nil {
			db.Close()
		}
	}()

	//настраиваем пул
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	//ограничиваем время подключения
	pingCtx, cancel := context.WithTimeout(ctx, cfg.ConnectTimeout)
	defer cancel()

	//проверяем, что база отвечает
	if err := db.PingContext(pingCtx); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return db, nil
}
