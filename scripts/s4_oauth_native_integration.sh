#!/bin/sh
# S4-A only. The caller has already proved this is a pristine one-use Ubuntu
# GitHub-hosted Incus 7.6 runner, and its installed HTTP MCP has created
# a disposable exact owned Box. No physical L0/L1 service or config changes.
set -eu
test "${GITHUB_ACTIONS:-}" = true
test "${RUNNER_ENVIRONMENT:-}" = github-hosted
test "$(id -u)" -ne 0
: "${S4_A_WORK:?}"
: "${S4_A_DB:?}"
: "${S4_A_PROJECT:?}"
echo S4_A_OAUTH_STAGE=RUNNER_SCOPE_VERIFIED
test -x "$S4_A_WORK/s4-oauth-native-probe"
test -f "$S4_A_WORK/bh018-atomic-start.log"
# SQLite is intentionally private (0700 owned by boxweave) and therefore
# cannot be stat()'d by the CI runner. Check the exact service identity,
# never chmod/chown the durable journal or broaden the runner's permissions.
if ! sudo runuser -u boxweave -- test -f "$S4_A_DB"; then
  echo S4_A_OAUTH_STAGE=RESTRICTED_LEDGER_UNAVAILABLE >&2
  exit 8
fi
echo S4_A_OAUTH_STAGE=RESTRICTED_LEDGER_VISIBLE

box_id="$(sed -n 's/^BH018_MCP_CREATE_BOX_UUID=\([^ ]*\).*/\1/p' "$S4_A_WORK/bh018-atomic-start.log" | head -n 1)"
test "${#box_id}" -eq 36
echo S4_A_OAUTH_STAGE=ORIGINAL_BOX_UUID_RESOLVED
# This digest is not a token or credential; it finds one new original in the
# disposable SQLite WAL BEFORE the first native mutation is dispatched.
key_digest="$(printf %s s4-oauth-native-one-effect-001 | sha256sum | cut -d ' ' -f 1)"
sudo runuser -u boxweave -- env -u INCUS_CONF -u INCUS_CONFIG_DIR \
  -u XDG_CONFIG_HOME HOME=/home/boxweave \
  XDG_CONFIG_HOME=/home/boxweave/.config \
  python3 "$S4_A_WORK/s4-raw-operation-witness.py" \
  --db "$S4_A_DB" --key-digest "$key_digest" --timeout-seconds 120 \
  --events-jsonl "$S4_A_WORK/ledger/s4-native-events.jsonl" \
  > "$S4_A_WORK/s4-oauth-native-raw.json" \
  2> "$S4_A_WORK/s4-oauth-native-raw.redacted.err" &
witness_pid=$!
echo S4_A_OAUTH_STAGE=INDEPENDENT_NATIVE_WITNESS_STARTED

# Credentials and unredacted Go errors are never written to the public log.
# The probe runs AS the restricted installed service user against the exact
# one-use ledger and its loopback official MCP endpoint.
if ! sudo runuser -u boxweave -- env -u INCUS_CONF -u INCUS_CONFIG_DIR \
  -u XDG_CONFIG_HOME HOME=/home/boxweave \
  XDG_CONFIG_HOME=/home/boxweave/.config \
  GITHUB_ACTIONS=true RUNNER_ENVIRONMENT=github-hosted \
  "$S4_A_WORK/s4-oauth-native-probe" -db "$S4_A_DB" \
  -project "$S4_A_PROJECT" -box-id "$box_id" \
  -endpoint http://127.0.0.1:18650/mcp \
  > "$S4_A_WORK/s4-oauth-native.redacted.log" 2>&1; then
  echo S4_A_INSTALLED_OAUTH_NATIVE=FAILED_OR_BLOCKED >&2
  # Only emit fixed, secret-free failure classes, never the raw Go panic,
  # OAuth credential, response body or remote request URL.
  if grep -Fq 'S4_A_INSTALLED_OAUTH_SETUP_FAILED' "$S4_A_WORK/s4-oauth-native.redacted.log"; then
    echo S4_A_OAUTH_FAILURE_CLASS=OWNER_BOOTSTRAP
  elif grep -Fq 'S4_A_INSTALLED_OAUTH_REFRESH_HTTP_FAILED' "$S4_A_WORK/s4-oauth-native.redacted.log"; then
    echo S4_A_OAUTH_FAILURE_CLASS=REFRESH_HTTP
  elif grep -Fq 'S4_A_INSTALLED_OAUTH_OFFICIAL_MCP_CONNECT_FAILED' "$S4_A_WORK/s4-oauth-native.redacted.log"; then
    echo S4_A_OAUTH_FAILURE_CLASS=OFFICIAL_MCP_CONNECT
  elif grep -Fq 'S4_A_INSTALLED_OAUTH_NATIVE_ORIGINAL_NOT_SUCCEEDED' "$S4_A_WORK/s4-oauth-native.redacted.log"; then
    echo S4_A_OAUTH_FAILURE_CLASS=ORIGINAL_NATIVE_EFFECT
  elif grep -Fq 'S4_A_INSTALLED_OAUTH_REFRESH_REDISPATCHED_ORIGINAL' "$S4_A_WORK/s4-oauth-native.redacted.log"; then
    echo S4_A_OAUTH_FAILURE_CLASS=REPLAY_REDIRECTED
  else
    echo S4_A_OAUTH_FAILURE_CLASS=OTHER_REDACTED_PROBE_FAILURE
  fi
  exit 7
fi
grep -Fxq 'S4_A_REAL_INSTALLED_OAUTH_REFRESH_GRANT_REPLAY_ONCE=PASS' "$S4_A_WORK/s4-oauth-native.redacted.log"
grep -Fxq 'S4_A_INDEPENDENT_GUEST_ONE_EFFECT=PASS' "$S4_A_WORK/s4-oauth-native.redacted.log"
grep '^S4_A_' "$S4_A_WORK/s4-oauth-native.redacted.log"

# Native Operation objects can expire. A missing one is UNKNOWN as evidence,
# NEVER a simulated PASS or justification to retry the original command.
wait "$witness_pid" || :
python3 - "$S4_A_WORK/s4-oauth-native-raw.json" <<'S4_OAUTH_RAW_CHECK'
import json
import sys
with open(sys.argv[1]) as handle:
    record = json.load(handle)
original = record.get("original_sqlite") or {}
native = record.get("native_terminal") or {}
if (record.get("observation") == "CAPTURED_TERMINAL"
        and record.get("original_id_matches") is True
        and original.get("native_operation_id") == native.get("native_operation_id")
        and native.get("status_code") in (200, 400, 401)
        and native.get("evidence_source") == "INCUS_OPERATION_EVENT_WEBSOCKET"):
    print("S4_A_OAUTH_RAW_BW_ID=" + original["operation_id"]
          + " NATIVE_ID=" + native["native_operation_id"])
    print("S4_A_OAUTH_INDEPENDENT_RAW_OPERATION_TERMINAL=PASS")
else:
    print("S4_A_OAUTH_INDEPENDENT_RAW_OPERATION_TERMINAL=NOT_CAPTURED")
S4_OAUTH_RAW_CHECK
