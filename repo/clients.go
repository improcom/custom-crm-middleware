package repo

import (
	"crm-middleware/build"
	"crm-middleware/crypt"
	"crm-middleware/db"
	"crm-middleware/model"
	"errors"
	"fmt"
	"time"
)

var (
	ErrClientNotFound = errors.New("client not found")
)

var Clients clients

type clients struct{}

func (clients) List() (clients []*model.Client, err error) {
	rows, err := db.Query(`SELECT id, secret, name, app, status, time_updated FROM clients ORDER BY time_updated`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var c model.Client
		if err := rows.Scan(&c.ID, &c.Secret, &c.Name, &c.App, &c.Status, &c.TimeUpdated); err != nil {
			return nil, err
		}
		clients = append(clients, &c)
	}

	return clients, rows.Err()
}

func (clients) Create(name string, status model.ClientStatus) (*model.Client, error) {
	var client = &model.Client{
		ID:     crypt.Random(8),
		Secret: crypt.Random(32, build.Token),
		Name:   name,
		Status: status,
	}
	_, err := db.Exec(`
		INSERT INTO clients (id, secret, name, app, status, time_updated) VALUES (?, ?, ?, ?, ?, ?)
	`, client.ID, client.Secret, client.Name, client.App, client.Status, time.Now())

	if err != nil {
		return nil, err
	}
	return client, nil
}

func (clients) Get(id string) (*model.Client, error) {
	var client = new(model.Client)
	err := db.QueryRow(`
		SELECT id, secret, name, app, status, time_updated FROM clients WHERE id = ?
	`, id).
		Scan(&client.ID, &client.Secret, &client.Name, &client.App, &client.Status, &client.TimeUpdated)

	if err != nil {
		return nil, err
	}
	return client, nil
}

func (clients) UpdateStatus(id string, appName string, status model.ClientStatus) error {
	result, err := db.Exec(`UPDATE clients SET app = ?, status = ?, time_updated = ? WHERE id = ?`, appName, status, time.Now(), id)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check affected rows: %w", err)
	}
	if affected == 0 {
		return ErrClientNotFound
	}
	return nil
}

func (clients) DeleteClientData(id string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	result, err := tx.Exec(`UPDATE clients SET status = ?, time_updated = ? WHERE id = ?`, model.ClientStatusDeleted, time.Now(), id)
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check affected rows: %w", err)
	}
	if affected == 0 {
		return ErrClientNotFound
	}

	if _, err := tx.Exec(`DELETE FROM data_schemas WHERE client_id = ?`, id); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM users WHERE client_id = ?`, id); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM tokens WHERE client_id = ?`, id); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

func (clients) ResetSecret(id string) error {
	result, err := db.Exec(`
		UPDATE clients SET secret = ?, status = ?, time_updated = ? WHERE id = ?
	`, crypt.Random(32, build.Token), model.ClientStatusRevoked, time.Now(), id)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check affected rows: %w", err)
	}
	if affected == 0 {
		return ErrClientNotFound
	}
	return nil
}
