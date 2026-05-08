package salesforce

import (
	"crm-middleware/api"
	"crm-middleware/model"
	"crm-middleware/model/dataschema"
	"crm-middleware/repo"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// validFieldName matches Salesforce API field names: letters, digits, underscores, optional dot
// for relationship traversal (e.g. "Account.Name", "Custom__c").
var validFieldNamePattern = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_.]*$`)

// IsValidFieldName reports whether s is a safe Salesforce field name for use in SOQL/SOSL.
func IsValidFieldName(s string) bool {
	return len(s) > 0 && len(s) <= 80 && validFieldNamePattern.MatchString(s)
}

var soslReservedChars = strings.NewReplacer(
	`\`, `\\`, `?`, `\?`, `&`, `\&`, `|`, `\|`, `!`, `\!`, `{`, `\{`, `}`, `\}`, `[`, `\[`, `]`, `\]`,
	`(`, `\(`, `)`, `\)`, `^`, `\^`, `~`, `\~`, `*`, `\*`, `:`, `\:`, `"`, `\"`, `'`, `\'`, `+`, `\+`, `-`, `\-`,
)

// EscapeSOSLCharacters escapes reserved SOSL characters in a search string.
func EscapeSOSLCharacters(s string) string {
	return soslReservedChars.Replace(s)
}

func getSchemaAPIFields(schema *model.DataSchema) []string {
	fields := make([]string, 0, len(schema.Fields))

	for _, f := range schema.Fields {
		switch dataschema.BaseType(f.Type) {
		case dataschema.ToOne:
			fields = append(fields, f.ID+".Id")
			fields = append(fields, f.ID+".Name")
		case dataschema.ToMany:
			// skip -- not supported in defined schemas
		default:
			fields = append(fields, f.ID)
		}
	}

	return fields
}

func baseApiPattern(with api.AuthMethod) string {
	switch with {
	case api.AuthMethodOAuth2:
		return "/salesforce/user"
	case api.AuthMethodBasicAuth:
		return "/salesforce/direct"
	case api.AuthMethodNone:
		return "/salesforce/system"
	}
	return "/salesforce"
}

func servicesDataEndpoint(path string, objectType string, objectID string, v url.Values) string {
	endpoint := fmt.Sprintf("/services/data/v%s/%s/%s", Version, path, objectType)
	if objectID != "" {
		endpoint += "/" + objectID
	}
	if len(v) > 0 {
		return endpoint + "?" + v.Encode()
	}
	return endpoint
}

func convertToUNIXTimestamp(v any) int64 {
	datetime, ok := v.(string)
	if !ok || datetime == "" {
		return 0
	}
	for _, format := range []string{
		"2006-01-02T15:04:05.000Z0700",
		"2006-01-02T15:04:05Z0700",
	} {
		if ts, err := time.Parse(format, datetime); err == nil {
			return ts.Unix()
		}
	}
	return 0
}

func convertFromUNIXTimestamp(ts int64) string {
	return time.Unix(ts, 0).UTC().Format("2006-01-02T15:04:05.000Z0700")
}

func apiSObject(clientID string, objectType string, data map[string]any) ([]byte, error) {
	schema, err := repo.DataSchemas.Load(objectType, clientID)
	if err != nil {
		return nil, err
	}

	delete(data, "Id")

	for _, field := range schema.Fields {
		if value, ok := data[field.ID]; ok {
			switch dataschema.BaseType(field.Type) {
			case dataschema.ToOne, dataschema.ToMany:
				data[field.ID+"Id"] = value
				delete(data, field.ID)

			case dataschema.Timestamp:
				if ts, ok := value.(float64); ok {
					data[field.ID] = convertFromUNIXTimestamp(int64(ts))
				}
			}
		}
	}

	return json.Marshal(data)
}

func formatSObject(so map[string]any, schema *model.DataSchema) map[string]any {
	record := make(map[string]any)

	for _, f := range schema.Fields {
		switch dataschema.BaseType(f.Type) {
		case dataschema.ToOne:
			if assoc, ok := so[f.ID].(map[string]any); ok {
				record[f.ID] = map[string]any{
					"Id":   assoc["Id"],
					"Name": assoc["Name"],
				}
			}
		case dataschema.ToMany:
			// skip
		case dataschema.Timestamp:
			if value, ok := so[f.ID]; ok {
				record[f.ID] = convertToUNIXTimestamp(value)
			}
		default:
			if value, ok := so[f.ID]; ok {
				record[f.ID] = value
			}
		}
	}

	return record
}
