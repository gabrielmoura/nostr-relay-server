package db

import (
	"database/sql"
	"errors"

	"github.com/jackc/pgx/v5"
)

func isNotFound(err error) bool {
	return errors.Is(err, sql.ErrNoRows) || errors.Is(err, pgx.ErrNoRows)
}
