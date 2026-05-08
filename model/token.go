package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

type Attributes struct {
	data map[string]string
}

func (attr *Attributes) Get(key string) string {
	v, _ := attr.data[key]
	return v
}

func (attr *Attributes) Set(key string, values ...any) string {
	if attr.data == nil {
		attr.data = make(map[string]string)
	}
	for _, v := range values {
		if strval := fmt.Sprint(v); v != nil && strval != "" {
			attr.data[key] = strval
			break
		}
	}
	return attr.Get(key)
}

func (attr *Attributes) Coalesce(keys ...string) (val string) {
	for _, key := range keys {
		if val = attr.Get(key); val != "" {
			break
		}
	}
	return
}

func (a Attributes) Value() (driver.Value, error) {
	if a.data == nil {
		a.data = make(map[string]string)
	}
	b, err := json.Marshal(a.data)
	return string(b), err
}

func (a *Attributes) Scan(src any) error {
	var raw string
	switch v := src.(type) {
	case string:
		raw = v
	case []byte:
		raw = string(v)
	case nil:
		return nil
	default:
		return fmt.Errorf("attrs: cannot scan %T", src)
	}
	return json.Unmarshal([]byte(raw), &a.data)
}

type Token struct {
	UserID       string
	ClientID     string
	Type         string
	AccessToken  string
	RefreshToken string
	Expiry       time.Time
	TimeUpdated  time.Time

	Attrs *Attributes
}

type TokenGetter interface {
	Token(userID string, clientID string) (*Token, error)
}

func TokenGetterConfig[T any](getter TokenGetter) (*T, error) {
	config, ok := any(getter).(*T)
	if !ok {
		return nil, fmt.Errorf("token getter is not %T configuration", (*T)(nil))
	}
	return config, nil
}
