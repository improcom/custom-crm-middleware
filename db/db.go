package db

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

var db *sql.DB

func Init(path string) (err error) {
	db, err = sql.Open("sqlite", path)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)

	for _, pragma := range []string{
		`PRAGMA journal_mode=WAL`, `PRAGMA busy_timeout=5000`, `PRAGMA synchronous=NORMAL`, `PRAGMA foreign_keys=ON`,
	} {
		if _, err = db.Exec(pragma); err != nil {
			return fmt.Errorf("apply pragma %q: %w", pragma, err)
		}
	}

	return nil
}

func Close() {
	db.Close()
}

func Exec(query string, args ...any) (sql.Result, error) {
	return db.Exec(query, args...)
}

func Query(query string, args ...any) (*sql.Rows, error) {
	return db.Query(query, args...)
}

func QueryRow(query string, args ...any) *sql.Row {
	return db.QueryRow(query, args...)
}

func Begin() (*sql.Tx, error) {
	return db.Begin()
}

type Statement struct {
	Query string
	Args  []any
}

func Stmt(q string, args ...any) *Statement {
	return &Statement{q, args}
}

