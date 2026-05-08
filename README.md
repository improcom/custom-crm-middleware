# CRM Middleware -- Example

Reference implementation of the [Bicom Systems CRM Middleware API](docs/API-docs.md), connecting Salesforce to PBXware. Use it as a starting point for building a custom CRM integration.

---

## Build and run

`make` - Builds the binary into `bin/`. Service must be run as root (state is written to `/var/lib/crm-middleware/`).

`make dev` - Runs the service directly from the project directory with all state files placed there and a default setup ready to use out of the box.

`make fresh` - Removes the dev state directory (`.crm-middleware/`). Use to reset to a clean state in dev mode.

## State files

Configuration, database, logs and other files are stored in `/var/lib/crm-middleware/`. In dev mode they are placed in the project directory as mentioned above.

The service listens on port **9996** by default. A config file (`config.yaml`) is created automatically at first run with all defaults applied.

## CLI

On ***systemd***-based operating systems, use `systemd install` to install the binary as a systemd service, or to update an existing installation with a binary that is executing command. *(not available in dev builds)*

```bash
crm-middleware systemd install
```

Use the `client` subcommand to manage clients against a running service:

```bash
crm-middleware client list
crm-middleware client create <name>
crm-middleware client create <name> --routing
crm-middleware client reset <id>
```

`create` issues a new Client ID and secret. Pass `--routing` to create a client with `ROUTING` status, which is required for the custom source routing endpoint (`/custom-source/routing/`). Without this flag the client is in a pending state until PBXware activates it through the standard setup flow.

`reset` regenerates the client secret (the API key used by PBXware for Basic Auth to the middleware). The Client ID is unchanged.

The `schema` tool is a separate binary (`make schema` builds it to `bin/schema`) for working with data schema files outside of the running service:

```bash
schema generate <file.yaml|dir/> [outdir]
schema validate <file.json|dir/>
```

`generate` compiles YAML model sources to JSON. `validate` checks JSON schema files and reports each as `[OK]` or `[FAIL]`.

---

## Salesforce

Salesforce integration requires a Connected App with OAuth enabled. Once created, set the config values and reload the service to apply:

```yaml
apps:
  salesforce:
    domain: <your-salesforce-domain>  # e.g. myorg.my.salesforce.com -- required for routing
    consumer:
      key: <consumer-key>
      secret: <consumer-secret>
```

Both OAuth2 authorization code and password grant auth flows are supported (support for both needs to be enabled in the Connected App with scopes: `openid`, `email`, `refresh_token`, `offline_access`, `api`), exposed under separate base paths:

| Base path | Auth method | Note |
|-----------|-------------| ---- |
| `/salesforce/user` | Authorization code -- user is redirected to Salesforce login | - |
| `/salesforce/direct` | Password grant -- credentials are passed directly | *Password is not plain user account password, but password with suffixed user security token* |

For the authorization code flow, the Connected App must have the callback URL whitelisted. The service constructs it from the `host` config values, which is set to `localhost` by default. Starting the service will output to `stderr` the callback URL that the service uses as verification during user authorization. *Service host values should match the domain name configured on the server.*

---

## Adding a new integration

1. Create a new package under `apps/` following the structure of other `apps`.
2. Define an object data model for each supported object type and embed or hard-code it in the app. See [`docs/data-model.md`](docs/data-model.md) for the format.
3. Mount the new app router in `main.go`.

---

## Object data model

Each app must supply a data model description for every object type it supports. The model is a JSON document that defines all fields PBXware will display and work with -- it is returned verbatim by the `GET /api/v1/object/describe/{type}` endpoint.

Writing and maintaining large JSON files by hand is tedious. As an alternative, models can be authored as compact YAML files (`models/`) and compiled to JSON (`schemas/`) with the `schema` tool (`make schema` to build it, `make schema-gen` to run it for registered apps). The YAML files are a developer-only authoring aid -- they are never exposed in the API or embedded directly in the binary; only the generated JSON is.

See [`docs/data-model.md`](docs/data-model.md) for the full JSON field reference, validation rules, and the YAML authoring format.

---

## API

Full specification: [`docs/API-docs.md`](docs/API-docs.md)

The middleware exposes three Salesforce app paths and one standalone routing endpoint. The first three are standard PBXware integrations; the third is mounted for routing use only and is not registered as a PBXware integration. All share the same data endpoints under `/api/v1`; only the base path and auth flow differ.

### Integration paths

| Base path | Login interface | Auth method |
|-----------|-----------------|-------------|
| `/salesforce/user` | Login URL prompts to Salesforce Login | OAuth2 Authorization Code |
| `/salesforce/direct` | Username+Password within integration | OAuth2 Password Grant |
| `/salesforce/system` | None | OAuth2 Client Credentials |

The OAuth2 callback for the authorization code flow is at `/salesforce/user/callback/code`.

### Endpoints

All endpoints except `GET /api/v1/token` require `Authorization: Bearer <JWT>`.

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/v1/token` | Exchange Basic auth credentials for a JWT |
| `GET` | `/api/v1/config` | Integration metadata (name, icon, auth method) |
| `GET` | `/api/v1/config/versions` | Supported API versions; activates the Client ID |
| `GET` | `/api/v1/config/objects` | Supported object types |
| `POST` | `/api/v1/config/custom-fields` | Receive custom field definitions from PBXware |
| `GET` | `/api/v1/auth/login_url` | Generate OAuth2 login URL (`/user` only) |
| `POST` | `/api/v1/auth/login` | Password login (`/direct` only) |
| `GET` | `/api/v1/auth/status` | Check user authorization status |
| `GET` | `/api/v1/auth/logout` | Invalidate user session |
| `GET` | `/api/v1/object/describe/{type}` | Object field schema |
| `POST` | `/api/v1/object/{type}` | Create a record |
| `GET` | `/api/v1/object/{type}/{id}` | Fetch a record |
| `PUT` | `/api/v1/object/{type}/{id}` | Update a record |
| `GET` | `/api/v1/search` | Search records |
| `GET` | `/api/v1/associations` | Search association fields |
| `POST` | `/api/v1/create-task` | Create an activity task |
| `DELETE` | `/api/v1/integration` | Delete all data for a client |

### Call routing

Standalone endpoint -- no JWT required, uses `X-Client-Id` header and client credentials OAuth2.

| Method | Path |
|--------|------|
| `GET \| POST` | `/custom-source/routing/{Contact\|Lead}` |

---
