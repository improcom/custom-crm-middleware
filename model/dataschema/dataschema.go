package dataschema

import (
	"embed"
	"encoding/json"
)

type BaseType string

const (
	String    BaseType = "string" // default
	Integer   BaseType = "integer"
	Float     BaseType = "float"
	Bool      BaseType = "bool"
	Timestamp BaseType = "timestamp"
	ToOne     BaseType = "to_one"
	ToMany    BaseType = "to_many"
)

type LogicType string

const (
	LogicOther      LogicType = "other" // default
	LogicID         LogicType = "id"
	LogicPhone      LogicType = "phone"
	LogicEmail      LogicType = "email"
	LogicURL        LogicType = "url"
	LogicTitle      LogicType = "title"
	LogicFirstName  LogicType = "first_name"
	LogicLastName   LogicType = "last_name"
	LogicMiddleName LogicType = "middle_name"

	// --- internal --------------------------
	logicRecordURL LogicType = "record_url"
)

type EditType string

const (
	EditText        EditType = "text" // default
	EditTextarea    EditType = "textarea"
	EditCheckbox    EditType = "checkbox"
	EditList        EditType = "list"
	EditAssociation EditType = "association"
	EditDateTime    EditType = "datetime"
)

type Item struct {
	ID    string `json:"id"`
	Value string `json:"value"`
}

type Visibility struct {
	View     bool `json:"view"`
	Create   bool `json:"create"`
	Edit     bool `json:"edit"`
	Required bool `json:"required"`
	Editable bool `json:"editable"`
}

type List struct {
	Multiselect bool    `json:"multiselect"`
	Default     *Item   `json:"default,omitempty"`
	Items       []*Item `json:"items"`
}

type Association struct {
	ObjectType string `json:"object"`
	Fields     Item   `json:"fields"`
}

type Field struct {
	ID          string       `json:"id"`
	Label       string       `json:"label"`
	Type        BaseType     `json:"type"`
	Logic       LogicType    `json:"logic_type"`
	Edit        EditType     `json:"edit_type"`
	Visibility  Visibility   `json:"visibility"`
	List        *List        `json:"list,omitempty"`
	Association *Association `json:"relation,omitempty"`
}

type Model struct {
	ID         string   `json:"id"`
	Label      string   `json:"label"`
	Extendable bool     `json:"extendable"`
	Fields     []*Field `json:"fields"`
}

func MustLoad(fs embed.FS, filename string) *Model {
	data, err := fs.ReadFile(filename)
	if err != nil {
		panic("schema: failed to read " + filename + ": " + err.Error())
	}
	var m Model
	if err := json.Unmarshal(data, &m); err != nil {
		panic("schema: failed to parse " + filename + ": " + err.Error())
	}
	if err := ValidateModel(&m); err != nil {
		panic("schema: invalid " + filename + ": \n" + err.Error())
	}
	return &m
}

func ValidateModel(m *Model) error {
	return validate().model(m)
}

func ValidateField(f *Field) error {
	return validate().field(f)
}

func ValidateTypes(baseType BaseType, logicType LogicType, editType EditType) error {
	return validate().types(baseType, logicType, editType)
}

func ValidateList(l *List) error {
	return validate().list(l)
}

func ValidateItem(i *Item) error {
	return validate().item(i)
}

func ValidateAssociation(a *Association) error {
	return validate().association(a)
}
