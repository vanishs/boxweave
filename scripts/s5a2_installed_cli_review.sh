#!/bin/sh
# Only on the pristine hosted S5A2 test runner. Compare the same original
# evidence on an offline clone; never modify a live controller's ledger.
set -eu
test "${GITHUB_ACTIONS:-}" = true
test "${RUNNER_ENVIRONMENT:-}" = github-hosted
test "${S5A2_RECOVERY_NATIVE_ONLY:-}" = true
test "$(id -u)" -ne 0 && sudo -n true
test "$#" -eq 4
work="$1"; db="$2"; project="$3"; socket="$4"
case "$work" in /tmp/bw-v1-s2-ghci.*) ;; *) exit 91;; esac
test "$db" = "$work/ledger/ledger.sqlite"
test "$project" = "user-$(id -u boxweave)"
test "$socket" = /var/lib/incus/unix.socket.user
sudo test -f "$work/ledger/s5a2-cli-prebind.sqlite"
sudo test -f "$work/ledger/s5a2-cli-expected.json"
# The original HTTP Controller has exited. Never adopt another installation.
for path in /var/lib/boxweave /usr/local/bin/boxweave /etc/systemd/system/boxweave.service; do
  if sudo test -e "$path" || sudo test -L "$path"; then exit 92; fi
done
digest="$(sha256sum "$work/boxweave" | cut -d' ' -f1)"
sudo "$work/boxweave" install-stage -source-binary "$work/boxweave" \
  -expect-sha256 "$digest" -project "$project" -incus-socket "$socket" \
  -service-user boxweave -approve-stage > "$work/s5a2-cli-stage.json"
sudo /usr/local/bin/boxweave install-status -project "$project" -incus-socket "$socket" \
  -service-user boxweave > "$work/s5a2-cli-status.json"
python3 - "$work/s5a2-cli-status.json" "$digest" <<'PY'
import json,sys
v=json.load(open(sys.argv[1]))
assert v["binary_sha256"]==sys.argv[2] and not v["service_active"],v
PY
sudo install -o boxweave -g incus -m 0600 "$work/ledger/s5a2-cli-prebind.sqlite" /var/lib/boxweave/ledger.sqlite
op="$(sudo python3 - "$work/ledger/s5a2-cli-expected.json" <<'PY'
import json,re,sys
v=json.load(open(sys.argv[1]))
op=v["diagnosis"]["original_operation_id"]
assert re.fullmatch("[0-9a-f]{32}",op)
print(op)
PY
)"
recover() {
 sudo /usr/local/bin/boxweave recover -db /var/lib/boxweave/ledger.sqlite \
   -project "$project" -incus-socket "$socket" -service-user boxweave \
   -operation "$op" -reviewer s5a2-ci-reviewer "$@"
}
if recover -action diagnose > "$work/s5a2-cli-no-approval.txt" 2>&1; then exit 93; fi
if sudo runuser -u boxweave -- /usr/local/bin/boxweave recover \
   -db /var/lib/boxweave/ledger.sqlite -project "$project" -incus-socket "$socket" \
   -service-user boxweave -operation "$op" -reviewer s5a2-ci-reviewer \
   -action diagnose -approve-local > "$work/s5a2-cli-nonroot.txt" 2>&1; then exit 94; fi
recover -action diagnose -approve-local > "$work/s5a2-cli-diagnose.json"
recover -action reconcile -recovery-key s5a2-cli-bind-original-001 -approve-local > "$work/s5a2-cli-first.json"
recover -action reconcile -recovery-key s5a2-cli-bind-original-001 -approve-local > "$work/s5a2-cli-replay.json"
sudo python3 - "$work" "$db" "$op" <<'PY'
import json,sqlite3,sys
from pathlib import Path
w,source,op=sys.argv[1:]
w=Path(w)
expected=json.load(open(w/"ledger/s5a2-cli-expected.json"))
diagnose=json.load(open(w/"s5a2-cli-diagnose.json"))
first=json.load(open(w/"s5a2-cli-first.json"))
replay=json.load(open(w/"s5a2-cli-replay.json"))
d=diagnose["diagnosis"]
e=expected["diagnosis"]
for k in ("original_operation_id","original_action","original_state","original_native_operation_id","observed_native_uuid","claim_protection"):
    assert d[k]==e[k],(k,d,e)
assert d["disposition"]=="SAFE_RECOVER" and d["proposed_action"]=="BIND_IDENTITY"
assert diagnose.get("recovery") is None and not diagnose["native_retried"]
r=first["recovery"]; rr=replay["recovery"]
rid=r["operation"]["operation_id"]
assert r["disposition"]==expected["recovery"]["disposition"]=="IDENTITY_BOUND"
assert r["original_operation_id"]==op and r["operation"]["state"]=="SUCCEEDED"
assert rid!=op and rid!=expected["recovery"]["operation"]["operation_id"]
assert rr["operation"]["operation_id"]==rid and rr["operation"]["duplicate"] is True
assert first["diagnosis"]["original_state"]=="UNKNOWN"
assert first["diagnosis"]["claim_protection"]=="RETAINED"
assert not first["native_retried"] and not replay["native_retried"]
with sqlite3.connect("file:/var/lib/boxweave/ledger.sqlite?mode=ro",uri=True) as cl, sqlite3.connect("file:"+source+"?mode=ro",uri=True) as orig:
    for db in (cl,orig):
        row=db.execute("SELECT state,box_uuid,native_operation_id FROM operations WHERE id=?",(op,)).fetchone()
        assert row==("UNKNOWN",e["observed_native_uuid"],e["original_native_operation_id"]),row
        assert db.execute("SELECT count(*) FROM recovery_audit WHERE operation_id=? AND decision='RELEASED'",(op,)).fetchone()==(0,)
        assert db.execute("SELECT count(*) FROM recovery_audit WHERE operation_id=? AND decision='RECOVERY_IDENTITY_BOUND'",(op,)).fetchone()==(1,)
        assert db.execute("SELECT count(*) FROM operation_events WHERE operation_id=? AND state='UNKNOWN'",(op,)).fetchone()==(2,)
    original=orig.execute("SELECT seq,at,safe_detail FROM operation_events WHERE operation_id=? AND state='UNKNOWN' ORDER BY seq LIMIT 1",(op,)).fetchone()
    assert cl.execute("SELECT seq,at,safe_detail FROM operation_events WHERE operation_id=? AND state='UNKNOWN' ORDER BY seq LIMIT 1",(op,)).fetchone()==original
    assert cl.execute("SELECT actor_id,native_operation_id FROM operations WHERE id=?",(rid,)).fetchone()==("host-local:s5a2-ci-reviewer","")
print("S5A2_NATIVE_STAGED_ROOT_CLI_MCP_PARITY_NO_REPLAY=PASS")
PY
