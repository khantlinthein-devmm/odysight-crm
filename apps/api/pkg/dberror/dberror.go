package dberror

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/odysight/crm/pkg/response"
)

// Map converts Postgres constraint errors to user-facing API errors.
// Falls back to the original error (→ 500) for anything else.
func Map(err error, notFound error, notFoundMsg string) error {
	if errors.Is(err, notFound) {
		return response.NewAPIError(404, notFoundMsg)
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return response.NewAPIError(409, "resource already exists")
		case "23503":
			return response.NewAPIError(400, "referenced record does not exist")
		case "23514", "23502", "22P02", "22001":
			return response.NewAPIError(400, "invalid data: "+pgErr.Message)
		}
	}
	return err
}
