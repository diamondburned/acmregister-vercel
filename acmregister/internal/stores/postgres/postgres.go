package postgres

import (
	"context"
	"database/sql"
	"strings"

	_ "embed"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pkg/errors"
)

//go:embed schema.sql
var schemaSQL string

// Versions returns the versions of the schema.
func Versions() []string {
	return strings.Split(schemaSQL, "-- NEW VERSION\n")
}

const codeTableNotFound = "42P01"

// Migrate migrates the given database to the latest migrations. It uses the
// user_version pragma.
func Migrate(ctx context.Context, db *sql.DB) error {
	var firstRun bool

	v, err := New(db).Version(ctx)
	if err != nil {
		if !IsErrorCode(err, codeTableNotFound) {
			return errors.Wrap(err, "cannot get schema version")
		}
		firstRun = true
	}

	versions := Versions()
	if int(v) >= len(versions) {
		return nil
	}

	tx, err := db.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelSerializable,
	})
	if err != nil {
		return errors.Wrap(err, "cannot begin transaction")
	}
	defer tx.Rollback()

	if !firstRun {
		v, err = New(tx).Version(ctx)
		if err != nil {
			return errors.Wrap(err, "cannot get schema version")
		}
	}

	for i := int(v); i < len(versions); i++ {
		_, err := tx.ExecContext(ctx, versions[i])
		if err != nil {
			return errors.Wrapf(err, "cannot apply migration %d (from 0th)", i)
		}
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrap(err, "cannot commit new migrations")
	}

	return nil
}

// IsConstraintFailed returns true if err is returned because of a unique
// constraint violation.
func IsConstraintFailed(err error) bool {
	return IsErrorCode(err, "23505") // unique_violation
}

func IsErrorCode(err error, code string) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}
	// https://www.postgresql.org/docs/current/errcodes-appendix.html
	return pgErr.Code == code
}
