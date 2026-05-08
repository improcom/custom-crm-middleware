package dataschema

import (
	"errors"
	"fmt"
)

type validator struct {
	*report
}

func validate() *validator {
	return &validator{}
}

func (v *validator) model(m *Model) error {
	if m == nil {
		return errors.New("model cannot be nil")
	}

	v.report = v.new("model <%s>", m.ID)

	v.mustNonEmpty("id", m.ID)
	v.mustNonEmpty("label", m.Label)
	v.mustLen("fields", len(m.Fields))

	if err := v.error(); err != nil {
		return err
	}

	for _, field := range m.Fields {
		v.must(validate().field(field))
	}

	seen := make(map[LogicType]bool)
	for _, field := range m.Fields {
		switch field.Logic {
		case LogicID, LogicTitle, LogicFirstName, LogicLastName, LogicMiddleName:
			if seen[field.Logic] {
				v.must(fmt.Errorf("logic type <%s> is assigned to multiple fields", field.Logic))
			}
			seen[field.Logic] = true
		}
	}

	if !seen[LogicID] {
		v.must(errors.New("a field with logic type <id> is required"))
	}

	return v.error()
}

func (v *validator) field(f *Field) error {
	if f == nil {
		return errors.New("field cannot be nil")
	}

	v.report = v.new("field <%s>", f.ID)

	v.mustNonEmpty("id", f.ID)
	v.mustNonEmpty("label", f.Label)

	if err := v.error(); err != nil {
		return err
	}

	v.must(validate().types(f.Type, f.Logic, f.Edit))

	var vis = f.Visibility
	switch f.Logic {
	case logicRecordURL: // skip internal

	case LogicID:
		if vis.View || vis.Create || vis.Edit || vis.Required || vis.Editable {
			v.must(errors.New("visibility of <id> field must be false"))
		}

	default:
		if !vis.View && !vis.Create && !vis.Edit {
			v.must(errors.New("at least one of 'view', 'create', or 'edit' must be enabled"))
		}
		if vis.Required && (!vis.Create || !vis.Edit) {
			v.must(errors.New("'required' field must also have 'create' and 'edit' enabled"))
		}
	}

	if f.Edit == EditList {
		v.must(validate().list(f.List))
	}

	if f.Type == ToOne || f.Type == ToMany {
		v.must(validate().association(f.Association))
	}

	return v.error()
}

func (v *validator) types(base BaseType, logic LogicType, edit EditType) error {
	v.report = v.new("types")

	v.mustNonEmpty("base type", string(base))
	v.mustNonEmpty("logic type", string(logic))
	v.mustNonEmpty("edit type", string(edit))

	if err := v.error(); err != nil {
		return err
	}

	switch base {
	case String, Integer, Float, Bool, Timestamp, ToOne, ToMany:
	default:
		v.must(fmt.Errorf("unknown base type <%s>", base))
	}

	switch logic {
	case LogicOther, LogicID, LogicPhone, LogicEmail,
		LogicTitle, LogicFirstName, LogicLastName, LogicMiddleName,
		LogicURL,
		logicRecordURL:
	default:
		v.must(fmt.Errorf("unknown logic type <%s>", logic))
	}

	switch edit {
	case EditText, EditTextarea, EditCheckbox, EditList, EditAssociation, EditDateTime:
	default:
		v.must(fmt.Errorf("unknown edit type <%s>", edit))
	}

	if logic != LogicOther && base != String {
		v.must(fmt.Errorf("base type <%s> cannot be combined with logic type <%s>", base, logic))
	}

	var valid = true
	switch edit {
	case EditText:
		valid = base == String || base == Integer || base == Float
	case EditTextarea, EditList:
		valid = base == String
	case EditCheckbox:
		valid = base == Bool
	case EditAssociation:
		valid = base == ToOne || base == ToMany
	case EditDateTime:
		valid = base == Timestamp
	}
	if !valid {
		v.must(fmt.Errorf("base type <%s> cannot be combined with edit type <%s>", base, edit))
	}

	return v.error()
}

func (v *validator) list(l *List) error {
	if l == nil {
		return errors.New("list cannot be nil")
	}

	v.report = v.new("list")

	if err := validate().item(l.Default); err != nil {
		v.must(fmt.Errorf("default item: %w", err))
	}

	v.mustLen("items", len(l.Items))
	for _, item := range l.Items {
		v.must(validate().item(item))
	}

	return v.error()
}

func (v *validator) item(i *Item) error {
	if i == nil {
		return errors.New("item cannot be nil")
	}

	v.report = v.new("item <%s<%s>>", i.ID, i.Value)

	v.mustNonEmpty("id", i.ID)
	v.mustNonEmpty("value", i.Value)

	return v.error()
}

func (v *validator) association(assoc *Association) error {
	if assoc == nil {
		return errors.New("association cannot be nil")
	}

	v.report = v.new("association <%s>", assoc.ObjectType)

	v.mustNonEmpty("object type", assoc.ObjectType)

	if err := validate().item(&assoc.Fields); err != nil {
		v.must(fmt.Errorf("fields mapping: %w", err))
	}

	return v.error()
}
