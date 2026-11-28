#!/usr/bin/env bash
set -euo pipefail

: "${OC_URL:?Set OC_URL}"
: "${OC_ADMIN_SESSION:?Set OC_ADMIN_SESSION}"
: "${OC_READER_SESSION:?Set OC_READER_SESSION}"
: "${OC_SERVICE_TOKEN:?Set OC_SERVICE_TOKEN}"
: "${OC_READER_ID:?Set OC_READER_ID}"

smoke_dir=$(mktemp -d)
trap 'rm -rf "$smoke_dir"' EXIT
smoke_scope="smoke_$(date +%s)_${RANDOM}"
response_file="$smoke_dir/response.json"

request() {
    local expected=$1 method=$2 path=$3 body=$4 identity=$5
    shift 5
    local -a args=(--silent --show-error --connect-timeout 2 --max-time 5 --request "$method" --output "$response_file" --write-out '%{http_code}' --header 'Content-Type: application/json' --header 'X-Role: admin' --header 'X-User-Id: 1')
    case "$identity" in
        admin) args+=(--cookie "session_id=$OC_ADMIN_SESSION") ;;
        reader) args+=(--cookie "session_id=$OC_READER_SESSION") ;;
        service) args+=(--header "Authorization: Bearer $OC_SERVICE_TOKEN") ;;
    esac
    if [[ -n "$body" ]]; then args+=(--data "$body"); fi
    local actual
    actual=$(curl "${args[@]}" "$@" "$OC_URL$path")
    if [[ "$actual" != "$expected" ]]; then
        printf 'FAIL %s %s: HTTP %s, expected %s\n' "$method" "$path" "$actual" "$expected" >&2
        cat "$response_file" >&2
        return 1
    fi
    printf '%s %s -> %s\n' "$method" "$path" "$actual"
}

field() {
    python3 - "$response_file" "$1" <<'PY'
import json,sys
with open(sys.argv[1]) as stream: value=json.load(stream)
for part in sys.argv[2].split('.'):
    value=value[int(part)] if isinstance(value,list) else value[part]
print(value if isinstance(value,str) else json.dumps(value,ensure_ascii=False))
PY
}

expect_field() {
    local actual
    actual=$(field "$1")
    if [[ "$actual" != "$2" ]]; then printf 'FAIL field %s: %s, expected %s\n' "$1" "$actual" "$2" >&2; return 1; fi
}

wait_for_flag_snapshot() {
    request 200 GET "/api/onlineconf/admin/audit?resource_type=flag&resource_id=$flag_id&limit=1" '' admin
    local target current
    target=$(field items.0.revision)
    for ((attempt=0; attempt<150; attempt++)); do
        curl --silent --show-error --fail --max-time 5 --header "Authorization: Bearer $OC_SERVICE_TOKEN" "$OC_URL/internal/onlineconf/snapshot" --output "$response_file"
        current=$(field revision)
        if ((current >= target)); then return 0; fi
        sleep 0.1
    done
    printf 'FAIL snapshot did not reach revision %s\n' "$target" >&2
    return 1
}

update_flag() {
    local enabled=$1 percent=$2 force_on=$3 force_off=$4
    local body
    body=$(printf '{"scope":"%s","key":"test_flag","title":"Flag","description":"curl smoke","enabled":%s,"rollout_percent":%s,"force_on_user_ids":%s,"force_off_user_ids":%s,"version":%s}' "$smoke_scope" "$enabled" "$percent" "$force_on" "$force_off" "$flag_version")
    request 200 PUT "/api/onlineconf/admin/flags/$flag_id" "$body" admin
    flag_version=$(field version)
    wait_for_flag_snapshot
}

check_decision() {
    request 200 POST "/api/onlineconf/admin/flags/$flag_id/evaluate" "{\"user_id\":$OC_READER_ID}" admin
    expect_field enabled "$1"
    expect_field reason "$2"
}

request 200 GET /healthz '' none
request 401 GET /api/onlineconf/admin/me '' none
request 403 GET /api/onlineconf/admin/me '' reader
request 200 GET /api/onlineconf/admin/me '' admin
expect_field role admin
request 401 GET /internal/onlineconf/snapshot '' admin
request 200 GET /internal/onlineconf/snapshot '' service

scope_body=$(printf '{"name":"%s","title":"Curl smoke"}' "$smoke_scope")
request 403 POST /api/onlineconf/admin/scopes "$scope_body" admin --header 'Origin: https://evil.example' --header 'Sec-Fetch-Site: cross-site'
request 201 POST /api/onlineconf/admin/scopes "$scope_body" admin
config_body=$(printf '{"scope":"%s","key":"button_color","title":"Button color","value":"blue"}' "$smoke_scope")
request 201 POST /api/onlineconf/admin/configs "$config_body" admin
config_id=$(field id)
expect_field value blue
request 409 POST /api/onlineconf/admin/configs "$config_body" admin
expect_field code duplicate_key
update_body=$(printf '{"scope":"%s","key":"button_color","title":"Button color","description":"curl smoke","value":"green","version":1}' "$smoke_scope")
request 200 PUT "/api/onlineconf/admin/configs/$config_id" "$update_body" admin
expect_field value green
request 409 PUT "/api/onlineconf/admin/configs/$config_id" "$update_body" admin
expect_field code version_conflict
request 200 GET "/api/onlineconf/admin/configs?scope=$smoke_scope&q=BUTTON_COLOR&limit=1" '' admin
expect_field total 1
expect_field items.0.value green

for kind in object array number boolean null; do
    case "$kind" in
        object) value='{"primary":"blue"}' ;;
        array) value='["blue","green"]' ;;
        number) value='12.5' ;;
        boolean) value='false' ;;
        null) value='null' ;;
    esac
    body=$(printf '{"scope":"%s","key":"value_%s","title":"JSON value","value":%s}' "$smoke_scope" "$kind" "$value")
    request 201 POST /api/onlineconf/admin/configs "$body" admin
 done

flag_body=$(printf '{"scope":"%s","key":"test_flag","title":"Flag","enabled":true,"rollout_percent":0,"force_on_user_ids":[],"force_off_user_ids":[]}' "$smoke_scope")
request 201 POST /api/onlineconf/admin/flags "$flag_body" admin
flag_id=$(field id)
flag_version=$(field version)
if python3 - "$response_file" <<'PY'
import json,sys
with open(sys.argv[1]) as stream: value=json.load(stream)
sys.exit(0 if 'salt' in value else 1)
PY
then printf 'FAIL salt exposed in admin response\n' >&2; exit 1; fi
wait_for_flag_snapshot
check_decision false rollout_miss
update_flag true 100 '[]' '[]'
check_decision true rollout_match
update_flag true 0 "[$OC_READER_ID]" '[]'
check_decision true forced_on
update_flag false 100 "[$OC_READER_ID]" '[]'
check_decision false globally_disabled
update_flag true 100 '[]' "[$OC_READER_ID]"
check_decision false forced_off
conflict_body=$(printf '{"scope":"%s","key":"test_flag","title":"Flag","description":"curl smoke","enabled":true,"rollout_percent":100,"force_on_user_ids":[%s],"force_off_user_ids":[%s],"version":%s}' "$smoke_scope" "$OC_READER_ID" "$OC_READER_ID" "$flag_version")
request 400 PUT "/api/onlineconf/admin/flags/$flag_id" "$conflict_body" admin
request 404 POST "/api/onlineconf/admin/flags/$flag_id/evaluate" '{"user_id":9007199254740991}' admin
request 204 DELETE "/api/onlineconf/admin/flags/$flag_id?version=$flag_version" '' admin
request 200 GET "/api/onlineconf/admin/audit?resource_type=flag&resource_id=$flag_id" '' admin
expect_field items.0.action delete
expect_field items.0.after null
request 204 DELETE "/api/onlineconf/admin/configs/$config_id?version=2" '' admin
request 404 GET "/api/onlineconf/admin/configs/$config_id" '' admin
request 200 GET "/api/onlineconf/admin/audit?resource_type=config&resource_id=$config_id" '' admin
expect_field total 3
printf 'PASS: auth, CSRF, JSON types, CRUD, search, versions, rollout, overrides, snapshot and audit. Scope: %s\n' "$smoke_scope"
