package migrations

import (
	"crm-middleware/db"
)

var (
	createClients = db.Stmt(`
		CREATE TABLE clients (
			id TEXT,
			secret TEXT NOT NULL,
			name TEXT UNIQUE NOT NULL,
			app TEXT NOT NULL,
			status TEXT NOT NULL,
			time_updated DATETIME DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (id)
		)
	`)

	createUsers = db.Stmt(`
		CREATE TABLE users (
			id TEXT NOT NULL,
			client_id TEXT NOT NULL,
			username TEXT NOT NULL,
			password_enc TEXT NOT NULL DEFAULT '',
			time_updated DATETIME DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (id, client_id)
		)
	`)

	createDataSchemas = db.Stmt(`
		CREATE TABLE data_schemas (
			client_id TEXT NOT NULL,
			object_type TEXT NOT NULL,
			field_id TEXT NOT NULL,
			type TEXT NOT NULL,
			logic TEXT NOT NULL,
			custom BOOL DEFAULT FALSE,
			time_updated DATETIME DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (client_id, object_type, field_id)
		)
	`)

	createTokens = db.Stmt(`
		CREATE TABLE tokens (
			user_id TEXT NOT NULL,
			client_id TEXT NOT NULL,
			type TEXT NOT NULL,
			token TEXT NOT NULL,
			refresh TEXT NOT NULL,
			expiry DATETIME NOT NULL,
			attrs TEXT,
			time_updated DATETIME DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (user_id, client_id)
		)
	`)
)
