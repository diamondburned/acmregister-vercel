# SQL Tricks

## SQLite Dependency

This app makes use of a pure Go SQLite library. As such, the user does not need
to install any C libraries to compile and/or run the binary.

## Migration

This app uses migrations to manage the SQLite schema. Once stable, newer commits
that modify the schema must use migrations to do so. The SQLite schema can be
found in [sqlite/schema.sql](./sqlite/schema.sql).

Example:

```sql
-- NEW VERSION
-- This is the base schema.
CREATE TABLE a (
	field_a TEXT NOT NULL
);

-- NEW VERSION
-- We add a new field into table a.
ALTER TABLE a ADD field_b TEXT NOT NULL;
```

As seen above, each migration version is delimited by the comment `-- NEW
VERSION` on its own line.

This will make use of the `user_version` pragma to determine which version the
current database instance is on and make the right migrations.

Make sure to run `go generate ./...` to regenerate the Go code using
[sqlc](https://github.com/kyleconroy/sqlc).

## Queries

SQL queries inside [sqlite/queries.sql](./sqlite/queries.sql) are
converted using Go code using the same command above.

Make sure to update [sqlite.go](./sqlite.go) after regenerating the code with a
modified SQL file.
