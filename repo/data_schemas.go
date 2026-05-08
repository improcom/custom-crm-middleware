package repo

import (
	"crm-middleware/db"
	"crm-middleware/model"
	"fmt"
	"strings"
	"time"
)

const DataSchemaExcludeCustom = true

var DataSchemas dataSchemas

type dataSchemas struct{}

func (dataSchemas) Load(objectType, clientID string, excludeCustom ...bool) (*model.DataSchema, error) {
	query := `
		SELECT field_id, type, logic, custom, time_updated FROM data_schemas WHERE object_type = ? AND client_id = ?
	`
	if len(excludeCustom) > 0 && excludeCustom[0] {
		query += ` AND custom = FALSE`
	}

	rows, err := db.Query(query, objectType, clientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	schema := &model.DataSchema{
		ObjectType: objectType,
		ClientID:   clientID,
		Fields:     make([]*model.DataSchemaField, 0),
	}
	for rows.Next() {
		var f model.DataSchemaField
		if err := rows.Scan(&f.ID, &f.Type, &f.Logic, &f.Custom, &f.TimeUpdated); err != nil {
			return nil, err
		}
		schema.Fields = append(schema.Fields, &f)
	}

	return schema, rows.Err()
}

func (dataSchemas) Save(clientID string, schemas ...*model.DataSchema) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err = tx.Exec(`DELETE FROM data_schemas WHERE client_id = ?`, clientID); err != nil {
		return err
	}

	for _, s := range schemas {
		if len(s.Fields) == 0 {
			continue
		}

		args := make([]any, 0, len(s.Fields)*7)
		placeholders := make([]string, 0, len(s.Fields))

		for _, f := range s.Fields {
			args = append(args, s.ObjectType, clientID, f.ID, f.Type, f.Logic, f.Custom, time.Now())
			placeholders = append(placeholders, "(?, ?, ?, ?, ?, ?, ?)")
		}

		if _, err = tx.Exec(fmt.Sprintf(`
			INSERT INTO data_schemas (object_type, client_id, field_id, type, logic, custom, time_updated) VALUES %s
		`, strings.Join(placeholders, ",")), args...); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}
