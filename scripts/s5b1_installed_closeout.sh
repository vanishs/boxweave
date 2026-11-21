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
  BW_S3_MCP_URL=http://127.0.0.1:18650/mcp BW_S3_ADMIN_TOKEN_FILE="$work/private/admin2.secret" BW_S3_PEER_TOKEN_FILE="$work/private/peer.secret" BW_S3_FILE_DISPOSABLE=approved-ephemeral-vm "$work/staging/current-sdk" > "$work/private/current-sdk.log" 2>&1
  for marker in S3_INSTALLED_23_TOOL_OFFICIAL_SDK_CATALOG S3_INSTALLED_REAL_HTTP_MCP_FILE_NEGATIVE_AND_IDEMPOTENCY S3_INSTALLED_MCP_ORIGINAL_BOX_CLEANUP_VERIFIED; do
    grep -Fxq "$marker=PASS" "$work/private/current-sdk.log"
    echo "$marker=PASS"
  done
  test -z "$(sudo incus list --all-projects --format csv -c n)"
  echo S5B1_EXACT_INSTALLED_OLD_NEW_SDK_SCOPED=PASS
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
