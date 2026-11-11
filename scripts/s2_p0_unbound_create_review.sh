#!/bin/sh
# Safety negative for a REAL native Create with its HTTP 202 ACK lost:
# original SQLite UNKNOWN has no owned Box UUID, and local root MUST refuse
# to unlock/adopt that unknown create. This cannot be silently fixed by
# guessing the later native UUID. Disposable CI-only staged installed clone.
set -eu
if [ "$GITHUB_ACTIONS" != true ] || [ "$RUNNER_ENVIRONMENT" != github-hosted ] ||
  [ "$(id -u)" -eq 0 ] || [ "$#" -ne 3 ]; then exit 91; fi
work="$1"; project="$2"; socket="$3"
case "$work" in /tmp/bw-v1-s2-ghci.*) ;; *) exit 92;; esac
test "$project" = "user-$(id -u boxweave)"
test "$socket" = /var/lib/incus/unix.socket.user
test -z "$(sudo incus list --all-projects --format csv -c n)"
sudo test -f "$work/ledger/ack.sqlite"
sudo test -f /usr/local/bin/boxweave
sudo test -f /var/lib/boxweave/install-manifest.json
original="$(sudo python3 - "$work/ack-original.json" "$work/ledger/ack.sqlite" "$project" <<'PY'
import json,sqlite3,sys
checkpoint,db,project=sys.argv[1:]
v=json.load(open(checkpoint))
op=v["original_id"]
assert len(op)==32
with sqlite3.connect(f"file:{db}?mode=ro",uri=True) as s:
    row=s.execute("SELECT state,action,box_uuid,incarnation,resource_key,layer,safe_intent,native_operation_id FROM operations WHERE id=?",(op,)).fetchone()
    assert row and row[:6]==("UNKNOWN","box_create","","","box:bw-s2-ack-native-110d",project),row[:6] if row else row
    assert json.loads(row[6])["name"]=="bw-s2-ack-native-110d"
    assert row[7]==""
    assert s.execute("SELECT COUNT(*) FROM recovery_audit WHERE operation_id=?",(op,)).fetchone()[0]==0
print(op)
PY
)"
# The accepted native Create has already been independently observed as an
# original UUID and cleaned ONLY by the earlier one-use CI administrator.
# Clone original UNKNOWN controller receipt, never mutate its source SQLite.
sudo python3 - "$work/ledger/ack.sqlite" /var/lib/boxweave/ledger.sqlite "$original" <<'PY'
import sqlite3,sys
src,dst,op=sys.argv[1:]
with sqlite3.connect(f"file:{src}?mode=ro",uri=True) as original:
    assert original.execute("PRAGMA integrity_check").fetchone()[0]=="ok"
    assert original.execute("SELECT state FROM operations WHERE id=?",(op,)).fetchone()==("UNKNOWN",)
    with sqlite3.connect(dst) as cloned:
        original.backup(cloned)
PY
sudo chown boxweave:incus /var/lib/boxweave/ledger.sqlite
sudo chmod 0600 /var/lib/boxweave/ledger.sqlite
recover_create() {
  sudo /usr/local/bin/boxweave recover \
    -db /var/lib/boxweave/ledger.sqlite -incus-socket "$socket" \
    -project "$project" -service-user boxweave \
    -operation "$original" -reviewer ci-unbound-create -approve-local "$@"
}
if recover_create -action inspect > "$work/p0-unbound-create-inspect.denied" 2>&1; then
  echo "S2_P0_FAIL: unbound native Create UNKNOWN was accepted for manual review" >&2
  exit 93
fi
if recover_create -action challenge > "$work/p0-unbound-create-challenge.denied" 2>&1; then
  echo "S2_P0_FAIL: root issued adoption challenge without an owned UUID" >&2
  exit 94
fi
if recover_create -action release -challenge 0000000000000000000000000000000000000000000000000000000000000000 \
  -confirm-uuid 00000000-0000-0000-0000-000000000000 \
  -reason "Unknown native Create lacks an original verified UUID; never release." \
  -acknowledge-uncertain-native > "$work/p0-unbound-create-release.denied" 2>&1; then
  echo "S2_P0_FAIL: unknown Create was released/adopted without identity proof" >&2
  exit 95
fi
sudo python3 - /var/lib/boxweave/ledger.sqlite "$work/ledger/ack.sqlite" "$original" <<'PY'
import sqlite3,sys
clone,original,op=sys.argv[1:]
for path in (clone,original):
    with sqlite3.connect(f"file:{path}?mode=ro",uri=True) as db:
        assert db.execute("PRAGMA integrity_check").fetchone()[0]=="ok"
        assert db.execute("SELECT state,box_uuid,native_operation_id FROM operations WHERE id=?",(op,)).fetchone()==("UNKNOWN","","")
        assert db.execute("SELECT COUNT(*) FROM recovery_audit WHERE operation_id=?",(op,)).fetchone()[0]==0
print("S2_P0_UNBOUND_NATIVE_CREATE_ORIGINAL_IMMUTABLE_NO_ADOPTION=PASS")
PY
test -z "$(sudo incus list --all-projects --format csv -c n)"
echo S2_P0_REAL_LOST_ACK_CREATE_WITHOUT_UUID_LOCAL_RECOVERY_REFUSED=PASS
