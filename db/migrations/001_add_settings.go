package migrations

import (
	"crm-middleware/crypt"
	"crm-middleware/db/settings"
)

var (
	insertInstanceID = settings.Stmt(settings.Key("sys", "instance-id"), crypt.Random(32))

	insertJWTSecret = settings.Stmt(settings.Key("sys", "jwt-secret"), crypt.Random(64))
)
