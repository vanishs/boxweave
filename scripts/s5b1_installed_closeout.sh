#!/bin/sh
# S5B1 test-only witness. Invoked by the existing pristine VM harness.
# Does not run on an existing user host or publish credentials/artifacts.
set -eu
test "${GITHUB_ACTIONS:-}" = true
test "${RUNNER_ENVIRONMENT:-}" = github-hosted
test "$(id -u)" != 0
test "$(git rev-parse HEAD)" = a3e02b4750edcabb80427e3311a19614d82b0029
test "$#" = 5
work="$1"; project="$2"; net="$3"; acl="$4"; binary_sha="$5"
case "$work" in /tmp/bw-v1-s25-lab.*) ;; *) exit 81;; esac
test "$project" = "user-$(id -u boxweave)"
test -z "$(sudo incus list --all-projects --format csv -c n)"
test "$(sudo incus project get "$project" restricted)" = true
test "$(sudo sha256sum /usr/local/bin/boxweave | cut -d' ' -f1)" = "$binary_sha"
sudo systemctl is-active --quiet boxweave.service
echo "S5B1_EXACT_SOURCE=$(git rev-parse HEAD)"
echo "S5B1_ACTUAL_BINARY_SHA256=$binary_sha"
if [ "$binary_sha" = a7a6a9fb7f32886cad8a891581cdc2eb5a8938200f48ea35b3a8cee3fed3fbf6 ]; then
  echo S5B1_LOCAL_AND_CI_BINARY_IDENTICAL=PASS
else
  echo S5B1_LOCAL_AND_CI_BINARY_IDENTICAL=UNVERIFIED
fi
# Build-info, installer status, restricted doctor and native plan/status
# must report actual success; missing journals must refuse, not create.
bw=/usr/local/bin/boxweave
service_cli() {
  sudo runuser -u boxweave -- env -u INCUS_CONF -u INCUS_CONFIG_DIR -u XDG_CONFIG_HOME HOME=/home/boxweave XDG_CONFIG_HOME=/home/boxweave/.config "$bw" "$@"
}
for leaf in build-info install-status native-prepare-status native-prereq-plan; do
  sudo "$bw" "$leaf" -project "$project" > "$work/private/cli-$leaf.json"
  echo "S5B1_CLI_POSITIVE=$leaf"
done
for leaf in status doctor oauth-status oauth-list-grants; do
  service_cli "$leaf" -project "$project" > "$work/private/cli-$leaf.json"
  echo "S5B1_CLI_POSITIVE=$leaf"
done
python3 - "$work/private" "$binary_sha" <<'PY'
import json,pathlib,sys
p=pathlib.Path(sys.argv[1])
assert json.load(open(p/'cli-build-info.json')) == {'version':'1.0.0-rc.10','commit':'a3e02b4750edcabb80427e3311a19614d82b0029'}
m=json.load(open(p/'cli-install-status.json'))
assert m['binary_sha256']==sys.argv[2] and m['service_active'] and not m['restart_required']
for n in ('status','doctor'):
 s=json.load(open(p/('cli-'+n+'.json')))
 assert s['ready'] and s['mcp_ready'] and s['schema_version']==5
PY
# Only explicit missing-approval or absent-journal negatives. No native
# package changes, arbitrary recovery release or synthetic approved plans.
for leaf in native-bootstrap native-migrate-apply native-migrate-reconcile native-migrate-rollback native-migrate-commit native-prepare-rollback native-prepare-archive native-peer-root-unit-stage native-peer-root-serve native-scriptlet-apply native-scriptlet-rollback native-scriptlet-reconcile install-stage install-activate upgrade rollback install-commit; do
  if sudo "$bw" "$leaf" -project "$project" > "$work/private/refused-$leaf.log" 2>&1; then
    echo "S5B1_CLI_UNEXPECTED_SUCCESS=$leaf" >&2; exit 82
  fi
  echo "S5B1_CLI_NO_APPROVAL_REFUSED=$leaf"
done
for leaf in native-migrate-status native-scriptlet-status; do
  if sudo "$bw" "$leaf" -project "$project" > "$work/private/refused-$leaf.log" 2>&1; then
    echo "S5B1_CLI_UNEXPECTED_JOURNAL=$leaf" >&2; exit 83
  fi
  echo "S5B1_CLI_ABSENT_JOURNAL_REFUSED=$leaf"
done
test "$(sudo sha256sum "$bw" | cut -d' ' -f1)" = "$binary_sha"
sudo systemctl is-active --quiet boxweave.service
test -z "$(sudo incus list --all-projects --format csv -c n)"
echo S5B1_CLI_GUARDED_ACTUAL_SUBSET=PASS
case "$S5B1_MODE" in
 sdk)
  legacy="$work/private/legacy-sdk"
  mkdir -m 0700 "$legacy"
  sed '1,2d' scripts/s2_legacy_sdk_probe.go > "$legacy/main.go"
  (
    cd "$legacy"
    GOWORK=off go mod init boxweave-s5b1-legacy-client
    GOWORK=off go get github.com/modelcontextprotocol/go-sdk@v1.6.1
    GOWORK=off go mod tidy
    test "$(GOWORK=off go list -m -f '{{.Version}}' github.com/modelcontextprotocol/go-sdk)" = v1.6.1
    GOWORK=off CGO_ENABLED=0 go build -trimpath -o "$work/staging/legacy-sdk" .
  )
  # Existing probe consumes only the already-issued isolated CI fixture.
  admin="$(sed -n 's/^token=//p' "$work/private/admin1.secret")"
  test -n "$admin"
  BW_S2_LEGACY_MCP_URL=http://127.0.0.1:18650/mcp BW_S2_LEGACY_TOKEN="$admin" "$work/staging/legacy-sdk" > "$work/private/legacy-sdk.log" 2>&1
  unset admin
  for marker in S2_P1_PREVIOUS_OFFICIAL_MCP_SDK_NATIVE_CREATE_UPDATE_EXEC_DELETE S4_B4_PREVIOUS_OFFICIAL_SDK_23_TOOLS_DISCOVERY S2_P1_PREVIOUS_OFFICIAL_MCP_SDK_LEGACY_PROTOCOL_16_TOOLS_NO_NATIVE_EFFECT; do
    grep -Fxq "$marker=PASS" "$work/private/legacy-sdk.log"
    echo "$marker=PASS"
  done
  test -z "$(sudo incus list --all-projects --format csv -c n)"
  test "$(go list -m -f '{{.Version}}' github.com/modelcontextprotocol/go-sdk)" = v1.8.0
  CGO_ENABLED=1 go build -trimpath -o "$work/staging/current-sdk" ./scripts/s3_file_installed_probe.go
  proxy_policy="$(sudo incus project get "$project" restricted.devices.proxy)"
  case "$proxy_policy" in allow) proxy_policy=allow;; ""|block) proxy_policy=deny;; *) echo S5B1_UNRECOGNIZED_NATIVE_PROXY_POLICY >&2; exit 88;; esac
  echo "S5B1_NATIVE_PROXY_POLICY=$proxy_policy"
  if ! BW_S5B1_PROXY_POLICY="$proxy_policy" BW_S5B1_EXACT_TOOLS=true BW_S3_MCP_URL=http://127.0.0.1:18650/mcp BW_S3_ADMIN_TOKEN_FILE="$work/private/admin2.secret" BW_S3_PEER_TOKEN_FILE="$work/private/peer.secret" BW_S3_FILE_DISPOSABLE=approved-ephemeral-vm "$work/staging/current-sdk" > "$work/private/current-sdk.log" 2>&1 ; then
    # Emit only approved fixed test marker classes. Never print raw HTTP,
    # credentials, private responses or native error contents.
    grep -E '^S5B1_STAGE=[a-z_]+$|^S5B1_EXACT_MCP_SCHEMA_PREFLIGHT=[a-z_]+$|^S3_INSTALLED_23_TOOL_OFFICIAL_SDK_CATALOG=PASS$' "$work/private/current-sdk.log" || :
    python3 - "$work/private/current-sdk.log" <<'PY'
import pathlib,re,sys
s=pathlib.Path(sys.argv[1]).read_text()
classes=[
"S5B1 installed identity","S5B1 exact installed binary mismatch",
"S5B1 missing actual metadata","S5B1 input schema not closed",
"S5B1 invalid arguments accepted","S5B1 transport not connected",
"S5B1 initialize instructions missing","S5B1 typed and text result missing",
"S5B1 typed result marshal","S5B1 typed result parse",
"S5B1 structured/text mismatch","S5B1 missing peer unexpectedly admitted",
"S5B1 missing forward owner unexpectedly admitted","S5B1 missing forward unexpectedly deleted",
"S5B1 missing cancellation unexpectedly admitted","S5B1 private response cache boundary missing",
"S5B1 recovery inspect unexpectedly replayed","S5B1 recovery original identity mismatch",
"S5B1 Activity original absent","S5B1 native forward receipt missing",
"S5B1 forward inventory mismatch","S5B1 forward exact device not verified",
"S5B1 native forward delete receipt missing","S5B1 removed forward remains"]
for c in classes:
 if c in s: print("S5B1_REDACTED_FAILURE="+c)
m=re.search(r'S3_INSTALLED_MCP_TOOL_ERROR name=([a-z_]+)',s)
if m: print("S5B1_REDACTED_FAILED_TOOL="+m[1])
if 'panic: runtime error:' in s: print('S5B1_REDACTED_FAILURE=GO_RUNTIME_PANIC')
for c in ['original durable operation ID absent','missing object diagnosis','native owned Guest is not running','new Box native UUID missing','original did not complete in bounded wait']:
 if c in s: print('S5B1_REDACTED_FAILURE='+c)
# Fixed method + terminal state labels only, never arbitrary response text.
m=re.search(r'original MCP tool ([a-z_]+) transport failed:',s)
if m: print('S5B1_REDACTED_TRANSPORT_TOOL='+m[1])
m=re.search(r'original ([0-9a-f]{32}) finished (FAILED|UNKNOWN|CANCELLED|SUCCEEDED) but expected (FAILED|SUCCEEDED|CANCELLED)',s)
if m: print('S5B1_REDACTED_ORIGINAL='+m[1]+' STATE='+m[2]+' EXPECTED='+m[3])
print("S5B1_CURRENT_SDK_FAIL_OR_BLOCKED")
PY
    exit 89
  fi
  for marker in S3_INSTALLED_23_TOOL_OFFICIAL_SDK_CATALOG S3_INSTALLED_REAL_HTTP_MCP_FILE_NEGATIVE_AND_IDEMPOTENCY S3_INSTALLED_MCP_ORIGINAL_BOX_CLEANUP_VERIFIED; do
    grep -Fxq "$marker=PASS" "$work/private/current-sdk.log"
    echo "$marker=PASS"
  done
  test -z "$(sudo incus list --all-projects --format csv -c n)"
  for marker in S5B1_EXACT_23_SCHEMA_CACHE_TYPED_TEXT_AND_REFUSALS; do
    grep -Fxq "$marker=PASS" "$work/private/current-sdk.log"
    echo "$marker=PASS"
  done
  grep '^S5B1_EXACT_READINESS_RECOVERY_ACTIVITY_FORWARD=PASS ' "$work/private/current-sdk.log"
  test "$(grep -c '^S5B1_EXACT_MCP_SCHEMA_PREFLIGHT=' "$work/private/current-sdk.log")" = 23
  echo S5B1_EXACT_INSTALLED_OLD_NEW_SDK_SCOPED=PASS
  ;;
 metadata)
  # Actual stopped Box with independently created owned native snapshot/backup.
  # No native security change or host credential exposure.
  PYTHONPATH=scripts python3 -B - "$work" "$project" "$binary_sha" <<'PY'
import os,sys,pathlib,subprocess,json,time,base64
from mcp_testkit import LoopbackMCP,MCPToolError
work=pathlib.Path(sys.argv[1]); project=sys.argv[2]; sha=sys.argv[3]
assert os.environ['GITHUB_ACTIONS']=='true' and os.environ['RUNNER_ENVIRONMENT']=='github-hosted'
assert os.geteuid()!=0
name='s5b2-metadata-disposable'
token=next(x[6:].strip() for x in (work/'private/admin2.secret').read_text().splitlines() if x.startswith('token='))
def native_argv(*args):
 # incus query rejects --project; pin the exact Project in its request URI.
 if args[0]=='query':
  from urllib.parse import urlsplit,parse_qs
  assert parse_qs(urlsplit(args[-1]).query).get('project')==[project]
  return ['sudo','incus',*args]
 return ['sudo','incus','--project',project,*args]
def native(*args):
 return subprocess.check_output(native_argv(*args),text=True,timeout=80).strip()
def call(m,t,**args):
 return m.call(t,args).structured
def submit(m,t,key,**args):
 z=call(m,t,idempotency_key='s5b2-metadata-'+key,**args)
 op=z['operation_id']
 assert len(op)==32
 return op
def await_state(m,op,want):
 r=m.poll_operation(op,seconds=100)
 assert r['state']==want,(op,r['state'],want)
 print('S5B2_ORIGINAL='+op+' STATE='+want+' NATIVE='+r.get('native_operation_id','none'),flush=True)
 return r
def ready(m,key,seconds):
 return submit(m,'box_wait_ready',key,box_id=box,checks=['running','exec'],timeout_seconds=seconds)
with LoopbackMCP(token=token,timeout=30,allow_mutations=True) as m:
 assert m.host_status()['binary_sha256']==sha
 assert not native('list','-f','csv','-c','n')
 create=submit(m,'box_create','create',name=name,cpu=1,memory_mib=768,disk_gib=4,start=False,internet=False)
 box=await_state(m,create,'SUCCEEDED')['box_uuid']
 assert native('config','get',name,'volatile.uuid')==box
 basic=call(m,'box_get',box_id=box)
 assert 'instance_full' not in basic and basic['box']['native_state']=='Stopped'
 t=ready(m,'timeout',1)
 o=await_state(m,t,'TIMED_OUT')
 assert not o.get('native_operation_id')
 waiting=ready(m,'delayed',90)
 start=submit(m,'box_start','start',box_id=box)
 await_state(m,start,'SUCCEEDED');await_state(m,waiting,'SUCCEEDED')
 e=submit(m,'box_exec','output',box_id=box,argv=['/bin/printf','s5b2-archive-marker'])
 await_state(m,e,'SUCCEEDED')
 stop=submit(m,'box_stop','stop',box_id=box)
 await_state(m,stop,'SUCCEEDED')
 assert native('config','get',name,'volatile.uuid')==box
 # Restricted Incus can refuse BOTH native operator creation requests.
 # Classify only an actual explicit Project policy refusal, independently
 # verify absence, and continue the unrelated readiness/retention tests.
 def metadata_create(kind,args):
  p=subprocess.run(native_argv(*args),text=True,capture_output=True,timeout=80)
  if p.returncode==0:
   print('S5B2_NATIVE_'+kind.upper()+'_CREATED=PASS',flush=True)
   return True
  msg=p.stderr.strip()
  if "Project "+json.dumps(project)+" doesn't allow for "+kind+" creation" not in msg:
   raise RuntimeError('unexpected native '+kind+' creation failure; not a policy refusal')
  print('S5B2_NATIVE_'+kind.upper()+'_PROJECT_POLICY_REFUSAL=PASS',flush=True)
  return False
 snapshot=metadata_create('snapshot',['snapshot','create',name,'s5b2-snap'])
 backup=metadata_create('backup',['query','-X','POST','--wait','-d',json.dumps(dict(name='s5b2-backup',instance_only=True)),f'/1.0/instances/{name}/backups?project={project}'])
 assert native('config','get',name,'volatile.uuid')==box
 # Native root API reads and MCP opt-in detail remain independently checked.
 # Read the exact native lists, not merely the expected empty default.
 for kind,created,item in [('snapshots',snapshot,'s5b2-snap'),('backups',backup,'s5b2-backup')]:
  rows=json.loads(native('query',f'/1.0/instances/{name}/{kind}?project={project}'))
  if isinstance(rows,dict): rows=rows['metadata']
  assert isinstance(rows,list) and len(rows)==int(created)
  if created: assert str(rows[0]).endswith('/'+item)
 det=call(m,'box_get',box_id=box,detail=True)['instance_full']
 assert det['config']['volatile.uuid']==box
 for kind,created,item in [('snapshots',snapshot,'s5b2-snap'),('backups',backup,'s5b2-backup')]:
  rows=det[kind] or []
  assert len(rows)==int(created)
  if created: assert rows[0]['name'].endswith(item)
 print('S5B2_ACTUAL_DETAIL_NATIVE_LIST_MATCH=PASS',flush=True)
 if snapshot and backup:
  print('S5B2_ACTUAL_NONEMPTY_DETAIL=PASS',flush=True)
 else:
  print('S5B2_NONEMPTY_DETAIL_COVERAGE=PROJECT_POLICY_LIMITED',flush=True)
 pending=ready(m,'restart',300)
 assert m.operation_get(pending)['state']=='OBSERVING'
# Restart only the freshly installed controller on this disposable VM.
subprocess.run(['sudo','systemctl','restart','boxweave.service'],check=True,timeout=30)
for _ in range(30):
 try:
  with LoopbackMCP(token=token,timeout=3) as check:
   assert check.host_status()['binary_sha256']==sha
  break
 except Exception: time.sleep(1)
else: raise AssertionError('service did not become ready')
with LoopbackMCP(token=token,timeout=30,allow_mutations=True) as m:
 original=await_state(m,pending,'FAILED')
 assert original['error_class']=='PROCESS_RESTART_BEFORE_DISPATCH' and not original.get('native_operation_id')
 same=call(m,'box_wait_ready',box_id=box,checks=['running','exec'],timeout_seconds=300,idempotency_key='s5b2-metadata-restart')
 assert same['duplicate'] and same['operation_id']==pending
 assert native('config','get',name,'volatile.uuid')==box
 if backup: native('query','-X','DELETE','--wait',f'/1.0/instances/{name}/backups/s5b2-backup?project={project}')
 if snapshot: native('snapshot','delete',name,'s5b2-snap')
 done=submit(m,'box_delete','delete',box_id=box)
 await_state(m,done,'SUCCEEDED')
 assert not native('list','-f','csv','-c','n')
 out=m.operation_output(e,limit=4096)
 assert out['exit_code']==0 and base64.b64decode(out['content_base64'])==b's5b2-archive-marker'
 print('S5B2_READY_TIMEOUT_DELAY_RESTART_NATIVE_DETAIL_ARCHIVE=PASS',flush=True)
PY
  ;;
 retirement)
  # Compatible VM, not the nested environment. Never modify drop-ins.
  test -z "$(sudo systemctl show boxweave.service -p DropInPaths --value)"
  test ! -e /var/lib/boxweave-installer-backup
  # Preserve and check actual native topology before/after retirement.
  sudo incus storage list --format csv -c n > "$work/private/storage.before"
  sudo incus network list --format csv -c n > "$work/private/networks.before"
  sudo incus project list --format csv -c n > "$work/private/projects.before"
  sudo "$bw" native-prepare-status > "$work/private/retirement-native.json"
  native_sha="$(python3 -c 'import json,sys;print(json.load(open(sys.argv[1]))["plan"]["review_sha256"])' "$work/private/retirement-native.json")"
  sudo sh uninstall.sh --plan --scope full --project "$project" --native-prepare-sha256 "$native_sha" > "$work/private/uninstall-plan.json"
  python3 - "$work/private/uninstall-plan.json" <<'PY'
import json,sys
p=json.load(open(sys.argv[1]))
assert p['ready'] and not p['blockers'] and not p['incus_instances'] and not p['sqlite']['boxes']
assert p['native_prepare']['status']=='APPLIED'
PY
  echo S5B1_CLEAN_VM_FULL_UNINSTALL_PLAN=PASS
  # The wrong review digest is a deliberate negative; no effect.
  host="$(python3 -c 'import json,sys;print(json.load(open(sys.argv[1]))["host_machine_id_sha256"])' "$work/private/uninstall-plan.json")"
  if sudo sh uninstall.sh --apply --scope full --project "$project" --native-prepare-sha256 "$native_sha" --confirm-host-sha256 "$host" --confirm-review-sha256 0000000000000000000000000000000000000000000000000000000000000000 > "$work/private/uninstall-wrong.log" 2>&1; then
    echo S5B1_UNEXPECTED_WRONG_UNINSTALL_APPROVAL >&2; exit 84
  fi
  sudo systemctl is-active --quiet boxweave.service
  test "$(sudo sha256sum "$bw"|cut -d' ' -f1)" = "$binary_sha"
  echo S5B1_WRONG_UNINSTALL_DIGEST_REFUSED=PASS
  # Explicitly authorized ephemeral fixture only: exercise the real guided
  # terminal path and verify the retirement archive; no shared native assets.
  test -z "$(sudo incus list --all-projects --format csv -c n)"
  printf 'REMOVE %s\n' "$project" | sudo script -q -e -c "sh uninstall.sh --full --project $project" /dev/null > "$work/private/guided-retirement.log" 2>&1
  sudo test ! -e "$bw"
  sudo test ! -e /etc/systemd/system/boxweave.service
  sudo test ! -e /var/lib/boxweave
  sudo test ! -e /var/lib/boxweave-native-prereq-v1.json
  ! sudo systemctl is-active --quiet boxweave.service
  sudo systemctl is-active --quiet incus.service
  test -z "$(sudo incus list --all-projects --format csv -c n)"
  sudo incus storage list --format csv -c n > "$work/private/storage.after"
  sudo incus network list --format csv -c n > "$work/private/networks.after"
  sudo incus project list --format csv -c n > "$work/private/projects.after"
  for kind in storage networks projects; do cmp "$work/private/$kind.before" "$work/private/$kind.after"; done
  sudo python3 - "$binary_sha" <<'PY'
import pathlib,json,hashlib,sqlite3,stat,sys
root=pathlib.Path('/var/lib/boxweave-retired')
assert root.stat().st_uid==0 and stat.S_IMODE(root.stat().st_mode)==0o700
items=list(root.glob('retired-*')); assert len(items)==1
p=items[0]
assert p.stat().st_uid==0 and stat.S_IMODE(p.stat().st_mode)==0o700
assert hashlib.sha256((p/'boxweave.bin').read_bytes()).hexdigest()==sys.argv[1]
for f in p.iterdir(): assert f.is_file() and not f.is_symlink() and f.stat().st_uid==0 and stat.S_IMODE(f.stat().st_mode)==0o600
with sqlite3.connect('file:'+str(p/'state-ledger.sqlite')+'?mode=ro',uri=True) as db:
 assert db.execute('pragma integrity_check').fetchone()[0]=='ok'
 assert db.execute('select version from schema_version').fetchone()[0]==5
assert json.load(open('/var/lib/boxweave-uninstall-v1-audit.json'))['status']=='COMPLETE'
PY
  echo S5B1_ACTUAL_GUIDED_FULL_RETIREMENT_ARCHIVE_AND_INCUS_PRESERVED=PASS
  ;;
 *) exit 85;;
esac
