# Custom CRM Integration — Developer Handover

Skeleton of a **"Custom"** CRM integration for PBXware, built on Bicom's
CRM middleware reference implementation. It was added, activated, and
enabled against a live PBXware 7 system on 2026-09-12, so the plumbing is
proven — every handler marked `TODO` in `custom.go` is where the real CRM
logic goes.

```
PBXware  ⇄  this middleware (Go service)  ⇄  your CRM
```

PBXware speaks a fixed REST protocol to the middleware
([`docs/API-docs.md`](../../docs/API-docs.md)); the middleware translates
those calls to the target CRM. `apps/salesforce` is Bicom's complete
reference implementation — copy its patterns.

## What's in this package

| File | Purpose |
|------|---------|
| `custom.go` | App registration (`App()`) + all HTTP handlers. Stubs marked `TODO`. |
| `schemas/Contact.json` | Field schema for the Contact object (PBXware's "Data Models" tab). |
| `icons/custom.webp` | Integration icon. **Must be 240×240 .webp** — see Nuances. |
| `../../activation-demo.sh` | Replays PBXware's exact activation request sequence with curl. |

The app is mounted in `main.go` at base path **`/custom`**, name
**"Custom"**, auth method **`basic-auth`**.

## Running locally

```bash
make dev        # runs from source; state (config, sqlite DB, logs) in ./.crm-middleware/
make fresh      # wipe dev state and start clean
```

Service listens on **localhost:9996**. PBXware needs a public **HTTPS**
URL, so for development tunnel it (e.g. `ngrok http 9996`) and remember
the free-tier ngrok hostname changes on every restart — the URL in
PBXware must be updated to match.

### Creating API clients

PBXware authenticates to the middleware with a Client ID + secret
("API Key" in the PBXware dialog). Mint them with the CLI:

```bash
# NOTE: `go run main.go client ...` does NOT work (the CLI inspects the
# binary name). Build a dev-tagged binary named exactly `crm-middleware`:
CGO_ENABLED=0 go build -tags dev \
  -ldflags "-X 'crm-middleware/build.serviceName=crm-middleware'" \
  -o /tmp/cli/crm-middleware .

/tmp/cli/crm-middleware client create myclient   # prints ID + secret
/tmp/cli/crm-middleware client list
```

Run it from the repo root (it opens `./.crm-middleware/crm-middleware.db`).
Also note `make dev` deletes `bin/`, so don't build the CLI there.

## Adding the integration in PBXware

1. Select the tenant → **CRM → Integrations → Add Custom Integration**.
2. Fill **Client ID**, **API Key**, and **URL** — the URL **must include
   the base path**: `https://<host>/custom`, not just `https://<host>`.
3. OK → PBXware runs the activation sequence (below). On success you land
   on the integration's Configuration page.
4. Toggle **Active** on and **Save** → the integration appears under
   Active Integrations.

## The activation protocol

Observed live (ngrok inspector) — this is exactly what PBXware sends:

```text
1. GET /custom/api/v1/token                    Basic <ClientID:APIKey>
      + header X-Integration-ID: <PBXware's internal id>   → {"token": <JWT>}
   (all following requests: Authorization: Bearer <JWT>)
2. GET /custom/api/v1/config                   → name, auth_method, icon
3. GET /custom/api/v1/config/objects           → ["Contact"]
4. GET /custom/api/v1/object/describe/Contact  → field schema
5. GET /custom/api/v1/config/versions          → ["1.0"]   ← ACTIVATES the client
6. (fresh token + versions again to verify)
```

Reproduce it any time with `activation-demo.sh` against a **fresh**
client:

```bash
BASE=https://<host>/custom ID=<client-id> KEY=<api-key> ./activation-demo.sh
```

## Nuances (hard-won — read before touching anything)

- **PBXware's failure mode is a single generic error.** "Failed to
  activate custom integration. Please contact administrator." is all you
  ever get, whatever the cause. Diagnose from the middleware side: watch
  which requests arrive and where the sequence stops (ngrok's inspector
  at `localhost:4040` is ideal).
- **The URL must include the app base path.** A bare hostname makes
  PBXware call `/api/v1/token` at the root → 404 → generic error.
- **`auth_method: "none"` is rejected by PBXware 7** during custom
  integration activation, even though `docs/API-docs.md` lists it as
  valid. PBXware fetches `/config` (HTTP 200) and silently aborts.
  Confirmed by control test with the reference `/salesforce/system` app,
  which differs from a working app only in auth method. Use
  **`basic-auth`** or **`oauth2`**.
- **The icon must be a 240×240 `.webp`** (`docs/API-docs.md`, Config
  section). Suspected to cause the same silent abort if wrong (we
  switched from PNG before isolating it; don't burn time re-testing).
  `brew install webp && cwebp icon.png -o icon.webp` — macOS `sips`
  cannot write webp.
- **`GET /config/versions` is one-shot.** It flips the client to
  `ACTIVE`; afterwards `/config` answers **409** and the client can never
  be activated again. Never call it while testing by hand. To re-run an
  activation, create a fresh client, or reset one:

  ```bash
  sqlite3 .crm-middleware/crm-middleware.db \
    "UPDATE clients SET status='', app='' WHERE id='<client-id>'"
  ```

- **Deleting the integration in PBXware kills the client.** PBXware calls
  `DELETE /api/v1/integration`; the middleware marks the client
  `DELETED` and its credentials stop working. Re-adding needs a new (or
  reset) client.
- **Client lifecycle:** pending (`''`) → `ACTIVE` (versions call) →
  `DELETED` (integration removed). One client belongs to one integration.

## What's stubbed — the actual development work

All in `custom.go`, each marked `TODO`:

| Handler | Called when | Currently |
|---------|-------------|-----------|
| `LoginHandler` | A user enters username/password in gloCOM/PBXware | **Accepts ANY credentials** — replace with real validation + session persistence |
| `SearchHandler` | Contact popup on calls, manual lookups, associations | Returns empty list |
| `ObjectGetHandler` | Opening a record | 501 |
| `ObjectCreateHandler` | Creating a record from PBXware | 501 |
| `ObjectUpdateHandler` | Editing a record | 501 |
| `CreateTaskHandler` | Logging a finished call as a CRM activity | 501 |

Plus:

- `AuthHandlers.Status` currently reports every user as authorized;
  wire it to real session state (see `api.OAuth2StatusHandlerFunc` usage
  in `apps/salesforce` for the pattern).
- Extend `schemas/Contact.json` with the real field set, or author it as
  YAML and compile with the `schema` tool (`make schema`,
  [`docs/data-model.md`](../../docs/data-model.md)).
- Add more object types by adding schemas + entries in
  `objectDescriptions`.
- Custom fields from PBXware (`POST /config/custom-fields`) are already
  persisted by the framework.

## References

- [`docs/pbxware-wire-reference.md`](../../docs/pbxware-wire-reference.md) — every request PBXware sends and the reply it expects, with the live-captured activation handshake
- [`docs/troubleshooting-prompt.md`](../../docs/troubleshooting-prompt.md) — ready-made prompt for an AI coding agent to diagnose and fix activation failures
- [`docs/API-docs.md`](../../docs/API-docs.md) — full endpoint contracts
- [`docs/data-model.md`](../../docs/data-model.md) — schema format
- [`apps/salesforce`](../salesforce) — complete reference implementation
