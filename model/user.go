package model

import (
	"time"
)

type User struct {
	ID          string
	ClientID    string
	Username    string
	Password    string
	TimeUpdated time.Time
}
