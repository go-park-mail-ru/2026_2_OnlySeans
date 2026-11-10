package auth

import (
	"database/sql"
	"github.com/go-park-mail-ru/2026_2_OnlySeans/pkg/logger"
)

type ProfileRepo struct {
	db  *sql.DB
	log logger.Logger
}

func NewProfileRepo(db *sql.DB, appLogger logger.Logger) *ProfileRepo {
	return &ProfileRepo{log: appLogger, db: db}
}
