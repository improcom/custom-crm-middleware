package salesforce

import (
	"crm-middleware/api"
	"crm-middleware/httputil"
	"crm-middleware/model"
	"crm-middleware/repo"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	_ "embed"

	"github.com/go-chi/chi/v5"
)

type routingRequest struct {
	Number           string   `json:"number"`
	APIFields        string   `json:"api_fields"`
	CallerInput      string   `json:"caller_input"`
	InputTargetField string   `json:"input_target_field"`
	Fields           []string `json:"-"`
}

type recordWithIdField struct {
	ID string `json:"Id"`
}

//go:embed custom-source-guide/setup.html
var setupGuideHTML []byte

func RouterCustomSource(r chi.Router) {
	salesforce := App(api.AuthMethodNone)

	r.Route("/custom-source", func(r chi.Router) {
		r.Get("/routing/{object_type}", salesforce.customSourceRoutingHandler)
		r.Post("/routing/{object_type}", salesforce.customSourceRoutingHandler)

		r.Get("/setup", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write(setupGuideHTML)
		})
	})
}

func (salesforce *salesforce) customSourceRoutingHandler(w http.ResponseWriter, r *http.Request) {
	var clientID = r.Header.Get("X-Client-Id")
	if clientID == "" {
		httputil.Error(w, "client ID not provided in header", http.StatusForbidden)
		return
	}

	client, err := repo.Clients.Get(clientID)
	if err != nil {
		httputil.Error(w, "missing client", http.StatusForbidden, httputil.SlogError(err))
		return
	}

	if client.Status != model.ClientStatusRouting {
		httputil.Error(w, "requested client has no routing status", http.StatusForbidden)
		return
	}

	objectType := chi.URLParam(r, "object_type")
	if objectType != Contact && objectType != Lead {
		httputil.Error(w, "unsupported object type: only Contact and Lead are allowed", http.StatusBadRequest,
			slog.String("client-id", clientID),
		)
		return
	}

	req := &routingRequest{Fields: []string{"Id"}}
	if err := json.NewDecoder(r.Body).Decode(req); err != nil {
		slog.Debug("custom source: ignoring malformed request body, falling back to query params",
			slog.String("client-id", clientID),
			httputil.SlogError(err),
		)
	}
	r.Body.Close()

	q := r.URL.Query()
	if req.Number == "" {
		req.Number = q.Get("number")
	}
	if req.APIFields == "" {
		req.APIFields = q.Get("api_fields")
	}
	if req.CallerInput == "" {
		req.CallerInput = q.Get("caller_input")
	}
	if req.InputTargetField == "" {
		req.InputTargetField = q.Get("input_target_field")
	}

	seen := map[string]bool{"Id": true}
	for _, f := range strings.Split(req.APIFields, ",") {
		if f = strings.TrimSpace(f); f != "" && !seen[f] {
			if !IsValidFieldName(f) {
				httputil.Error(w, "invalid field name in api_fields: "+f, http.StatusBadRequest,
					slog.String("client-id", clientID),
				)
				return
			}
			seen[f] = true
			req.Fields = append(req.Fields, f)
		}
	}

	if req.InputTargetField != "" && !IsValidFieldName(req.InputTargetField) {
		httputil.Error(w, "invalid input_target_field name", http.StatusBadRequest,
			slog.String("client-id", clientID),
		)
		return
	}

	if req.InputTargetField != "" {
		if req.CallerInput == "" {
			httputil.Error(w, "caller_input is required when input_target_field is set", http.StatusBadRequest,
				slog.String("client-id", clientID),
			)
			return
		}
	} else if req.Number == "" {
		httputil.Error(w, "number is required for phone-based routing", http.StatusBadRequest,
			slog.String("client-id", clientID),
		)
		return
	}

	recordID, err := salesforce.findRecord(clientID, objectType, req)
	if err != nil {
		httputil.Error(w, err.Error(), http.StatusNotFound,
			slog.String("client-id", clientID),
		)
		return
	}

	record, err := salesforce.getRecord(clientID, objectType, recordID, req.Fields)
	if err != nil {
		httputil.Error(w, err.Error(), http.StatusInternalServerError,
			slog.String("client-id", clientID),
			slog.String("object-type", objectType),
			slog.String("record-id", recordID),
		)
		return
	}

	slog.Debug("custom source: record returned", "objectType", objectType, "record_id", recordID)
	httputil.JSONResponse(w, record)
}

func (salesforce *salesforce) getRecord(clientID, objectType, recordID string, fields []string) (map[string]any, error) {
	if recordID == "" {
		return nil, errors.New("record ID is required")
	}

	soql := fmt.Sprintf("SELECT %s FROM %s WHERE Id = '%s'", strings.Join(fields, ","), objectType, recordID)
	path := fmt.Sprintf("/services/data/v%s/query/?q=%s", Version, url.QueryEscape(soql))

	respBody, err := salesforce.Request(clientID, "routing", http.MethodGet, path, []byte{})
	if err != nil {
		return nil, fmt.Errorf("get record failed: %w", err)
	}

	var result struct {
		Records []map[string]any `json:"records"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("parsing record response: %w", err)
	}
	if len(result.Records) == 0 {
		return nil, fmt.Errorf("%s record %s not found", objectType, recordID)
	}

	record := result.Records[0]
	delete(record, "attributes")
	return record, nil
}

func (salesforce *salesforce) findRecord(clientID string, objectType string, req *routingRequest) (string, error) {
	var path string

	if req.InputTargetField != "" {
		soql := fmt.Sprintf(
			"SELECT Id FROM %s WHERE %s LIKE '%%%s%%' ORDER BY LastModifiedDate DESC LIMIT 1",
			objectType, req.InputTargetField, strings.ReplaceAll(req.CallerInput, "'", "''"),
		)
		path = fmt.Sprintf("/services/data/v%s/query/?q=%s", Version, url.QueryEscape(soql))
	} else {
		sosl := fmt.Sprintf(
			"FIND {%s} IN PHONE FIELDS RETURNING %s(Id LIMIT 1)",
			EscapeSOSLCharacters(req.Number), objectType,
		)
		path = fmt.Sprintf("/services/data/v%s/search/?q=%s", Version, url.QueryEscape(sosl))
	}

	respBody, err := salesforce.Request(clientID, "routing", http.MethodGet, path, []byte{})
	if err != nil {
		return "", fmt.Errorf("record search failed: %w", err)
	}

	var result struct {
		SearchRecords []recordWithIdField `json:"searchRecords"`
		Records       []recordWithIdField `json:"records"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("parsing search response: %w", err)
	}

	if len(result.SearchRecords) > 0 {
		return result.SearchRecords[0].ID, nil
	}
	if len(result.Records) > 0 {
		return result.Records[0].ID, nil
	}

	return "", fmt.Errorf("no %s records found for the given input", objectType)
}
