package repo

import (
	"crm-middleware/crypt"
	"crm-middleware/db"
	"crm-middleware/model"
	"errors"
	"fmt"
	"time"
)

var Users users

type users struct{}

func (users) List(clientID string) ([]*model.User, error) {
	rows, err := db.Query(`
		SELECT id, username FROM users WHERE client_id = ? ORDER BY username
	`, clientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*model.User
	for rows.Next() {
		var u model.User
		if err := rows.Scan(&u.ID, &u.Username); err != nil {
			return nil, err
		}
		users = append(users, &u)
	}

	return users, rows.Err()
}

func (users) Create(id string, clientID string, username string, password string) (*model.User, error) {
	var u = &model.User{
		ID:          id,
		ClientID:    clientID,
		Username:    username,
		TimeUpdated: time.Now(),
	}

	if u.Username == "" {
		u.Username = "-"
	}

	if len(password) > 0 {
		var password_enc crypt.CipherText
		password_enc.Encrypt(password)
		u.Password = string(password_enc)
	}

	_, err := db.Exec(`
		INSERT INTO users (id, client_id, username, password_enc, time_updated) VALUES (?, ?, ?, ?, ?)
	`, u.ID, u.ClientID, u.Username, u.Password, u.TimeUpdated)

	if err != nil {
		return nil, err
	}

	u.Password = password
	return u, nil
}

func (users) Get(id string, clientID string) (*model.User, error) {
	var user = new(model.User)
	err := db.QueryRow(`SELECT id, client_id, username, password_enc, time_updated FROM users WHERE id = ? AND client_id = ?`, id, clientID).Scan(
		&user.ID, &user.ClientID, &user.Username, &user.Password, &user.TimeUpdated,
	)
	if err != nil {
		return nil, err
	}

	if len(user.Password) > 0 {
		user.Password = crypt.CipherText(user.Password).Decrypt()
	}
	return user, nil
}

func (users) Delete(id string, clientID string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	result, err := tx.Exec(`DELETE FROM users WHERE id = ? AND client_id = ?`, id, clientID)
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check affected rows: %w", err)
	}
	if affected == 0 {
		return errors.New("user not found")
	}

	if _, err := tx.Exec(`DELETE FROM tokens WHERE user_id = ? AND client_id = ?`, id, clientID); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}
