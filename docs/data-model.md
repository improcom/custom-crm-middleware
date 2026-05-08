# Object Data Model

An object data model is a JSON document that describes all fields of a single CRM object type. It is returned by the `GET /api/v1/object/describe/{type}` endpoint and is the contract between the middleware and PBXware: PBXware uses it to know what fields exist, how to display them, and which are required or read-only.

Each app must embed or hard-code a valid model for every object type it declares in `GET /api/v1/config/objects`.

---

## JSON format

```json
{
    "id": "Contact",
    "label": "Contact",
    "extendable": false,
    "fields": [ ... ]
}
```

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Object type identifier. Must match the value used in URL paths and `config/objects`. |
| `label` | string | Display name shown in PBXware. |
| `extendable` | bool | Whether PBXware may send custom field definitions for this object via `POST /api/v1/config/custom-fields`. |
| `fields` | array | Ordered list of field definitions (see below). |

### Field object

```json
{
    "id": "Phone",
    "label": "Phone Number",
    "type": "string",
    "logic_type": "phone",
    "edit_type": "text",
    "visibility": {
        "view": true,
        "create": true,
        "edit": true,
        "required": false,
        "editable": true
    }
}
```

| Property | Type | Description |
|----------|------|-------------|
| `id` | string | Field identifier. Does not have to match the underlying CRM API field name -- the app can translate -- but mapping them one-to-one is encouraged. |
| `label` | string | Display name. Defaults to `id` if omitted. |
| `type` | string | Base type. See [Base types](#base-types). |
| `logic_type` | string | Semantic role. See [Logic types](#logic-types). |
| `edit_type` | string | Input widget. See [Edit types](#edit-types). |
| `visibility` | object | Controls where and how the field appears. See [Visibility](#visibility). |
| `relation` | object or null | Required when `type` is `to_one` or `to_many`. See [Relation](#relation). |
| `list` | object or null | Required when `edit_type` is `list`. See [List](#list). |

### Base types

| Value | Description |
|-------|-------------|
| `string` | Plain text (default) |
| `integer` | Whole number |
| `float` | Decimal number |
| `bool` | Boolean |
| `timestamp` | Unix timestamp |
| `to_one` | Single association to another object |
| `to_many` | Multiple associations to another object |

### Logic types

Logic type is a semantic hint that tells PBXware how to interpret and use a field -- for example, which field holds the phone number to match against an incoming call.

| Value | Description |
|-------|-------------|
| `other` | No special semantic role (default) |
| `id` | Internal record identifier. Never shown in view or edit forms. Every model must have exactly one field with this logic type. |
| `first_name` | First name component of the record title |
| `last_name` | Last name component of the record title |
| `middle_name` | Middle name component of the record title |
| `title` | Full title or name of the record |
| `phone` | Phone number field |
| `email` | Email address field |
| `url` | URL field |

Logic types `id`, `first_name`, `last_name`, `middle_name`, and `title` must each appear at most once per model. `logic_type` other than `other` can only be used with `type: string`.

### Edit types

| Value | Description |
|-------|-------------|
| `text` | Single-line text input (default for string, integer, float) |
| `textarea` | Multi-line text input |
| `checkbox` | Boolean toggle (default for bool) |
| `datetime` | Date and time picker (default for timestamp) |
| `list` | Dropdown select. Requires a `list` object on the field. |
| `association` | Record picker for linked objects (default for to_one, to_many). Requires a `relation` object on the field. |

### Visibility

Controls in which contexts a field is shown or required.

| Property | Type | Description |
|----------|------|-------------|
| `view` | bool | Shown when viewing a record |
| `create` | bool | Shown in the create form |
| `edit` | bool | Shown in the edit form |
| `required` | bool | Must be filled when creating or editing |
| `editable` | bool | Field visibility can be altered by PBXware admin |

A field with `logic_type: id` must have all visibility flags set to `false`. For all other fields, at least one of `view`, `create`, or `edit` must be `true`.

### Relation

Required for `to_one` and `to_many` fields. Specifies which object type to link to and which of its fields carry the record ID and display label.

```json
"relation": {
    "object": "Account",
    "fields": {
        "id": "Id",
        "value": "Name"
    }
}
```

| Property | Description |
|----------|-------------|
| `object` | Object type of the linked record |
| `fields.id` | Field on the linked object used as the record ID |
| `fields.value` | Field on the linked object used as the display label |

### List

Required when `edit_type` is `list`. Defines the available options and the default selection.

```json
"list": {
    "multiselect": false,
    "default": { "id": "Warm", "value": "Warm" },
    "items": [
        { "id": "Cold", "value": "Cold" },
        { "id": "Warm", "value": "Warm" },
        { "id": "Hot",  "value": "Hot"  }
    ]
}
```

| Property | Description |
|----------|-------------|
| `multiselect` | Whether multiple items can be selected |
| `default` | The item selected by default. Must be one of the items in the list. |
| `items` | All available options. Each item has `id` (the value sent to the CRM) and `value` (the display label; defaults to `id` if omitted). |

---

## Simplified YAML format (schema-gen)

Instead of writing JSON by hand, models can be authored as compact YAML files and compiled to JSON with `make schema-gen`. The YAML files live in the app's `models/` directory; generated JSON is written to `schemas/` and embedded in the binary at build time. YAML files are a developer-only authoring tool -- they are not embedded in the binary and not exposed in the API.

### YAML model structure

```yaml
id: Contact        # object type identifier (required)
label: Contact     # display name (defaults to id)
extend: false      # sets extendable on the model
fields:
  - ...
```

### YAML field properties

| Property | Type | Default | Description |
|----------|------|---------|-------------|
| `id` | string | -- | Field identifier, as used by the CRM API (required) |
| `label` | string | same as `id` | Display name |
| `type` | string | `string` | Base type: `string`, `integer`, `float`, `bool`, `timestamp`, `to_one`, `to_many` |
| `logic` | string | `other` | Logic type: `id`, `first_name`, `last_name`, `middle_name`, `title`, `phone`, `email`, `url`, `other` |
| `edit` | string | inferred | Edit type: `text`, `textarea`, `checkbox`, `list`, `association`, `datetime` |
| `required` | bool | false | Maps to `visibility.required`; also forces the field into create and edit forms even if `readonly` |
| `readonly` | bool | false | Field appears in view only; excluded from create/edit forms (unless also `required`) |
| `locked` | bool | false | Field `visibility.editable` is set to false |
| `default` | string | -- | Default item ID from `list` property - if omitted, first element is considered as default |
| `multiselect` | bool | false | Allows `list` and `association` fields to accept multiple values |
| `list` | items | -- | Dropdown options; each item has `id` and optional `label` |
| `association` | object | -- | Relation config: `with` (object type), `id`, `value` |

### Notes

To understand inference rules for each property, follow `tools/schema-gen/parse.go`.

`apps/salesforce/models` hosts good examples with corresponding generated `apps/salesforce/schemas`. Generally, YAML field properties are clear enough to intuitively define a data model description that generates a valid JSON spec. `tools/schema-gen` will be successful only if the generated JSON satisfies the data model validity check; with clear validation error messages, it helps identify issues in a YAML object's data model description.
