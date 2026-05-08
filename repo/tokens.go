package repo

import (
	"crm-middleware/db"
	"crm-middleware/model"
	"time"
)

var Tokens tokens

type tokens struct{}

func (tokens) Get(userID string, clientID string) (*model.Token, error) {
	var t = new(model.Token)
	err := db.QueryRow(`
		SELECT type, token, refresh, expiry, attrs, time_updated
		FROM tokens WHERE user_id = ? AND client_id = ?
	`, userID, clientID).Scan(
		&t.Type, &t.AccessToken, &t.RefreshToken, &t.Expiry, &t.Attrs, &t.TimeUpdated,
	)
	if err != nil {
		return nil, err
	}

	t.UserID = userID
	t.ClientID = clientID
	return t, nil
}

func (tokens) Save(userID string, clientID string, token *model.Token) error {
	_, err := db.Exec(`
		INSERT OR REPLACE
		INTO tokens (user_id, client_id, type, token, refresh, expiry, attrs, time_updated)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, userID, clientID, token.Type, token.AccessToken, token.RefreshToken, token.Expiry, token.Attrs, time.Now())

	return err
}
