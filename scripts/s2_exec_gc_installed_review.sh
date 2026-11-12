#!/bin/sh
# Single-use GitHub-hosted test ONLY. Stage an exact candidate into otherwise
# nonexistent canonical install paths, clone the stopped original ledger,
# and verify the actual privileged root-peer recovery CLI against real Incus.
# This is NOT production installation, daemon activation, or native mutation.
set -eu

if [ "$GITHUB_ACTIONS" != true ] || [ "$RUNNER_ENVIRONMENT" != github-hosted ] ||
   [ "$(id -u)" -eq 0 ] || ! sudo -n true || [ "$#" -ne 7 ]; then
  echo "S2_INSTALLED_GC_REVIEW_BLOCKED: disposable nonroot hosted CI only" >&2
  exit 91
fi
work="$1"
source_db="$2"
project="$3"
socket="$4"
operation="$5"
native="$6"
uuid="$7"
case "$work" in
  /tmp/bw-v1-s2-ghci.*) ;;
  *) echo "S2_INSTALLED_GC_REVIEW_BLOCKED: outside disposable directory" >&2; exit 92 ;;
esac
test "$source_db" = "$work/mcp-quota-ledger/ledger.sqlite"
test "$socket" = "/var/lib/incus/unix.socket.user"
test "$project" = "user-$(id -u boxweave)"
printf '%s\n' "$operation" | grep -Eq '^[0-9a-f]{32}$'
printf '%s\n' "$native" | grep -Eq '^[0-9a-f-]{36}$'
printf '%s\n' "$uuid" | grep -Eq '^[0-9a-f-]{36}$'
test -f "$work/boxweave"
# This state directory is intentionally private to the nonroot service user.
# A CI runner has no permission even to stat it; only verified local root may.
sudo test -f "$source_db"
echo "S2_GC_INSTALLED_DISPOSABLE_CANONICAL_PATH_PRECHECK=PASS"
if [ -e /var/lib/boxweave ] || [ -e /usr/local/bin/boxweave ] ||
   [ -e /etc/systemd/system/boxweave.service ]; then
  echo "S2_INSTALLED_GC_REVIEW_BLOCKED: existing install identity" >&2
  exit 93
fi
digest="$(sha256sum "$work/boxweave" | cut -d' ' -f1)"
sudo "$work/boxweave" install-stage -source-binary "$work/boxweave" \
  -expect-sha256 "$digest" -project "$project" -incus-socket "$socket" \
  -service-user boxweave -approve-stage > "$work/gc-installed-stage.json"
sudo /usr/local/bin/boxweave install-status -project "$project" \
  -incus-socket "$socket" -service-user boxweave > "$work/gc-installed-status.json"
python3 - "$work/gc-installed-status.json" "$digest" <<'PY'
import json,sys
v=json.load(open(sys.argv[1]))
assert v["binary_sha256"] == sys.argv[2]
assert v["service_active"] is False
print("S2_GC_EXACT_CANDIDATE_STAGED_INSTALL_MANIFEST_VERIFIED=PASS")
PY
# No concurrent controller is running: only a read-only SQLite online backup
# copies the original REVIEW_REQUIRED receipt to the new canonical test DB.
# The root-reviewed release changes only this installed clone; it does not
# dispatch a second native DELETE or reset the independent original ledger.
sudo python3 - "$source_db" /var/lib/boxweave/ledger.sqlite "$operation" "$native" "$uuid" <<'PY'
import sqlite3,sys
src,dest,op,native,uuid=sys.argv[1:]
with sqlite3.connect(f"file:{src}?mode=ro",uri=True) as original:
    row=original.execute(
        "SELECT j.state,j.native_operation_id,j.box_uuid FROM exec_output_gc j "
        "WHERE j.operation_id=? AND j.stream='stdout'",(op,)).fetchone()
    assert row == ("REVIEW_REQUIRED",native,uuid),row
    with sqlite3.connect(dest) as cloned:
        original.backup(cloned)
PY
sudo chown boxweave:incus /var/lib/boxweave/ledger.sqlite
sudo chmod 0600 /var/lib/boxweave/ledger.sqlite

recover_gc() {
  reviewer="$1"
  shift
  sudo /usr/local/bin/boxweave recover \
    -db /var/lib/boxweave/ledger.sqlite -project "$project" \
    -incus-socket "$socket" -service-user boxweave \
    -operation "$operation" -stream stdout \
    -reviewer "$reviewer" -approve-local "$@"
}
# Privilege and consent are necessary even if an actor knows all original IDs.
if sudo runuser -u boxweave -- /usr/local/bin/boxweave recover \
    -db /var/lib/boxweave/ledger.sqlite -project "$project" \
    -incus-socket "$socket" -service-user boxweave \
    -operation "$operation" -stream stdout -reviewer ci-reviewer \
    -action gc-inspect -approve-local > "$work/gc-nonroot-denied.txt" 2>&1; then
  echo "S2_GC_FAIL: unprivileged host recovery accepted" >&2
  exit 94
fi
if sudo /usr/local/bin/boxweave recover \
    -db /var/lib/boxweave/ledger.sqlite -project "$project" \
    -incus-socket "$socket" -service-user boxweave \
    -operation "$operation" -stream stdout -reviewer ci-reviewer \
    -action gc-inspect > "$work/gc-no-approval-denied.txt" 2>&1; then
  echo "S2_GC_FAIL: missing explicit local approval accepted" >&2
  exit 95
fi
recover_gc ci-reviewer -action gc-inspect > "$work/gc-installed-inspect.json"
python3 - "$work/gc-installed-inspect.json" "$operation" "$native" "$uuid" "$project" <<'PY'
import json,sys
v=json.load(open(sys.argv[1]))
r=v["Original"]
assert r["OperationID"] == sys.argv[2]
assert r["NativeOperationID"] == sys.argv[3]
assert r["BoxUUID"] == sys.argv[4]
assert r["Project"] == sys.argv[5]
assert r["State"] == "REVIEW_REQUIRED"
assert v["NativeObservation"] == "NATIVE_MISSING"
print("S2_GC_INSTALLED_ROOT_CLI_EXACT_NATIVE_REOBSERVED=PASS")
PY
recover_gc ci-reviewer -action gc-challenge > "$work/gc-installed-challenge.secret.json"
challenge="$(python3 - "$work/gc-installed-challenge.secret.json" <<'PY'
import json,re,sys
v=json.load(open(sys.argv[1]))
challenge=v["Challenge"]
assert re.fullmatch("[0-9a-f]{64}",challenge)
assert v["ExpiresInSeconds"] == 300
print(challenge)
PY
)"
reason="Original disposable Incus stdout DELETE succeeded, but its native ACK was deliberately lost; never retry."
if recover_gc wrong-reviewer -action gc-release -challenge "$challenge" \
    -confirm-uuid "$uuid" -reason "$reason" -acknowledge-uncertain-native \
    > "$work/gc-wrong-reviewer-denied.txt" 2>&1; then
  echo "S2_GC_FAIL: foreign reviewer reused challenge" >&2
  exit 96
fi
if recover_gc ci-reviewer -action gc-release -challenge "$challenge" \
    -confirm-uuid "not-the-original-uuid" -reason "$reason" \
    -acknowledge-uncertain-native > "$work/gc-wrong-uuid-denied.txt" 2>&1; then
  echo "S2_GC_FAIL: wrong original UUID released claim" >&2
  exit 97
fi
recover_gc ci-reviewer -action gc-release -challenge "$challenge" \
    -confirm-uuid "$uuid" -reason "$reason" -acknowledge-uncertain-native \
    > "$work/gc-installed-released.json"
python3 - "$work/gc-installed-released.json" <<'PY'
import json,sys
v=json.load(open(sys.argv[1]))
assert v["State"] == "REVIEWED_RELEASED"
assert v["ClaimReleased"] is True
assert v["NativeRetried"] is False and v["ExpiryInferred"] is False
print("S2_GC_INSTALLED_ROOT_CLI_REVIEWED_RELEASE_NEVER_RETRIED_DELETE=PASS")
PY
if recover_gc ci-reviewer -action gc-release -challenge "$challenge" \
    -confirm-uuid "$uuid" -reason "$reason" -acknowledge-uncertain-native \
    > "$work/gc-replay-denied.txt" 2>&1; then
  echo "S2_GC_FAIL: replay of consumed review challenge accepted" >&2
  exit 98
fi
sudo python3 - /var/lib/boxweave/ledger.sqlite "$operation" <<'PY'
import sqlite3,sys
with sqlite3.connect(f"file:{sys.argv[1]}?mode=ro",uri=True) as db:
    state=db.execute("SELECT state FROM exec_output_gc WHERE operation_id=? AND stream='stdout'",(sys.argv[2],)).fetchone()
    assert state == ("REVIEWED_RELEASED",),state
    events=[x[0] for x in db.execute("SELECT event FROM exec_output_gc_audit WHERE operation_id=? AND stream='stdout' ORDER BY seq",(sys.argv[2],))]
    for e in ("BEFORE_DELETE","REVIEW_REQUIRED","CHALLENGE_ISSUED","REVIEWED_RELEASED"):
        assert events.count(e)==1,(e,events)
print("S2_GC_INSTALLED_ROOT_CLI_DURABLE_AUDIT_NO_REPLAY=PASS")
PY
echo "S2_GC_INSTALLED_STAGED_CANDIDATE_ROOT_RECOVER_CYCLE=PASS"
