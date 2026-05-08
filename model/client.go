package model

import "time"

type ClientStatus string

const (
	ClientStatusNone    ClientStatus = ""
	ClientStatusActive  ClientStatus = "ACTIVE"
	ClientStatusDeleted ClientStatus = "DELETED"
	ClientStatusRevoked ClientStatus = "REVOKED"
	ClientStatusRouting ClientStatus = "ROUTING"
)

type Client struct {
	ID          string
	Secret      string
	Name        string
	App         string
	Status      ClientStatus
	TimeUpdated time.Time
}
