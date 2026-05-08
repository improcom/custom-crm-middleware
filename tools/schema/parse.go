package main

import (
	"crm-middleware/model/dataschema"
	"errors"
	"fmt"

	"gopkg.in/yaml.v3"
)

type yamlModel struct {
	ID     string      `yaml:"id"`
	Label  string      `yaml:"label"`
	Extend bool        `yaml:"extend"`
	Fields []yamlField `yaml:"fields"`
}

type yamlField struct {
	ID          string     `yaml:"id"`
	Label       string     `yaml:"label"`
	Type        string     `yaml:"type"`
	Logic       string     `yaml:"logic"`
	Edit        string     `yaml:"edit"`
	Default     string     `yaml:"default"`
	Required    bool       `yaml:"required"`
	Locked      bool       `yaml:"locked"`
	ReadOnly    bool       `yaml:"readonly"`
	MultiSelect bool       `yaml:"multiselect"`
	List        []yamlItem `yaml:"list"`
	Assoc       *yamlAssoc `yaml:"association"`
}

type yamlItem struct {
	ID    string `yaml:"id"`
	Label string `yaml:"label"`
}

type yamlAssoc struct {
	With  string `yaml:"with"`
	ID    string `yaml:"id"`
	Value string `yaml:"value"`
}

func parseYAML(content []byte) (*dataschema.Model, error) {
	var src yamlModel
	if err := yaml.Unmarshal(content, &src); err != nil {
		return nil, fmt.Errorf("parse YAML: %w", err)
	}

	var m dataschema.Model
	m.Fields = make([]*dataschema.Field, 0, len(src.Fields))

	m.ID = src.ID
	if m.ID == "" {
		return nil, errors.New("invalid model: missing id")
	}
	m.Label = src.Label
	if m.Label == "" {
		m.Label = m.ID
	}
	m.Extendable = src.Extend

	for _, sf := range src.Fields {
		notID := dataschema.LogicType(sf.Logic) != dataschema.LogicID
		f := &dataschema.Field{
			ID:    sf.ID,
			Label: sf.Label,
			Type:  dataschema.BaseType(sf.Type),
			Logic: dataschema.LogicType(sf.Logic),
			Edit:  dataschema.EditType(sf.Edit),
			Visibility: dataschema.Visibility{
				Required: sf.Required,
				Editable: notID && !sf.Locked && !sf.ReadOnly,
				View:     notID,
				Edit:     notID && (!sf.ReadOnly || sf.Required),
				Create:   notID && (!sf.ReadOnly || sf.Required),
			},
		}

		if f.ID == "" {
			return nil, errors.New("invalid field: missing id")
		}
		if f.Label == "" {
			f.Label = f.ID
		}
		if f.Type == "" {
			f.Type = dataschema.String
		}
		if f.Logic == "" {
			f.Logic = dataschema.LogicOther
		}
		if f.Edit == "" {
			switch f.Type {
			case dataschema.Bool:
				f.Edit = dataschema.EditCheckbox
			case dataschema.Timestamp:
				f.Edit = dataschema.EditDateTime
			case dataschema.ToOne, dataschema.ToMany:
				f.Edit = dataschema.EditAssociation
			default:
				f.Edit = dataschema.EditText
			}
		}

		if listLen := len(sf.List); listLen > 0 {
			f.List = &dataschema.List{
				Multiselect: sf.MultiSelect,
				Default:     new(dataschema.Item),
				Items:       make([]*dataschema.Item, 0, listLen),
			}
			for _, li := range sf.List {
				value := li.Label
				if value == "" {
					value = li.ID
				}
				item := dataschema.Item{ID: li.ID, Value: value}
				if sf.Default == item.ID {
					f.List.Default = &item
				}
				f.List.Items = append(f.List.Items, &item)
			}
			if f.List.Default.ID == "" {
				f.List.Default = f.List.Items[0]
			}
			f.Type = dataschema.String
			f.Logic = dataschema.LogicOther
			f.Edit = dataschema.EditList
		}

		if sf.Assoc != nil {
			if len(sf.List) > 0 {
				return nil, fmt.Errorf("field %q: list and association are mutually exclusive", sf.ID)
			}
			f.Association = &dataschema.Association{
				ObjectType: sf.Assoc.With,
				Fields:     dataschema.Item{ID: sf.Assoc.ID, Value: sf.Assoc.Value},
			}
			f.Type = dataschema.ToOne
			f.Edit = dataschema.EditAssociation
			if sf.MultiSelect {
				f.Type = dataschema.ToMany
			}
		}

		m.Fields = append(m.Fields, f)
	}

	return &m, nil
}
