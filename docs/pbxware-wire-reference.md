# PBXware Wire Reference — Custom Integration

Every request PBXware sends to the middleware, and the exact reply it expects
back. The activation handshake below is a **live capture** — recorded through
the tunnel on 2026-09-12, the day the "Custom" integration was successfully
activated against PBXware 7.

| | |
|---|---|
| App base path | `/custom` |
| Auth method | `basic-auth` |
| Full contracts | [`API-docs.md`](API-docs.md) |
| Setup & handover | [`../apps/custom/README.md`](../apps/custom/README.md) |

**Conventions**

- `$BASE` is the integration URL *including the app path*, e.g.
  `https://<host>/custom`. Every endpoint lives under `$BASE/api/v1`.
- `$ID` / `$KEY` are the middleware client credentials (PBXware calls them
  Client ID and API Key). Only `/token` uses them — as HTTP Basic auth.
- Every other call carries `Authorization: Bearer $TOKEN`, the JWT returned by
  `/token`.
- Per-user operations (login, status, search, objects, tasks) also carry a
  `user_id` header identifying the PBXware/gloCOM user.

---

## Part 1 — The activation handshake (live capture)

What happens when an admin fills in Client ID, API Key and URL in PBXware's
**Add Custom Integration** dialog and clicks OK. Six requests, in order. If any
of them fails — or a reply doesn't match the contract — PBXware shows only
*"Failed to activate custom integration. Please contact administrator."*

### 1. `GET /api/v1/token`

Basic credentials in, JWT out. PBXware also sends its internal integration id
in `X-Integration-ID`.

```bash
curl -u "$ID:$KEY" -H "X-Integration-ID: 06g9xxxxxxxxxxxxxxxxxxxxxx" \
     $BASE/api/v1/token
```

Reply — `200 OK` (captured):

```json
{"token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."}
```

### 2. `GET /api/v1/config`

Integration metadata. **Only answers while the client is pending** — after
activation this must return `409` (see below). The icon must be a
base64-encoded **240×240 .webp**.

```bash
curl -H "Authorization: Bearer $TOKEN" $BASE/api/v1/config
```

Reply — `200 OK` (captured):

```json
{
  "name": "Custom",
  "auth_method": "basic-auth",
  "image": "UklGRqoAAABXRUJQVlA4IJ4A..."
}
```

*`auth_method: "none"` is rejected by PBXware 7 — see Gotchas.*

### 3. `GET /api/v1/config/objects`

Object types the CRM supports. Each value becomes an `<Object Type>` in later
calls.

```bash
curl -H "Authorization: Bearer $TOKEN" $BASE/api/v1/config/objects
```

Reply — `200 OK` (captured):

```json
["Contact"]
```

### 4. `GET /api/v1/object/describe/Contact`

Field schema per object type — rendered in the integration's **Data Models**
tab. Called once for every type returned in step 3.

```bash
curl -H "Authorization: Bearer $TOKEN" $BASE/api/v1/object/describe/Contact
```

Reply — `200 OK` (captured, abridged):

```json
{
  "id": "Contact",
  "label": "Contact",
  "extendable": false,
  "fields": [
    { "id": "Id",        "type": "string", "logic_type": "id",         "edit_type": "text", "visibility": {"...": "..."} },
    { "id": "FirstName", "type": "string", "logic_type": "first_name", "edit_type": "text", "visibility": {"...": "..."} },
    { "id": "LastName",  "type": "string", "logic_type": "last_name",  "edit_type": "text", "visibility": {"...": "..."} },
    { "id": "Phone",     "type": "string", "logic_type": "phone",      "edit_type": "text", "visibility": {"...": "..."} },
    { "id": "Email",     "type": "string", "logic_type": "email",      "edit_type": "text", "visibility": {"...": "..."} }
  ]
}
```

### 5. `GET /api/v1/config/versions`

**The activation call — one-shot.** Returning this response is the moment the
middleware must flip the client to ACTIVE. From here on `/config` refuses and
the client can never be activated again.

```bash
curl -H "Authorization: Bearer $TOKEN" $BASE/api/v1/config/versions
```

Reply — `200 OK` (captured):

```json
["1.0"]
```

### 6. Verification round

PBXware immediately fetches a fresh token and re-reads `/config/versions` to
verify the integration is up. Same requests, same replies as steps 1 and 5.

### After activation, `/config` must refuse

Captured from the live system — correct behavior, not an error:

```text
GET /api/v1/config  →  409 Conflict
client already activated, configuration not allowed
```

That also means the handshake cannot be replayed with the same client. To
re-test activation, mint a fresh client (`crm-middleware client create <name>`)
— or run [`activation-demo.sh`](../activation-demo.sh) from the repo root,
which replays steps 1–5 exactly.

---

## Part 2 — Runtime endpoints (to be implemented)

Called after activation, during normal use. In the skeleton these are stubs
(marked `TODO` in [`apps/custom/custom.go`](../apps/custom/custom.go)) — the
bodies below are the contracts from [`API-docs.md`](API-docs.md) that the real
implementation must honor. All carry `Authorization: Bearer $TOKEN`; per-user
calls also carry `user_id`.

### `POST /api/v1/auth/login`

A user typed CRM credentials into gloCOM/PBXware. Validate them against the
CRM and persist the session. **The skeleton currently accepts anything.**

```bash
curl -X POST -H "Authorization: Bearer $TOKEN" -H "user_id: 104" \
     -d '{"username": "agent@example.com", "password": "..."}' \
     $BASE/api/v1/auth/login
```

Reply — `200 OK` (valid credentials):

```json
{"status": "ok", "message": "authorized"}
```

Reply — `401 Unauthorized` (invalid credentials):

```json
{"status": "error", "message": "<reason>"}
```

### `GET /api/v1/auth/status`

Has this user logged in before? `email` is the user's CRM identifier, empty if
unknown. The skeleton answers `true` for everyone — wire it to real session
state.

```bash
curl -H "Authorization: Bearer $TOKEN" -H "user_id: 104" $BASE/api/v1/auth/status
```

Reply — `200 OK`:

```json
{"authorized": true, "email": "agent@example.com"}
```

### `GET /api/v1/auth/logout`

Invalidate the user's CRM session. Reply is just a status code — no body.

```bash
curl -H "Authorization: Bearer $TOKEN" -H "user_id: 104" $BASE/api/v1/auth/logout
```

### `GET /api/v1/search`

**The call-popup endpoint** — fires when a call rings and PBXware looks up the
caller. The skeleton returns `[]` (no match, no popup).

| Query param | Notes |
|---|---|
| `q` | Required. The search string (caller number, email, name…). |
| `search_by` | `phone`, `email`, or `field`. |
| `object_types` | Repeatable: `object_types=Contact&object_types=Lead`. |
| `phones` | Repeatable; alternative formats of the caller number sent with `search_by=phone` — use them to broaden the lookup. |
| `field` | Target field, only with `search_by=field` (exactly one object type). |
| `limit` | Max results (middleware clamps to 1–10). |

```bash
# inbound call from +1 443 111 4565
curl -H "Authorization: Bearer $TOKEN" -H "user_id: 104" \
  "$BASE/api/v1/search?q=%2B14431114565&search_by=phone&object_types=Contact&phones=14431114565&limit=10"
```

Reply — `200 OK`:

```json
[
  {
    "id": "0031x00001abcDEF",
    "type": "Contact",
    "title": "John Doe",
    "emails": ["djohn@example.com"],
    "phones": ["443-111-4565"],
    "last_modified_timestamp": 1716569790
  }
]
```

### `GET /api/v1/associations`

Populates relation pickers (`to_one`/`to_many` fields). Requires `q`, exactly
one `object_type`, and `fields=<id_field>,<label_field>`.

```bash
curl -H "Authorization: Bearer $TOKEN" -H "user_id: 104" \
  "$BASE/api/v1/associations?q=acme&object_type=Account&fields=Id,Name"
```

Reply — `200 OK`:

```json
[
  { "id": "0011x000...", "type": "Account", "title": "Acme Corp", "last_modified_timestamp": 1716569790 }
]
```

### `GET /api/v1/object/<Type>/<ID>`

Fetch one full record. Optional `?fields=` hints which field list PBXware
expects. Association fields come back as nested `{Id, Name}` objects; empty
fields as `null`.

```bash
curl -H "Authorization: Bearer $TOKEN" -H "user_id: 104" \
     $BASE/api/v1/object/Contact/0031x00001abcDEF
```

Reply — `200 OK`:

```json
{
  "Id": "0031x00001abcDEF",
  "FirstName": "John",
  "LastName": "Doe",
  "Phone": "443-111-4565",
  "Email": "djohn@example.com"
}
```

### `POST /api/v1/object/<Type>`

Create a record. Body field names must match the object's data model; all
fields marked required must be present.

```bash
curl -X POST -H "Authorization: Bearer $TOKEN" -H "user_id: 104" \
     -d '{"FirstName": "John", "LastName": "Doe", "Phone": "443-111-4565", "Email": "djohn@example.com"}' \
     $BASE/api/v1/object/Contact
```

Reply — `200 OK`:

```json
{ "id": "0031x00001abcDEF", "type": "Contact", "last_modified_timestamp": 1716569790 }
```

### `PUT /api/v1/object/<Type>/<ID>`

Update only the fields present in the body. Reply shape identical to Create.

```bash
curl -X PUT -H "Authorization: Bearer $TOKEN" -H "user_id: 104" \
     -d '{"Phone": "443-111-4466"}' \
     $BASE/api/v1/object/Contact/0031x00001abcDEF
```

Reply — `200 OK`:

```json
{ "id": "0031x00001abcDEF", "type": "Contact", "last_modified_timestamp": 1716569801 }
```

### `POST /api/v1/create-task`

Log a finished conversation (call, SMS, chat…) as a CRM activity. Reply is
**just `201 Created`** — any other status means failure.

```bash
curl -X POST -H "Authorization: Bearer $TOKEN" -H "user_id: 104" \
     -d '{
       "name": "Inbound call - Answered",
       "conversation_id": "1699999999.123",
       "tenant_code": "345",
       "ext": "104",
       "customer_id": "+14431114565",
       "answered": 1,
       "duration": 214,
       "subject": "Inbound call - Answered",
       "description": "Call from +14431114565 to ext 104",
       "channel": "voice",
       "conversation_finished": true,
       "conversation_start_time": 1789214000,
       "object_type": "Contact",
       "object_id": "0031x00001abcDEF"
     }' \
     $BASE/api/v1/create-task
```

Reply — `201 Created`, empty body.

### `POST /api/v1/config/custom-fields`

Admin-defined custom fields, pushed from PBXware, keyed by object type. Best
practice: replace all stored custom fields on every call. The framework
already persists these; declare `"extendable": false` in a schema to opt an
object out.

```bash
curl -X POST -H "Authorization: Bearer $TOKEN" \
     -d '{"Contact": [{"id": "custom_test", "type": "string", "logic_type": "other", "edit_type": "text"}]}' \
     $BASE/api/v1/config/custom-fields
```

### `DELETE /api/v1/integration`

The admin removed the integration in PBXware. Delete everything stored for
this client. **The client's credentials stop working permanently** — re-adding
the integration needs a fresh client.

```bash
curl -X DELETE -H "Authorization: Bearer $TOKEN" $BASE/api/v1/integration
```

Reply — `200 OK` (captured): `{}`

---

## Part 3 — Gotchas that cost us time

1. **PBXware's only error message is generic.** Whatever fails, the admin sees
   "Failed to activate custom integration. Please contact administrator."
   Diagnose from the middleware side: watch which request arrives last and
   what you answered (ngrok's inspector at `localhost:4040` is ideal in dev).
2. **The URL must include the app base path.** With a bare hostname PBXware
   calls `/api/v1/token` at the root, gets 404, and activation dies on
   request one.
3. **`auth_method: "none"` is rejected by PBXware 7** — it fetches `/config`
   (HTTP 200) and silently aborts, even though the middleware docs list "none"
   as valid. Confirmed by control test against the reference
   `/salesforce/system` app. Use `basic-auth` or `oauth2`.
4. **The icon must be a 240×240 `.webp`** ([`API-docs.md`](API-docs.md),
   Config section). Suspected of causing the same silent abort when wrong.
   `brew install webp && cwebp icon.png -o icon.webp` — macOS `sips` can't
   write webp.
5. **`/config/versions` activates the client, once, forever.** Never call it
   while poking around with curl — that burns the client and the real
   activation then gets 409 from `/config`. Reset in dev:
   `sqlite3 .crm-middleware/crm-middleware.db "UPDATE clients SET status='', app='' WHERE id='<id>'"`.
6. **One client per integration, never reused.** Lifecycle: pending → ACTIVE
   (versions call) → DELETED (integration removed in PBXware). Mint clients
   with `crm-middleware client create <name>`.
7. **Building the dev CLI:** `go run main.go client …` silently prints usage.
   Build a binary named exactly `crm-middleware`:
   `CGO_ENABLED=0 go build -tags dev -ldflags "-X 'crm-middleware/build.serviceName=crm-middleware'" -o /tmp/cli/crm-middleware .`
   and run it from the repo root. `make dev` deletes `bin/`, so build it
   elsewhere.
