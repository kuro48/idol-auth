#!/usr/bin/env bash
set -euo pipefail

APP_URL="${APP_URL:-http://localhost:3002}"
AUTH_URL="${AUTH_URL:-http://localhost:8080}"
KRATOS_BROWSER_URL="${KRATOS_BROWSER_URL:-http://localhost:4433}"
MAILPIT_URL="${MAILPIT_URL:-http://localhost:8025}"
ADMIN_TOKEN="${ADMIN_TOKEN:-dev-bootstrap-token-0123456789abcdef0123456789abcdef}"
WORKDIR="$(mktemp -d)"
COOKIE_JAR="$WORKDIR/cookies.txt"
PASSWORD="${PASSWORD:-CorrectHorseBatteryStaple123!}"
EMAIL="${EMAIL:-smoke.$(date +%s)@example.com}"

cleanup() {
  rm -rf "$WORKDIR"
}
trap cleanup EXIT

extract_attr() {
  local file="$1"
  local name="$2"
  sed -n "s/.*name=\"$name\" value=\"\\([^\"]*\\)\".*/\\1/p" "$file" | sed 's/&#43;/+/g'
}

extract_form_action() {
  local file="$1"
  sed -n 's/.*<form action="\([^"]*\)" method=.*/\1/p' "$file"
}

generate_totp_values() {
  local json_file="$1"
  python3 - "$json_file" <<'PY'
import base64
import hashlib
import hmac
import json
import struct
import sys
import time

with open(sys.argv[1], "r", encoding="utf-8") as fh:
    flow = json.load(fh)

action = flow["ui"]["action"]
csrf = ""
secret = ""
for node in flow["ui"]["nodes"]:
    attrs = node["attributes"]
    if attrs.get("name") == "csrf_token":
        csrf = attrs.get("value", "")
    if attrs.get("id") == "totp_secret_key":
        secret = attrs["text"]["context"]["secret"]

if not action or not csrf or not secret:
    raise SystemExit("missing TOTP settings flow fields")

key = base64.b32decode(secret, casefold=True)
counter = struct.pack(">Q", int(time.time()) // 30)
digest = hmac.new(key, counter, hashlib.sha1).digest()
offset = digest[-1] & 0x0F
code = (struct.unpack(">I", digest[offset:offset + 4])[0] & 0x7FFFFFFF) % 1000000

print(action)
print(csrf)
print(secret)
print(f"{code:06d}")
PY
}

extract_json_value() {
  local json="$1"
  local expr="$2"
  python3 - "$json" "$expr" <<'PY'
import json
import sys

payload = json.loads(sys.argv[1])
expr = sys.argv[2]

if expr == "session_identity_id":
    print(payload.get("identity_id", ""))
elif expr == "search_first_identity_id":
    items = payload.get("items", [])
    print(items[0].get("id", "") if items else "")
else:
    raise SystemExit(f"unsupported expr: {expr}")
PY
}

echo "==> Registering $EMAIL"
curl -sS -c "$COOKIE_JAR" -b "$COOKIE_JAR" -L "$APP_URL/registration" >"$WORKDIR/registration-step1.html"
ACTION="$(extract_form_action "$WORKDIR/registration-step1.html")"
CSRF="$(extract_attr "$WORKDIR/registration-step1.html" "csrf_token")"

curl -sS -D "$WORKDIR/registration-step1.headers" -o /dev/null -c "$COOKIE_JAR" -b "$COOKIE_JAR" -X POST "$ACTION" \
  --data-urlencode "csrf_token=$CSRF" \
  --data-urlencode "traits.primary_identifier_type=email" \
  --data-urlencode "traits.email=$EMAIL" \
  --data-urlencode "method=profile"

STEP2_URL="$(awk '/^Location:/ {print $2}' "$WORKDIR/registration-step1.headers" | tr -d '\r')"
curl -sS -c "$COOKIE_JAR" -b "$COOKIE_JAR" "$STEP2_URL" >"$WORKDIR/registration-step2.html"
ACTION="$(extract_form_action "$WORKDIR/registration-step2.html")"
CSRF="$(extract_attr "$WORKDIR/registration-step2.html" "csrf_token")"

curl -sS -o /dev/null -c "$COOKIE_JAR" -b "$COOKIE_JAR" -X POST "$ACTION" \
  --data-urlencode "csrf_token=$CSRF" \
  --data-urlencode "traits.primary_identifier_type=email" \
  --data-urlencode "traits.email=$EMAIL" \
  --data-urlencode "traits.phone=" \
  --data-urlencode "password=$PASSWORD" \
  --data-urlencode "method=password"

echo "==> Checking password-only session"
SESSION_JSON="$(curl -sS -c "$COOKIE_JAR" -b "$COOKIE_JAR" "$AUTH_URL/v1/auth/session")"
printf '%s\n' "$SESSION_JSON"
printf '%s' "$SESSION_JSON" | grep -q '"authenticated":true'
printf '%s' "$SESSION_JSON" | grep -q '"methods":\["password"\]'
IDENTITY_ID="$(extract_json_value "$SESSION_JSON" session_identity_id)"
if [[ -z "$IDENTITY_ID" ]]; then
  echo "failed to extract identity id from session payload" >&2
  exit 1
fi

echo "==> Verifying OAuth is blocked until MFA is enrolled"
curl -sS -L -c "$COOKIE_JAR" -b "$COOKIE_JAR" "$APP_URL/oauth/start" >"$WORKDIR/pre-mfa-oauth.html"
grep -q '<title>Settings</title>' "$WORKDIR/pre-mfa-oauth.html"

echo "==> Fetching TOTP settings flow"
curl -sS -D "$WORKDIR/settings.headers" -o /dev/null -c "$COOKIE_JAR" -b "$COOKIE_JAR" "$KRATOS_BROWSER_URL/self-service/settings/browser"
SETTINGS_FLOW_URL="$(awk '/^Location:/ {print $2}' "$WORKDIR/settings.headers" | tr -d '\r')"
SETTINGS_FLOW_ID="${SETTINGS_FLOW_URL##*flow=}"
curl -sS -H 'Accept: application/json' -c "$COOKIE_JAR" -b "$COOKIE_JAR" \
  "$KRATOS_BROWSER_URL/self-service/settings/flows?id=$SETTINGS_FLOW_ID" >"$WORKDIR/settings.json"

readarray_output="$(generate_totp_values "$WORKDIR/settings.json")"
SETTINGS_ACTION="$(printf '%s\n' "$readarray_output" | sed -n '1p')"
SETTINGS_CSRF="$(printf '%s\n' "$readarray_output" | sed -n '2p')"
TOTP_SECRET="$(printf '%s\n' "$readarray_output" | sed -n '3p')"
TOTP_CODE="$(printf '%s\n' "$readarray_output" | sed -n '4p')"

echo "==> Enrolling TOTP"
curl -sS -o /dev/null -c "$COOKIE_JAR" -b "$COOKIE_JAR" -X POST "$SETTINGS_ACTION" \
  --data-urlencode "csrf_token=$SETTINGS_CSRF" \
  --data-urlencode "totp_code=$TOTP_CODE" \
  --data-urlencode "method=totp"

echo "==> Checking AAL2 session after TOTP enrollment"
WHOAMI_JSON="$(curl -sS -b "$COOKIE_JAR" "$KRATOS_BROWSER_URL/sessions/whoami")"
printf '%s\n' "$WHOAMI_JSON"
printf '%s' "$WHOAMI_JSON" | grep -q '"authenticator_assurance_level":"aal2"'
printf '%s' "$WHOAMI_JSON" | grep -q '"method":"totp"'

echo "==> First-party OIDC login"
curl -sS -L -c "$COOKIE_JAR" -b "$COOKIE_JAR" "$APP_URL/oauth/start" >"$WORKDIR/oidc-first-party.html"
grep -q 'OIDC コールバックが完了しました' "$WORKDIR/oidc-first-party.html"

echo "==> Partner OIDC login"
curl -sS -L -c "$COOKIE_JAR" -b "$COOKIE_JAR" "$APP_URL/oauth/start?app=partner" >"$WORKDIR/oidc-partner-consent.html"
grep -q 'Idol Partner Demo Client' "$WORKDIR/oidc-partner-consent.html"
CHALLENGE="$(extract_attr "$WORKDIR/oidc-partner-consent.html" "consent_challenge")"
CSRF="$(extract_attr "$WORKDIR/oidc-partner-consent.html" "csrf_token")"

curl -sS -L -c "$COOKIE_JAR" -b "$COOKIE_JAR" -X POST "$AUTH_URL/v1/auth/consent" \
  --data-urlencode "consent_challenge=$CHALLENGE" \
  --data-urlencode "csrf_token=$CSRF" \
  --data-urlencode "action=accept" >"$WORKDIR/oidc-partner-result.html"
grep -q 'OIDC コールバックが完了しました' "$WORKDIR/oidc-partner-result.html"

echo "==> Mailpit status"
curl -sS "$MAILPIT_URL/api/v1/messages"

echo "==> Admin search / disable / enable"
SEARCH_JSON="$(curl -sS -H "Authorization: Bearer $ADMIN_TOKEN" "$AUTH_URL/v1/admin/users?identifier=$EMAIL")"
printf '%s\n' "$SEARCH_JSON"
SEARCH_IDENTITY_ID="$(extract_json_value "$SEARCH_JSON" search_first_identity_id)"
if [[ "$SEARCH_IDENTITY_ID" != "$IDENTITY_ID" ]]; then
  echo "unexpected identity id from admin search: $SEARCH_IDENTITY_ID" >&2
  exit 1
fi

DISABLE_JSON="$(curl -sS -H "Authorization: Bearer $ADMIN_TOKEN" -H "Content-Type: application/json" -X PATCH "$AUTH_URL/v1/admin/users/$IDENTITY_ID" -d '{"state":"inactive"}')"
printf '%s\n' "$DISABLE_JSON"
printf '%s' "$DISABLE_JSON" | grep -q '"state":"inactive"'

ENABLE_JSON="$(curl -sS -H "Authorization: Bearer $ADMIN_TOKEN" -H "Content-Type: application/json" -X PATCH "$AUTH_URL/v1/admin/users/$IDENTITY_ID" -d '{"state":"active"}')"
printf '%s\n' "$ENABLE_JSON"
printf '%s' "$ENABLE_JSON" | grep -q '"state":"active"'

echo "==> Admin revoke sessions"
curl -sS -o /dev/null -H "Authorization: Bearer $ADMIN_TOKEN" -X POST "$AUTH_URL/v1/admin/users/$IDENTITY_ID/revoke-sessions"
REVOKED_SESSION_JSON="$(curl -sS -c "$COOKIE_JAR" -b "$COOKIE_JAR" "$AUTH_URL/v1/auth/session")"
printf '%s\n' "$REVOKED_SESSION_JSON"
printf '%s' "$REVOKED_SESSION_JSON" | grep -q '"authenticated":false'

echo "==> Admin audit logs"
AUDIT_JSON="$(curl -sS -H "Authorization: Bearer $ADMIN_TOKEN" "$AUTH_URL/v1/admin/audit-logs?target_id=$IDENTITY_ID&limit=20")"
printf '%s\n' "$AUDIT_JSON"
printf '%s' "$AUDIT_JSON" | grep -q '"identity.disabled"'
printf '%s' "$AUDIT_JSON" | grep -q '"identity.enabled"'
printf '%s' "$AUDIT_JSON" | grep -q '"identity.sessions.revoked"'

echo "Smoke test passed for $EMAIL"
