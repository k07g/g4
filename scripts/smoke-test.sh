#!/usr/bin/env bash
# Exercises signup -> confirm -> signin -> signout -> forgot/reset password
# -> delete against a locally running server. Requires AUTH_PROVIDER=memory
# (the fixed confirmation/reset code below only works with the in-memory
# provider) and `jq` installed.
set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8080}"
EMAIL="smoke-test-$(date +%s)@example.com"
PASSWORD="Passw0rd!123"
NEW_PASSWORD="NewPassw0rd!456"

echo "== signup ($EMAIL) =="
curl -sS -X POST "$BASE_URL/auth/signup" \
  -H 'Content-Type: application/json' \
  -d "{\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\"}" | tee /tmp/g4-signup.json
echo

echo "== confirm =="
curl -sS -X POST "$BASE_URL/auth/confirm" \
  -H 'Content-Type: application/json' \
  -d "{\"email\":\"$EMAIL\",\"code\":\"000000\"}" -w '\nstatus=%{http_code}\n'

echo "== signin =="
SIGNIN_RESPONSE=$(curl -sS -X POST "$BASE_URL/auth/signin" \
  -H 'Content-Type: application/json' \
  -d "{\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\"}")
echo "$SIGNIN_RESPONSE"
ACCESS_TOKEN=$(echo "$SIGNIN_RESPONSE" | jq -r '.access_token')

echo "== signout =="
curl -sS -X POST "$BASE_URL/auth/signout" \
  -H "Authorization: Bearer $ACCESS_TOKEN" -w '\nstatus=%{http_code}\n'

echo "== forgot password =="
curl -sS -X POST "$BASE_URL/auth/forgot-password" \
  -H 'Content-Type: application/json' \
  -d "{\"email\":\"$EMAIL\"}" -w '\nstatus=%{http_code}\n'

echo "== reset password =="
curl -sS -X POST "$BASE_URL/auth/reset-password" \
  -H 'Content-Type: application/json' \
  -d "{\"email\":\"$EMAIL\",\"code\":\"000000\",\"new_password\":\"$NEW_PASSWORD\"}" -w '\nstatus=%{http_code}\n'

echo "== signin with the new password (to get a fresh token for delete) =="
SIGNIN_RESPONSE=$(curl -sS -X POST "$BASE_URL/auth/signin" \
  -H 'Content-Type: application/json' \
  -d "{\"email\":\"$EMAIL\",\"password\":\"$NEW_PASSWORD\"}")
ACCESS_TOKEN=$(echo "$SIGNIN_RESPONSE" | jq -r '.access_token')

echo "== delete account =="
curl -sS -X DELETE "$BASE_URL/auth/me" \
  -H "Authorization: Bearer $ACCESS_TOKEN" -w '\nstatus=%{http_code}\n'

echo "done"
