package settings

import (
	"crm-middleware/db"
	"strings"
	"time"
)

type key struct{ string }

func (k key) String() string {
	return k.string
}

func Key(segments ...string) (k key) {
	k.string = strings.Join(segments, ":")
	return
}

func Get[T any](k key) (v T) {
	db.QueryRow(`SELECT value FROM settings WHERE key = ?`, strings.ToUpper(k.string)).Scan(&v)
	return
}

func Set[T any](k key, v T) (err error) {
	stmt := Stmt(k, v)
	_, err = db.Exec(stmt.Query, stmt.Args...)
	return err
}

func Stmt[T any](k key, v T) *db.Statement {
	return &db.Statement{
		Query: `
			INSERT INTO settings (key, value, at) VALUES (?, ?, ?)
			ON CONFLICT(key) DO UPDATE SET value = excluded.value, at = excluded.at
		`,
		Args: []any{strings.ToUpper(k.String()), v, time.Now()},
	}
}
