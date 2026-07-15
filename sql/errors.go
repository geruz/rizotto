package sql

import (
	"database/sql"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	duplicateKeyErrorCode = "23505"
)

func isErrorCode(err error, errCode string) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == errCode
	}

	return false
}

func IsDuplicateError(err error) bool {
	return err != nil && isErrorCode(err, duplicateKeyErrorCode)
}

func IsNotFoundError(err error) bool {
	return err != nil && (errors.Is(err, pgx.ErrNoRows) || errors.Is(err, sql.ErrNoRows))
}
