// Package custom is a skeleton CRM integration. It registers with PBXware
// under the name "Custom" with no user authentication, so the integration
// can be added and activated without any external CRM configured.
//
// Every handler below marked with TODO is a stub: replace its body with
// calls to the target CRM. See apps/salesforce for a complete reference
// implementation and docs/API-docs.md for the endpoint contracts.
package custom

import (
	"crm-middleware/api"
	"crm-middleware/httputil"
	"crm-middleware/model/dataschema"
	"embed"
	"encoding/base64"
	"encoding/json"
	"net/http"
)

const (
	Version = "1.0"

	Contact = "Contact"
)

var (
	// PBXware requires the integration icon to be a 240x240 .webp
	// (see docs/API-docs.md, Config endpoint).
	//go:embed icons/custom.webp
	icon []byte

	//go:embed schemas
	schemaFiles        embed.FS
	objectDescriptions = map[string]*dataschema.Model{
		Contact: dataschema.MustLoad(schemaFiles, "schemas/"+Contact+".json"),
	}
)

type custom struct {
	*api.AppV1
}

func App() (c *custom) {
	c = &custom{AppV1: &api.AppV1{
		Path:     "/custom",
		Versions: []string{Version},
		Objects:  objectDescriptions,

		Config: api.IntegrationConfig{
			Name:       "Custom",
			AuthMethod: api.AuthMethodBasicAuth,
			ImageData:  base64.StdEncoding.EncodeToString(icon),
		},
	}}

	// AuthMethodBasicAuth: PBXware shows a username/password login inside
	// the integration and POSTs it to LoginHandler below. The docs also
	// list auth_method "none", but the PBXware build this was tested
	// against (7.x) rejects "none" during custom
	// integration activation, so basic-auth is the simplest working choice.
	c.AuthHandlers = api.AuthHandlers{
		Login:  c.LoginHandler,
		Status: api.AuthorizedClientUser,
		Logout: api.DeleteClientUser,
	}

	c.ObjectHandlers = api.ObjectHandlers{
		Create: c.ObjectCreateHandler,
		Get:    c.ObjectGetHandler,
		Update: c.ObjectUpdateHandler,
	}

	c.SearchHandlers = api.SearchHandlers{
		All:          c.SearchHandler,
		Associations: c.SearchHandler,
	}

	c.TaskHandlers = api.TaskHandlers{
		Create: c.CreateTaskHandler,
	}

	return c
}

// LoginHandler receives the username/password a user enters in PBXware
// (POST /api/v1/auth/login) and reports whether they are valid.
// TODO: validate the credentials against the target CRM and persist the
// user's session; this skeleton accepts ANY credentials.
func (c *custom) LoginHandler(w http.ResponseWriter, r *http.Request) {
	var credentials struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&credentials); err != nil {
		httputil.JSONResponse(w, map[string]string{
			"status":  "error",
			"message": "invalid request body",
		}, http.StatusBadRequest)
		return
	}

	httputil.JSONResponse(w, map[string]string{
		"status":  "ok",
		"message": "authorized",
	})
}

// SearchHandler answers PBXware record searches (contact popup, manual
// lookup). TODO: query the target CRM and return the matches.
func (c *custom) SearchHandler(w http.ResponseWriter, r *http.Request) {
	if _, err := api.GetSearchParameters(r); err != nil {
		httputil.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	httputil.JSONResponse(w, make([]*api.ObjectInfo, 0))
}

// ObjectGetHandler returns a single record by id.
// TODO: fetch the record from the target CRM.
func (c *custom) ObjectGetHandler(w http.ResponseWriter, r *http.Request) {
	httputil.Error(w, "not implemented", http.StatusNotImplemented)
}

// ObjectCreateHandler creates a record from the submitted fields.
// TODO: create the record in the target CRM and respond with api.ObjectInfo.
func (c *custom) ObjectCreateHandler(w http.ResponseWriter, r *http.Request) {
	httputil.Error(w, "not implemented", http.StatusNotImplemented)
}

// ObjectUpdateHandler updates a record's fields by id.
// TODO: update the record in the target CRM and respond with api.ObjectInfo.
func (c *custom) ObjectUpdateHandler(w http.ResponseWriter, r *http.Request) {
	httputil.Error(w, "not implemented", http.StatusNotImplemented)
}

// CreateTaskHandler logs a finished call as an activity/task in the CRM.
// TODO: create the activity in the target CRM.
func (c *custom) CreateTaskHandler(w http.ResponseWriter, r *http.Request) {
	httputil.Error(w, "not implemented", http.StatusNotImplemented)
}
