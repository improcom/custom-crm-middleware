package salesforce

import (
	"crm-middleware/api"
	"crm-middleware/httputil"
	"crm-middleware/model"
	"crm-middleware/model/dataschema"
	"crm-middleware/repo"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"slices"
	"strings"
)

func (salesforce *salesforce) SearchHandler(w http.ResponseWriter, r *http.Request) {
	claims := api.GetClaims(r)

	search, err := api.GetSearchParameters(r)
	if err != nil {
		httputil.Error(w, err.Error(), http.StatusBadRequest, claims.SlogAttr())
		return
	}

	if len(search.ObjectTypes) == 0 {
		search.ObjectTypes = []string{Account, Contact, Lead}
	}

	for _, ot := range search.ObjectTypes {
		if _, ok := salesforce.Objects[ot]; !ok {
			httputil.Error(w, "unsupported object type: "+ot, http.StatusBadRequest, claims.SlogAttr())
			return
		}
	}

	if search.Field != "" && !IsValidFieldName(search.Field) {
		httputil.Error(w, "invalid field name", http.StatusBadRequest, claims.SlogAttr())
		return
	}

	records, err := salesforce.search(claims.ClientID, claims.UserID, search)
	if err != nil {
		httputil.Error(w, "salesforce search failed", http.StatusInternalServerError,
			httputil.SlogError(err),
			search.SlogAttr(),
			claims.SlogAttr(),
		)
		return
	}

	httputil.JSONResponse(w, records)
}

func (salesforce *salesforce) SearchAssociationsHandler(w http.ResponseWriter, r *http.Request) {
	claims := api.GetClaims(r)

	search, err := api.GetSearchParameters(r)
	if err != nil {
		httputil.Error(w, err.Error(), http.StatusBadRequest, claims.SlogAttr())
		return
	}

	if len(search.ObjectTypes) != 1 {
		httputil.Error(w, "salesforce search association can query only single module", http.StatusBadRequest,
			claims.SlogAttr(),
			search.SlogAttr(),
		)
		return
	}

	if _, ok := salesforce.Objects[search.ObjectTypes[0]]; !ok {
		httputil.Error(w, "unsupported object type: "+search.ObjectTypes[0], http.StatusBadRequest, claims.SlogAttr())
		return
	}

	idField := search.AssociationFields.IDField
	labelField := search.AssociationFields.LabelField
	if !IsValidFieldName(idField) || !IsValidFieldName(labelField) {
		httputil.Error(w, "invalid association field name", http.StatusBadRequest, claims.SlogAttr())
		return
	}

	search.ReturningFields[0] = []string{idField, labelField}

	records, err := salesforce.search(claims.ClientID, claims.UserID, search)
	if err != nil {
		httputil.Error(w, "salesforce search associations failed", http.StatusInternalServerError,
			claims.SlogAttr(),
			search.SlogAttr(),
		)
		return
	}

	httputil.JSONResponse(w, records)
}

func (salesforce *salesforce) search(clientID string, userID string, search *api.SearchParameters) ([]*api.ObjectInfo, error) {
	var err error
	query := url.Values{}
	var endpoint string
	var schemasByType = make(map[string]*model.DataSchema)

	for _, objectType := range search.ObjectTypes {
		schemasByType[objectType], err = repo.DataSchemas.Load(objectType, clientID)
		if err != nil {
			return nil, err
		}
	}

	if search.Field != "" {

		var searchFieldType = dataschema.String
		for _, field := range schemasByType[search.ObjectTypes[0]].Fields {
			if field.ID == search.Field {
				searchFieldType = dataschema.BaseType(field.Type)
			}
		}

		var matchClause string
		if searchFieldType == dataschema.String {
			matchClause = fmt.Sprintf("LIKE '%%%s%%'", strings.ReplaceAll(search.Query, `'`, `''`))
		} else {
			matchClause = fmt.Sprintf("= %s", search.Query)
		}

		query.Set("q",
			fmt.Sprintf("SELECT %s FROM %s WHERE %s %s ORDER BY LastModifiedDate DESC LIMIT %s",
				strings.Join(getSchemaAPIFields(schemasByType[search.ObjectTypes[0]]), ","),
				search.ObjectTypes[0],
				search.Field,
				matchClause,
				search.Limit,
			),
		)

		endpoint = servicesDataEndpoint("query", "", "", query)

	} else {
		if search.By != "phone" && search.By != "email" {
			search.By = "all"
		}

		var returningObjects []string
		for idx, objectType := range search.ObjectTypes {

			if len(search.ReturningFields[idx]) == 0 {
				search.ReturningFields[idx] = getSchemaAPIFields(schemasByType[objectType])
			}
			if !slices.Contains(search.ReturningFields[idx], "LastModifiedDate") {
				search.ReturningFields[idx] = append(search.ReturningFields[idx], "LastModifiedDate")
			}

			returningObjects = append(returningObjects,
				fmt.Sprintf("%s(%s ORDER BY LastModifiedDate DESC LIMIT %s)", objectType, strings.Join(search.ReturningFields[idx], ","), search.Limit),
			)
		}

		query.Set("q",
			fmt.Sprintf("FIND {%s} IN %s FIELDS RETURNING %s",
				EscapeSOSLCharacters(search.Query),
				strings.ToUpper(search.By),
				strings.Join(returningObjects, ","),
			),
		)

		endpoint = servicesDataEndpoint("search", "", "", query)
	}

	resp, err := salesforce.Request(clientID, userID, http.MethodGet, endpoint, []byte{})
	if err != nil {
		return nil, fmt.Errorf("search request failed: %w", err)
	}

	var result struct {
		SOSLRecords []map[string]any `json:"searchRecords"` // SOSL response
		SOQLRecords []map[string]any `json:"records"`       // SOQL response
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse search response: %w", err)
	}

	var (
		records     []map[string]any
		objectInfos = make([]*api.ObjectInfo, 0)
	)

	if len(result.SOSLRecords) > 0 {
		records = result.SOSLRecords
	} else if len(result.SOQLRecords) > 0 {
		records = result.SOQLRecords
	}

	if len(records) > 0 {
		for _, record := range records {
			var info = &api.ObjectInfo{
				Phones: make([]string, 0),
				Emails: make([]string, 0),
			}

			attributes, ok := record["attributes"].(map[string]any)
			if !ok {
				continue
			}
			if info.Type, ok = attributes["type"].(string); !ok {
				continue
			}
			if _, ok := schemasByType[info.Type]; !ok {
				continue
			}

			info.ID, _ = record["Id"].(string)
			info.LastModifiedTimestamp = convertToUNIXTimestamp(record["LastModifiedDate"])

			var nameValues = make([]string, 0)
			for _, f := range schemasByType[info.Type].Fields {
				switch dataschema.LogicType(f.Logic) {
				case dataschema.LogicPhone:
					if val, ok := record[f.ID].(string); ok {
						info.Phones = append(info.Phones, val)
					}

				case dataschema.LogicEmail:
					if val, ok := record[f.ID].(string); ok {
						info.Emails = append(info.Emails, val)
					}

				case dataschema.LogicFirstName, dataschema.LogicLastName, dataschema.LogicMiddleName:
					if val, ok := record[f.ID].(string); ok {
						nameValues = append(nameValues, val)
					}
				}
			}

			if len(nameValues) == 0 {
				nameValues = append(nameValues, "Unknown")
			}

			info.DisplayName = strings.Join(nameValues, " ")

			objectInfos = append(objectInfos, info)
		}
	}

	slog.Debug("search complete", "results", len(objectInfos))
	return objectInfos, nil
}
