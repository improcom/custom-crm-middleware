# Troubleshooting Prompt — PBXware Custom Integration

Copy-paste the block below into an AI coding agent (Claude Code, etc.)
working in this repo when PBXware fails to add/activate the Custom
integration or an integration operation misbehaves. It encodes the known
failure modes and the diagnosis procedure, so the agent doesn't have to
rediscover them. Add your specific symptom at the end if you have one.

```text
You are working in the repo improcom/custom-crm-middleware (branch
custom-integration-skeleton). It is Bicom's CRM middleware reference
implementation plus a skeleton "Custom" integration in apps/custom/,
mounted in main.go at base path /custom. The integration was verified
working against PBXware 7 on 2026-09-12: added, activated, and enabled.

TASK: PBXware fails to add/activate the Custom integration (or an
integration operation misbehaves). Find the root cause and fix it.
Do not guess — locate evidence first, then fix, then verify.

READ FIRST (they document known failure modes and the exact protocol):
- docs/pbxware-wire-reference.md  — every request PBXware sends and the
  reply it expects, including a live-captured activation handshake
- apps/custom/README.md           — setup, client lifecycle, gotchas
- docs/API-docs.md                — Bicom's full endpoint contracts

KEY FACT: PBXware's only error message is generic ("Failed to activate
custom integration. Please contact administrator.") no matter what went
wrong. All diagnosis happens on the MIDDLEWARE side.

DIAGNOSIS PROCEDURE (in order):

1. Is the service up and reachable from the internet?
   - `make dev` runs it on localhost:9996; state lives in ./.crm-middleware/
   - PBXware needs public HTTPS. If tunneling: `ngrok http 9996`, and note
     the free-tier hostname changes on every restart — the URL configured
     in PBXware must match the CURRENT hostname.
   - Sanity check: curl the public URL yourself before blaming PBXware.

2. Capture the request trail. Use ngrok's inspector
   (http://127.0.0.1:4040, API at /api/requests/http) or the service log
   (.crm-middleware/app.log). Trigger the failure in PBXware, then look at
   which requests arrived, in what order, with what status codes.
   The healthy activation sequence is:
     GET /custom/api/v1/token            -> 200 (Basic auth, returns JWT)
     GET /custom/api/v1/config           -> 200
     GET /custom/api/v1/config/objects   -> 200
     GET /custom/api/v1/object/describe/<Type> -> 200 (per object type)
     GET /custom/api/v1/config/versions  -> 200 (ACTIVATES the client)
     (then a fresh token + versions again as verification)
   Where the trail stops tells you where to look:

   - NO requests at all -> PBXware can't reach you: wrong URL, dead
     tunnel, DNS/TLS problem.
   - "GET /api/v1/token -> 404" (no /custom prefix) -> the URL in PBXware
     is missing the base path. It must be https://<host>/custom.
   - token -> 403 -> wrong Client ID / API key, or the client is DELETED
     (deleting the integration in PBXware kills its client permanently).
   - config -> 409 "client already activated" -> the client was already
     activated once (config/versions is ONE-SHOT). Someone probably
     curl-tested it. Reset:
       sqlite3 .crm-middleware/crm-middleware.db \
         "UPDATE clients SET status='', app='' WHERE id='<client-id>'"
     or mint a fresh client.
   - token 200, config 200, then SILENCE (PBXware stops without calling
     auth/status or versions) -> PBXware rejected the CONTENT of the
     /config response. Known causes, confirmed by experiment:
       * auth_method "none" — PBXware 7 rejects it for custom
         integrations even though docs allow it. Use "basic-auth".
       * icon not a 240x240 .webp (docs require exactly that).
     Compare your /config JSON against the captured one in
     docs/pbxware-wire-reference.md.
   - describe/<Type> -> 500 or panic on startup -> schema JSON invalid;
     validate against apps/salesforce/schemas/*.json field structure.

3. Check client state when in doubt:
     sqlite3 .crm-middleware/crm-middleware.db \
       "SELECT id,name,app,status FROM clients;"
   Lifecycle: '' (pending) -> ACTIVE (versions call) -> DELETED
   (integration removed in PBXware). One client per integration, never
   reused across systems/tenants.

4. Managing clients — the CLI has a quirk: `go run main.go client ...`
   silently prints usage. Build a binary named exactly crm-middleware:
     CGO_ENABLED=0 go build -tags dev \
       -ldflags "-X 'crm-middleware/build.serviceName=crm-middleware'" \
       -o /tmp/cli/crm-middleware .
     /tmp/cli/crm-middleware client create <name>   # run from repo root
   (`make dev` deletes bin/, so build the CLI elsewhere.)

FIX RULES:
- Fix the root cause you evidenced, one change at a time. Do not call
  GET /config/versions manually while testing — it burns the client.
- After any change to /config output (name, auth_method, icon), restart
  the service AND use a fresh/reset client before retrying in PBXware.

VERIFY:
- Run ./activation-demo.sh (repo root) with a FRESH client:
    BASE=https://<host>/custom ID=<id> KEY=<key> ./activation-demo.sh
  All five steps must succeed end-to-end through the PUBLIC URL, not
  localhost. Then repeat the real "Add Custom Integration" in PBXware
  with another fresh client and confirm it lands on the integration's
  Configuration page and appears under Active Integrations.
```

Tips:

- If the middleware runs without ngrok, the same request trail exists in
  `.crm-middleware/app.log` or your reverse proxy's access log — the
  inspector is a convenience, not a requirement.
- Append your specific symptom to the prompt (even just "same generic
  error on Add Custom Integration") — the procedure converges faster
  with a starting point.
