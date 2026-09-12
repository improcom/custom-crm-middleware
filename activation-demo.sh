#!/bin/sh
# Replays the exact request sequence PBXware performs when a Custom CRM
# integration is added and activated (observed live against a PBXware 7
# system on 2026-09-12).
#
# Usage:
#   BASE=https://<middleware-host>/custom ID=<client-id> KEY=<api-key> ./activation-demo.sh
#
# NOTE: step 5 (config/versions) is ONE-SHOT — it flips the client to
# ACTIVE, after which /config answers 409. To rerun the demo, create a
# fresh client:  crm-middleware client create <name>
set -eu

: "${BASE:?set BASE, e.g. https://host/custom}"
: "${ID:?set ID (Client ID)}"
: "${KEY:?set KEY (API key / client secret)}"

j() { python3 -m json.tool 2>/dev/null || cat; }

echo "# 1. Exchange Basic credentials for a JWT."
echo "#    PBXware also sends X-Integration-ID: <its internal integration id>."
curl -sf -u "$ID:$KEY" -H "X-Integration-ID: demo" "$BASE/api/v1/token" | j

TOKEN=$(curl -sf -u "$ID:$KEY" "$BASE/api/v1/token" \
    | python3 -c "import sys,json;print(json.load(sys.stdin)['token'])")
AUTH="Authorization: Bearer $TOKEN"

echo
echo "# 2. Integration metadata: display name, auth method, 240x240 webp icon."
echo "#    Only answers while the client is pending; 409 once activated."
curl -sf -H "$AUTH" "$BASE/api/v1/config" | j

echo
echo "# 3. Supported object types."
curl -sf -H "$AUTH" "$BASE/api/v1/config/objects" | j

echo
echo "# 4. Field schema per object type (PBXware's Data Models tab)."
curl -sf -H "$AUTH" "$BASE/api/v1/object/describe/Contact" | j

echo
echo "# 5. Supported API versions -- THE ACTIVATION CALL (one-shot)."
curl -sf -H "$AUTH" "$BASE/api/v1/config/versions" | j

echo
echo "# Activation complete. PBXware then re-fetches token + versions to verify."
