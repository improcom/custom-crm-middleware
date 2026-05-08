package model

import "time"

type DataSchema struct {
	ClientID   string
	ObjectType string
	Fields     []*DataSchemaField
}

type DataSchemaField struct {
	ID          string
	Type        string
	Logic       string
	Custom      bool
	TimeUpdated time.Time
}
