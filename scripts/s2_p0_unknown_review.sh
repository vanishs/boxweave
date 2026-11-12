#!/bin/sh
# True installed host-local root->nonroot general UNKNOWN review on a stopped
# ledger backup of ONE verified disposable native Update. Never a remote API.
set -eu
if [ "$GITHUB_ACTIONS" != true ] || [ "$RUNNER_ENVIRONMENT" != github-hosted ] ||
  [ "$(id -u)" -eq 0 ] || [ "$#" -ne 5 ]; then exit 91; fi
work="$1"; lab="$2"; project="$3"; socket="$4"; uuid="$5"
case "$work" in /tmp/bw-v1-s2-ghci.*) ;; *) exit 92;; esac
case "$lab" in "$work"/p0-before|"$work"/p0-after) ;; *) exit 93;; esac
test "$socket" = /var/lib/incus/unix.socket.user
test "$project" = "user-$(id -u boxweave)"
sudo test -f /var/lib/boxweave/install-manifest.json
sudo /usr/local/bin/boxweave install-status -project "$project" \
  -incus-socket "$socket" -service-user boxweave > "$work/p0-installed-status.json"
operation="$(sudo python3 - "$lab/original.json" "$uuid" <<'PY'
import json,sys
v=json.load(open(sys.argv[1]))
assert v["box_id"]==sys.argv[2] and len(v["update_id"])==32
print(v["update_id"])
PY
)"
sudo python3 - "$lab/ledger.sqlite" /var/lib/boxweave/ledger.sqlite "$operation" <<'PY'
import sqlite3,sys
src,dst,op=sys.argv[1:]
with sqlite3.connect(f"file:{src}?mode=ro",uri=True) as old:
    assert old.execute("PRAGMA integrity_check").fetchone()[0]=="ok"
    assert old.execute("SELECT state FROM operations WHERE id=?",(op,)).fetchone()==("UNKNOWN",)
    with sqlite3.connect(dst) as target:
        old.backup(target)
PY
sudo chown boxweave:incus /var/lib/boxweave/ledger.sqlite
sudo chmod 0600 /var/lib/boxweave/ledger.sqlite
recover_p0() {
  sudo /usr/local/bin/boxweave recover \
    -db /var/lib/boxweave/ledger.sqlite -project "$project" \
    -incus-socket "$socket" -service-user boxweave \
    -operation "$operation" -reviewer p0-reviewer -approve-local "$@"
}
if sudo runuser -u boxweave -- /usr/local/bin/boxweave recover \
    -db /var/lib/boxweave/ledger.sqlite -project "$project" \
    -incus-socket "$socket" -service-user boxweave \
    -operation "$operation" -reviewer p0-reviewer \
    -action inspect -approve-local > "$work/p0-nonroot-denied.txt" 2>&1; then exit 94; fi
if sudo /usr/local/bin/boxweave recover \
    -db /var/lib/boxweave/ledger.sqlite -project "$project" \
    -incus-socket "$socket" -service-user boxweave \
    -operation "$operation" -reviewer p0-reviewer \
    -action inspect > "$work/p0-unapproved-denied.txt" 2>&1; then exit 95; fi
recover_p0 -action inspect > "$work/p0-installed-unknown-inspect.json"
python3 - "$work/p0-installed-unknown-inspect.json" "$operation" "$uuid" <<'PY'
import json,sys
v=json.load(open(sys.argv[1]))
r=v["original"]
assert r["operation_id"]==sys.argv[2] and r["box_uuid"]==sys.argv[3]
assert r["state"]=="UNKNOWN" and r["action"]=="box_update"
assert v["native_observation"]["state"]=="PRESENT_UUID_CONFIRMED"
assert len(v["native_observation"]["observation_digest"])==64
print("S2_P0_GENERAL_UNKNOWN_EXACT_NATIVE_REOBSERVED=PASS")
PY
recover_p0 -action challenge > "$work/p0-unknown-challenge.secret.json"
challenge="$(python3 - "$work/p0-unknown-challenge.secret.json" <<'PY'
import json,re,sys
s=json.load(open(sys.argv[1]))["challenge"]
assert re.fullmatch("[0-9a-f]{64}",s)
print(s)
PY
)"
reason="Actual disposable Incus Update accepted before SQLite WAL fsync interruption; manually verified."
if recover_p0 -action release -challenge "$challenge" \
    -confirm-uuid incorrect-uuid -reason "$reason" \
    -acknowledge-uncertain-native > "$work/p0-wrong-uuid-denied.txt" 2>&1; then exit 96; fi
recover_p0 -action release -challenge "$challenge" \
  -confirm-uuid "$uuid" -reason "$reason" -acknowledge-uncertain-native \
  > "$work/p0-general-unknown-released.json"
python3 - "$work/p0-general-unknown-released.json" <<'PY'
import json,sys
v=json.load(open(sys.argv[1]))
assert v["original_state"]=="UNKNOWN" and v["claim_released"] is True
assert v["auto_retry"] is False
print("S2_P0_GENERAL_UNKNOWN_LOCAL_RELEASE_NO_RETRY=PASS")
PY
if recover_p0 -action release -challenge "$challenge" \
    -confirm-uuid "$uuid" -reason "$reason" -acknowledge-uncertain-native \
    > "$work/p0-unknown-replay-denied.txt" 2>&1; then exit 97; fi
sudo python3 - /var/lib/boxweave/ledger.sqlite "$operation" <<'PY'
import sqlite3,sys
with sqlite3.connect(f"file:{sys.argv[1]}?mode=ro",uri=True) as db:
    assert db.execute("SELECT state FROM operations WHERE id=?",(sys.argv[2],)).fetchone()==("UNKNOWN",)
    events=[r[0] for r in db.execute("SELECT decision FROM recovery_audit WHERE operation_id=? ORDER BY seq",(sys.argv[2],))]
    assert events.count("CHALLENGE_ISSUED")==1 and events.count("RELEASED")==1,events
print("S2_P0_GENERAL_UNKNOWN_AUDIT_DURABLE=PASS")
PY
echo S2_P0_GENERAL_UNKNOWN_STAGED_ROOT_CLI_DUAL_CONFIRMATION=PASS
