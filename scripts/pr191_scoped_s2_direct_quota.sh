# Sourced after the installed Scoped public-MCP quota fixture is deleted.
# The second fixture is owned through MCP; only the original direct native
# seed/flood uses the restricted SDK. Do not manufacture MCP Exec receipts.
test "${GITHUB_ACTIONS:-}" = true
test "${RUNNER_ENVIRONMENT:-}" = github-hosted
test "$(id -u)" != 0
case "$work" in /tmp/bw-n2-installed.*) ;; *) exit 88 ;; esac
test "$project" = "user-$(id -u boxweave)"
test "$(git rev-parse HEAD)" = "${N2_EXACT_HEAD:?}"
test -z "$(sudo incus list --all-projects -c n -f csv)"
quota_name=bw-s2-quota-native-110h
quota_uuid=""
quota_mcp() {
  BW_S2_MCP_QUOTA_URL=http://127.0.0.1:18650/mcp \
    BW_S2_MCP_QUOTA_TOKEN="$mcp_quota_token" \
    BW_S2_MCP_QUOTA_UUID="$quota_uuid" BW_S2_MCP_QUOTA_DIRECT_SCOPED=1 \
    GITHUB_ACTIONS=true RUNNER_ENVIRONMENT=github-hosted \
    "$work/s2-mcp-quota-probe" "$1"
}
s2_quota_native() {
  sudo runuser -u boxweave -- env -u INCUS_CONF -u INCUS_CONFIG_DIR -u XDG_CONFIG_HOME \
    GITHUB_ACTIONS=true RUNNER_ENVIRONMENT=github-hosted \
    HOME=/home/boxweave XDG_CONFIG_HOME=/home/boxweave/.config \
    BW_S2_QUOTA_SCOPED=1 BW_S2_QUOTA_PROJECT="$project" \
    BW_S2_QUOTA_SOCKET=/var/lib/incus/unix.socket.user BW_S2_QUOTA_UUID="$quota_uuid" \
    "$work/s2-native-guest-quota" "$1"
}
quota_mcp create > "$work/direct-quota-create.txt"
quota_uuid="$(sed -n 's/^S2_MCP_QUOTA_BOX_UUID=//p' "$work/direct-quota-create.txt")"
quota_create="$(sed -n 's/^P1B_DIRECT_QUOTA_CREATE_ID=//p' "$work/direct-quota-create.txt")"
echo "$quota_uuid" | grep -Eq '^[0-9a-f-]{36}$'
echo "$quota_create" | grep -Eq '^[0-9a-f]{32}$'
test "$(sudo incus list --all-projects -c n -f csv)" = "$quota_name"
# Cross-check ownership and exact scoped ACL before the test-only SDK call.
sudo python3 - "$mcp_quota_db" "$project" "$quota_uuid" "$quota_create" "$quota_name" <<'PY'
import json,sqlite3,subprocess,sys
db,p,u,oid,name=sys.argv[1:]
with sqlite3.connect("file:"+db+"?mode=ro",uri=True) as c:
    assert c.execute("SELECT layer,box_uuid,incarnation,state,action FROM operations WHERE id=?",(oid,)).fetchall()==[(p,u,u,"SUCCEEDED","box_create")]
    assert c.execute("SELECT state FROM owned_boxes WHERE project=? AND uuid=?",(p,u)).fetchall()==[("OWNED",)]
    [(acl,state)]=c.execute("SELECT acl,state FROM network_bindings WHERE project=? AND box_uuid=?",(p,u)).fetchall()
    assert state=="PINNED" and acl!="bw-peer-quarantine"
    obj=json.loads(subprocess.check_output(["incus","query",f"/1.0/instances/{name}?project={p}"]))
    assert obj["config"]["volatile.uuid"]==u
    assert obj["expanded_devices"]["eth0"]["security.acls"]==acl
    assert not c.execute("SELECT id FROM operations WHERE box_uuid=? AND action!='box_create'",(u,)).fetchall()
print("P1B_SCOPED_DIRECT_QUOTA_OWNERSHIP_PINNED=PASS")
PY
. scripts/s2_native_guest_quota_witness.sh
grep -Fxq S2_NATIVE_QUOTA_GUEST_EXIT=1 "$work/quota-guest-flood.txt"
quota_seed_id="$(sed -n 's/^S2_NATIVE_QUOTA_OP_ID=//p' "$work/quota-guest-seed.txt")"
echo "$quota_seed_id" | grep -Eq '^[0-9a-f-]{36}$'
test "$quota_seed_id" != "$quota_exec_id"
# The native operations are real but deliberately not MCP originals.
sudo python3 - "$mcp_quota_db" "$quota_uuid" "$quota_seed_id" "$quota_exec_id" <<'PY'
import sqlite3,sys
db,u,seed,flood=sys.argv[1:]
with sqlite3.connect("file:"+db+"?mode=ro",uri=True) as c:
    assert not c.execute("SELECT id FROM operations WHERE native_operation_id IN (?,?)",(seed,flood)).fetchall()
    assert not c.execute("SELECT id FROM operations WHERE box_uuid=? AND action!='box_create'",(u,)).fetchall()
    assert not c.execute("SELECT id FROM operations WHERE state IN ('ADMITTED','DISPATCHING','RUNNING','VERIFYING','UNKNOWN')").fetchall()
print("P1B_SCOPED_DIRECT_NATIVE_NO_SYNTHETIC_MCP_RECEIPTS=PASS")
PY
# Incus may retain completed native operations briefly. Observe only.
idle=0
for attempt in $(seq 1 75); do
  if test "$(sudo incus operation list --project "$project" -f json | python3 -c 'import json,sys;print(len(json.load(sys.stdin)))')" = 0; then
    idle=1
    break
  fi
  sleep 1
done
test "$idle" = 1
quota_mcp cleanup > "$work/direct-quota-cleanup.txt"
quota_stop="$(sed -n 's/^P1B_DIRECT_QUOTA_STOP_ID=//p' "$work/direct-quota-cleanup.txt")"
quota_delete="$(sed -n 's/^P1B_DIRECT_QUOTA_DELETE_ID=//p' "$work/direct-quota-cleanup.txt")"
test -z "$(sudo incus list --all-projects -c n -f csv)"
test "$runtime" = "$(sudo systemctl show boxweave.service boxweave-scoped-guard.service boxweave-scoped-lifecycle.service incus.service -p ActiveEnterTimestampMonotonic -p MainPID)"
sudo python3 - "$mcp_quota_db" "$project" "$quota_uuid" "$quota_create" "$quota_stop" "$quota_delete" "$quota_seed_id" "$quota_exec_id" <<'PY'
import json,sqlite3,subprocess,sys
db,p,u,create,stop,delete,seed,flood=sys.argv[1:]
with sqlite3.connect("file:"+db+"?mode=ro",uri=True) as c:
    assert set(c.execute("SELECT id,action,state,layer,incarnation FROM operations WHERE box_uuid=?",(u,)))=={(oid,action,"SUCCEEDED",p,u) for oid,action in [(create,"box_create"),(stop,"box_stop"),(delete,"box_delete")]}
    assert c.execute("SELECT state FROM owned_boxes WHERE project=? AND uuid=?",(p,u)).fetchall()==[("DELETED",)]
    assert c.execute("SELECT state FROM network_bindings WHERE project=? AND box_uuid=?",(p,u)).fetchall()==[("RELEASED",)]
    assert not c.execute("SELECT id FROM operations WHERE native_operation_id IN (?,?)",(seed,flood)).fetchall()
    assert not c.execute("SELECT id FROM operations WHERE state IN ('ADMITTED','DISPATCHING','RUNNING','VERIFYING','UNKNOWN')").fetchall()
assert {a["name"] for a in json.loads(subprocess.check_output(["incus","network","acl","list","--format=json"]))}=={"bw-peer-quarantine"}
print("P1B_SCOPED_DIRECT_QUOTA_FIXED_LEDGER_AND_NATIVE_ABSENCE=PASS",flush=True)
print("P1B_SCOPED_DIRECT_QUOTA_ORIGINALS",create,seed,flood,stop,delete,u,flush=True)
PY
echo S2_NATIVE_GUEST_QUOTA_SPACE_RECOVERED_AND_FIXTURE_REMOVED=PASS
