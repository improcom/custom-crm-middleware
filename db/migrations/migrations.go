package migrations

import (
	"crm-middleware/db"
	"crm-middleware/db/settings"
	"fmt"
	"log/slog"
)

const latestVersion = 2

var enabledMigrations = [latestVersion][]*db.Statement{
	0: {
		createClients,
		createDataSchemas,
		createUsers,
		createTokens,
	},

	1: {
		insertInstanceID,
		insertJWTSecret,
	},
}

var (
	migrationVersion       = settings.Key("mig", "version")
	migrationVersionOffset = settings.Key("mig", "version-offset")
)

func Run() error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS settings (
			key TEXT NOT NULL,
			value TEXT NOT NULL,
			at DATETIME DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (key)
		)
	`)
	if err != nil {
		return fmt.Errorf("create settings table: %w", err)
	}

	version := settings.Get[int](migrationVersion)
	offset := settings.Get[int](migrationVersionOffset)

	if version > latestVersion {
		return fmt.Errorf("migration version %04d is ahead of latest supported version %04d", version, latestVersion)
	}

	if version == latestVersion {
		slog.Debug("migrations up to date", "version", version)
	}

	for ver := version; ver < latestVersion; ver++ {
		slog.Info("applying migration", "version", ver)
		for off := offset; off < len(enabledMigrations[ver]); off++ {

			statement := enabledMigrations[ver][off]
			if _, err := db.Exec(statement.Query, statement.Args...); err != nil {
				return fmt.Errorf("migration %04d statement %d: %w: %+v", ver, off, err, statement)
			}

			if err := settings.Set(migrationVersionOffset, off+1); err != nil {
				return fmt.Errorf("save migration offset: %w", err)
			}
		}

		if err := settings.Set(migrationVersion, ver+1); err != nil {
			return fmt.Errorf("save new migration version: %d: %w", ver, err)
		}
		if err := settings.Set(migrationVersionOffset, 0); err != nil {
			return fmt.Errorf("reset migration offset on version: %d: %w", ver, err)
		}
	}

	return nil
}
