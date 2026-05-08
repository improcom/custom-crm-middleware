package api

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
)

const CallbackEndpoint = "/callback/code"

type AuthMethod string

const (
	AuthMethodOAuth2    AuthMethod = "oauth2"
	AuthMethodBasicAuth AuthMethod = "basic-auth"
	AuthMethodNone      AuthMethod = "none"
)

type AuthStatus struct {
	Authorized bool   `json:"authorized"`
	Email      string `json:"email"`
}

type AuthHandlers struct {
	Callback http.HandlerFunc
	Status   http.HandlerFunc
	Login    http.HandlerFunc
	Logout   http.HandlerFunc
}

type ObjectHandlers struct {
	Create http.HandlerFunc
	Get    http.HandlerFunc
	Update http.HandlerFunc
}

type SearchHandlers struct {
	All          http.HandlerFunc
	Associations http.HandlerFunc
}

type TaskHandlers struct {
	Create http.HandlerFunc
}

type IntegrationConfig struct {
	Name       string     `json:"name"`
	AuthMethod AuthMethod `json:"auth_method"`
	ImageData  string     `json:"image"`
}

type ObjectInfo struct {
	ID                    string   `json:"id"`
	Type                  string   `json:"type"`
	DisplayName           string   `json:"title"`
	Emails                []string `json:"emails,omitempty"`
	Phones                []string `json:"phones,omitempty"`
	LastModifiedTimestamp int64    `json:"last_modified_timestamp"`
}

type SearchParameters struct {
	Query             string
	By                string
	Limit             string
	Field             string
	ObjectTypes       []string
	ReturningFields   [][]string
	AssociationFields struct {
		IDField    string
		LabelField string
	}
}

type State struct {
	OriginalState string `json:"state,omitempty"`
	ClientID      string `json:"client_id,omitempty"`
	UserID        string `json:"user_id,omitempty"`
	RedirectURI   string `json:"redirect_uri"`
}

func (s State) Encode() (string, error) {
	enc, err := json.Marshal(s)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.WithPadding(base64.NoPadding).EncodeToString(enc), nil
}

func (s *State) Decode(state string) {
	rawState, _ := base64.StdEncoding.WithPadding(base64.NoPadding).DecodeString(state)
	json.Unmarshal(rawState, &s)
}

func (p SearchParameters) SlogAttr() slog.Attr {
	var objtyp = make([]any, 0)
	for i, typ := range p.ObjectTypes {
		objtyp = append(objtyp, slog.String(fmt.Sprintf("obj-%d", i), typ))
	}

	return slog.Group("search-parameters",
		slog.String("query", p.Query),
		slog.String("search-by", p.By),
		slog.String("limit", p.Limit),
		slog.String("field", p.Field),
		slog.Group("object-types", objtyp...),
	)
}

func GetSearchParameters(r *http.Request) (*SearchParameters, error) {
	v := r.URL.Query()
	p := &SearchParameters{
		Query:           v.Get("q"),
		By:              v.Get("search_by"),
		Limit:           v.Get("limit"),
		Field:           v.Get("field"),
		ObjectTypes:     v["object_types"],
		ReturningFields: make([][]string, 0),
	}

	if p.Query == "" {
		return nil, errors.New("missing query")
	}

	if len(p.ObjectTypes) == 0 {
		p.ObjectTypes = v["object_type"]
	}

	for range p.ObjectTypes {
		p.ReturningFields = append(p.ReturningFields, make([]string, 0))
	}

	if assoc := v.Get("fields"); assoc != "" {
		if fields := strings.Split(assoc, ","); len(fields) == 2 && fields[0] != "" && fields[1] != "" {
			p.AssociationFields.IDField = fields[0]
			p.AssociationFields.LabelField = fields[1]
		}
	}

	if p.By == "field" {
		if p.Field == "" {
			return nil, errors.New("search by field: missing target field")
		}
		if len(p.ObjectTypes) == 0 {
			return nil, errors.New("search by field: missing object type")
		}
		if len(p.ObjectTypes) != 1 {
			return nil, errors.New("search by field: multiple object types not allowed")
		}
	}

	if limit, _ := strconv.Atoi(p.Limit); limit < 1 || limit > 10 {
		p.Limit = "10"
	}

	return p, nil
}
