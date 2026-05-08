package salesforce

import (
	"crm-middleware/api"
	"crm-middleware/httputil"
	"crm-middleware/repo"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func (salesforce *salesforce) ObjectGetHandler(w http.ResponseWriter, r *http.Request) {
	claims := api.GetClaims(r)
	obj := api.GetObject(r)

	schema, err := repo.DataSchemas.Load(obj.ObjectType, claims.ClientID)
	if err != nil {
		httputil.Error(w, "failed to get module", http.StatusInternalServerError,
			httputil.SlogError(err),
			claims.SlogAttr(),
			obj.SlogAttr(),
		)
		return
	}

	fields := getSchemaAPIFields(schema)
	if len(fields) == 0 {
		fields = []string{"Id"}
	}

	query := url.Values{}
	query.Set("fields", strings.Join(fields, ","))

	resp, err := salesforce.Request(claims.ClientID, claims.UserID, http.MethodGet, servicesDataEndpoint("sobjects", obj.ObjectType, obj.ObjectID, query), nil)
	if err != nil {
		httputil.Error(w, "salesforce get object failed", http.StatusInternalServerError,
			httputil.SlogError(err),
			claims.SlogAttr(),
			obj.SlogAttr(),
		)
		return
	}

	var data map[string]any
	if err := json.Unmarshal(resp, &data); err != nil {
		httputil.Error(w, "failed to read salesforce object", http.StatusInternalServerError,
			httputil.SlogError(err),
			claims.SlogAttr(),
			obj.SlogAttr(),
		)
		return
	}

	slog.Debug("salesforce object fetched", claims.SlogAttr(), obj.SlogAttr(), slog.Any("data", data))

	result := formatSObject(data, schema)

	if fieldsParam := r.URL.Query().Get("fields"); fieldsParam != "" {
		filtered := make(map[string]any)
		for f := range strings.SplitSeq(fieldsParam, ",") {
			if f = strings.TrimSpace(f); f != "" {
				filtered[f] = result[f]
			}
		}
		slog.Debug("returning filtered object fields", claims.SlogAttr(), obj.SlogAttr(), slog.String("fields", fieldsParam))
		httputil.JSONResponse(w, filtered)
		return
	}

	httputil.JSONResponse(w, result)
}

func (salesforce *salesforce) ObjectCreateHandler(w http.ResponseWriter, r *http.Request) {
	claims := api.GetClaims(r)
	obj := api.GetObject(r)

	body, err := io.ReadAll(r.Body)
	if err != nil {
		httputil.Error(w, err.Error(), http.StatusInternalServerError, claims.SlogAttr())
		return
	}

	var data map[string]any
	if err := json.Unmarshal(body, &data); err != nil {
		httputil.Error(w, "failed reading request body", http.StatusBadRequest,
			httputil.SlogError(err),
			claims.SlogAttr(),
		)
		return
	}

	payload, err := apiSObject(claims.ClientID, obj.ObjectType, data)
	if err != nil {
		httputil.Error(w, "failed preparing sobject payload", http.StatusInternalServerError,
			httputil.SlogError(err),
			claims.SlogAttr(),
			obj.SlogAttr(),
		)
		return
	}
	resp, err := salesforce.Request(claims.ClientID, claims.UserID, http.MethodPost, servicesDataEndpoint("sobjects", obj.ObjectType, "", nil), payload)
	if err != nil {
		httputil.Error(w, "failed to create salesforce object", http.StatusInternalServerError,
			httputil.SlogError(err),
			claims.SlogAttr(),
			obj.SlogAttr(),
		)
		return
	}

	data = nil
	if err := json.Unmarshal(resp, &data); err != nil {
		httputil.Error(w, "failed to parse salesforce create response", http.StatusInternalServerError,
			httputil.SlogError(err),
			claims.SlogAttr(),
		)
		return
	}

	objectID, ok := data["id"].(string)
	if !ok {
		httputil.Error(w, "salesforce create response missing id field", http.StatusInternalServerError,
			slog.String("data", string(resp)),
			claims.SlogAttr(),
			obj.SlogAttr(),
		)
		return
	}

	slog.Debug("salesforce object created", claims.SlogAttr(), obj.SlogAttr())

	httputil.JSONResponse(w, api.ObjectInfo{
		ID:                    objectID,
		Type:                  obj.ObjectType,
		LastModifiedTimestamp: time.Now().Unix(),
	})
}

func (salesforce *salesforce) ObjectUpdateHandler(w http.ResponseWriter, r *http.Request) {
	claims := api.GetClaims(r)
	obj := api.GetObject(r)

	body, err := io.ReadAll(r.Body)
	if err != nil {
		httputil.Error(w, err.Error(), http.StatusInternalServerError, claims.SlogAttr())
		return
	}

	var data map[string]any
	if err := json.Unmarshal(body, &data); err != nil {
		httputil.Error(w, "failed reading request body", http.StatusBadRequest,
			httputil.SlogError(err),
			claims.SlogAttr(),
		)
		return
	}

	payload, err := apiSObject(claims.ClientID, obj.ObjectType, data)
	if err != nil {
		httputil.Error(w, "failed preparing sobject payload", http.StatusInternalServerError,
			httputil.SlogError(err),
			claims.SlogAttr(),
			obj.SlogAttr(),
		)
		return
	}
	_, err = salesforce.Request(claims.ClientID, claims.UserID, http.MethodPatch, servicesDataEndpoint("sobjects", obj.ObjectType, obj.ObjectID, nil), payload)
	if err != nil {
		httputil.Error(w, "failed to update salesforce object", http.StatusInternalServerError,
			httputil.SlogError(err),
			claims.SlogAttr(),
			obj.SlogAttr(),
		)
		return
	}

	slog.Debug("salesforce object updated", claims.SlogAttr(), obj.SlogAttr())

	httputil.JSONResponse(w, api.ObjectInfo{
		ID:                    obj.ObjectID,
		Type:                  obj.ObjectType,
		LastModifiedTimestamp: time.Now().Unix(),
	})
}
