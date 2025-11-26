package database

import (
	"errors"

	"github.com/uptrace/bun/driver/pgdriver"
)

// PostgreSQL SQLSTATE error codes.
// See: https://www.postgresql.org/docs/current/errcodes-appendix.html
const (
	PgUniqueViolation     = "23505"
	PgForeignKeyViolation = "23503"
	PgNotNullViolation    = "23502"
)

// PostgreSQL protocol error field identifiers used with pgdriver.Error.Field().
// See: https://www.postgresql.org/docs/current/protocol-error-fields.html
//
// Common fields:
//   - 'C' - SQLSTATE code (e.g., "23505")
//   - 'M' - Primary error message
//   - 'D' - Detail message
//   - 'H' - Hint
//   - 's' - Schema name
//   - 't' - Table name
//   - 'c' - Column name
//   - 'n' - Constraint name
const (
	pgFieldCode       = 'C'
	pgFieldConstraint = 'n'
)

// IsUniqueViolation checks if the error is a PostgreSQL unique constraint violation.
func IsUniqueViolation(err error) bool {
	return hasPgCode(err, PgUniqueViolation)
}

// IsForeignKeyViolation checks if the error is a PostgreSQL foreign key violation.
func IsForeignKeyViolation(err error) bool {
	return hasPgCode(err, PgForeignKeyViolation)
}

// IsNotNullViolation checks if the error is a PostgreSQL not-null constraint violation.
func IsNotNullViolation(err error) bool {
	return hasPgCode(err, PgNotNullViolation)
}

// ConstraintName returns the constraint name from a PostgreSQL error, or empty string if not applicable.
func ConstraintName(err error) string {
	var pgErr pgdriver.Error
	if errors.As(err, &pgErr) {
		return pgErr.Field(pgFieldConstraint)
	}
	return ""
}

func hasPgCode(err error, code string) bool {
	var pgErr pgdriver.Error
	return errors.As(err, &pgErr) && pgErr.Field(pgFieldCode) == code
}
