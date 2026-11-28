# Shared original S2 quota fixture; no production entry point.
# Source from the guarded one-use hosted runner only.
test "${GITHUB_ACTIONS:-}" = true
test "${RUNNER_ENVIRONMENT:-}" = github-hosted
test "$(id -u)" != 0
case "${work:?}" in /tmp/bw-v1-s2-ghci.*|/tmp/bw-n2-installed.*) ;; *) exit 88 ;; esac
test "$(realpath "$work")" = "$work"
test "$(stat -c '%u' "$work")" = "$(id -u)"

test "$project" = "user-$(id -u boxweave)"
case "$mcp_quota_db" in
  "$work/mcp-quota-ledger/ledger.sqlite"|/var/lib/boxweave/ledger.sqlite) ;;
  *) exit 88 ;;
esac
test -z "$(sudo incus list --all-projects --format csv -c n)"
test "$(sudo incus profile device get default root pool --project "$project")" = bw-s2-native-pool
test "$(sudo incus storage get bw-s2-native-pool source)" = "$work/mounted/native-pool"
test "$(findmnt -n -o TARGET --target "$work/mounted")" = "$work/mounted"
test "$(findmnt -n -o FSTYPE --target "$work/mounted")" = ext4
findmnt -n -o OPTIONS --target "$work/mounted" | grep -Eq '(^|,)prjquota(,|$)'
test "$(sudo stat -Lc '%d' "$work/mounted/native-pool")" != "$(stat -Lc '%d' /)"
mcp_quota_name="bw-s2-mcp-quota-110i"
s2_mcp_quota create > "$work/mcp-quota-create.txt"
mcp_quota_uuid="$(sed -n 's/^S2_MCP_QUOTA_BOX_UUID=//p' "$work/mcp-quota-create.txt")"
echo "$mcp_quota_uuid" | grep -Eq '^[0-9a-f-]{36}$'
test "$(sudo incus config get "$mcp_quota_name" volatile.uuid --project "$project")" = "$mcp_quota_uuid"
mcp_ready=0
for attempt in $(seq 1 40); do
  if sudo incus exec "$mcp_quota_name" --project "$project" -- /bin/true >/dev/null 2>&1; then
    mcp_ready=1
    break
  fi
  sleep 1
done
test "$mcp_ready" -eq 1
s2_mcp_quota seed > "$work/mcp-quota-seed.txt"
grep -Fqx "S2_MCP_QUOTA_SEED_RECORDED=PASS" "$work/mcp-quota-seed.txt"

mcp_spool="$(printf '/var/lib/incus/containers/%s_%s/exec-output' "$project" "$mcp_quota_name")"
sudo test -d "$mcp_spool"
test "$(sudo stat -Lc '%d' "$mcp_spool")" = "$(sudo stat -Lc '%d' "$work/mounted/native-pool")"
mcp_state_url="/1.0/storage-pools/bw-s2-native-pool/volumes/container/$mcp_quota_name/state?project=$project"
sudo incus query "$mcp_state_url" > "$work/mcp-quota-before.json"
mcp_quota_values="$(python3 - "$work/mcp-quota-before.json" <<'PY'
import json,sys
v=json.load(open(sys.argv[1]))
v=v.get("metadata",v)["usage"]
print(int(v["used"]),int(v["total"]))
PY
)"
mcp_used="$(printf '%s\n' "$mcp_quota_values" | awk '{print $1}')"
mcp_limit="$(printf '%s\n' "$mcp_quota_values" | awk '{print $2}')"
test "$mcp_limit" -eq 2147483648
test "$mcp_used" -gt 10485760
mcp_fill_bytes=$((mcp_limit - mcp_used - 50331648))
test "$mcp_fill_bytes" -gt 1073741824
physical_available="$(df -B1 --output=avail "$work/mounted" | tail -n1 | tr -d ' ')"
host_available="$(df -B1 --output=avail / | tail -n1 | tr -d ' ')"
test "$physical_available" -gt $((mcp_fill_bytes + 268435456))
test "$host_available" -gt $((mcp_fill_bytes + 4294967296))
mcp_filler="$mcp_spool/.boxweave-s2-public-mcp-quota-filler"
sudo test ! -e "$mcp_filler"
sudo timeout 30s setpriv --bounding-set=-sys_resource --nnp -- \
  fallocate -l "$mcp_fill_bytes" "$mcp_filler"
echo "S2_MCP_QUOTA_NEAR_LIMIT_SAFE_MOUNT=PASS"

s2_mcp_quota flood > "$work/mcp-quota-flood.txt"
grep -Fqx "S2_MCP_QUOTA_GUEST_EXIT_ONE_PUBLIC_OUTPUT=PASS" "$work/mcp-quota-flood.txt"
mcp_original="$(sed -n 's/^S2_MCP_QUOTA_OPERATION_ID=//p' "$work/mcp-quota-flood.txt")"
mcp_native="$(sed -n 's/^S2_MCP_QUOTA_NATIVE_ID=//p' "$work/mcp-quota-flood.txt")"
echo "$mcp_original" | grep -Eq '^[0-9a-f]{32}$'
echo "$mcp_native" | grep -Eq '^[0-9a-f-]{36}$'
mcp_output_file="$(printf '%s/exec_%s.stdout' "$mcp_spool" "$mcp_native")"
sudo test -f "$mcp_output_file"
mcp_output_bytes="$(sudo stat -c '%s' "$mcp_output_file")"
echo "S2_MCP_QUOTA_PHYSICAL_STDOUT_BYTES=$mcp_output_bytes"
test "$mcp_output_bytes" -gt 16777216
test "$mcp_output_bytes" -lt 134217728
# At the hard native quota edge, the independent storage-state GET can
# transiently fail or lag behind ext4 quota accounting. Retry ONLY this
# read-only query, never Exec/native mutation or the original MCP operation.
# Persistent over-limit or unreadable state still FAILS closed.
mcp_after=""
mcp_after_read_attempts=0
for mcp_read_attempt in 1 2 3 4 5 6 7 8 9 10 11 12; do
  mcp_after_read_attempts="$mcp_read_attempt"
  if sudo incus query "$mcp_state_url" > "$work/mcp-quota-after.json" 2> "$work/mcp-quota-after.last-error"; then
    mcp_after="$(python3 - "$work/mcp-quota-after.json" <<'PY'
import json,sys
v=json.load(open(sys.argv[1]))
print(int(v.get("metadata",v)["usage"]["used"]))
PY
)" || mcp_after=""
    case "$mcp_after" in
      ''|*[!0-9]*) mcp_after="" ;;
    esac
    if [ -n "$mcp_after" ] && [ "$mcp_after" -le "$mcp_limit" ]; then
      break
    fi
  fi
  sleep 0.3
done
echo "S2_MCP_QUOTA_POSTEXEC_NATIVE_USED_BYTES=${mcp_after:-UNREADABLE}"
echo "S2_MCP_QUOTA_POSTEXEC_LIMIT_BYTES=$mcp_limit"
echo "S2_MCP_QUOTA_POSTEXEC_READ_ATTEMPTS=$mcp_after_read_attempts"
if [ -z "$mcp_after" ] || [ "$mcp_after" -gt "$mcp_limit" ]; then
  echo "S2_MCP_FAIL: bounded read-only native quota accounting not within its exact limit" >&2
  cat "$work/mcp-quota-after.last-error" >&2 || :
  exit 97
fi
echo "S2_MCP_REAL_GUEST_QUOTA_ORIGINAL_RECEIPT_OUTPUT_EXIT_ONE=PASS"

# The one-use test has already independently verified the exact immutable
# native operation and stdout path. Delete only its ORIGINAL stdout through
# the project-scoped official Incus API; do not delete stderr or foreign logs.
# This is an independent deliberate log-loss negative, NOT automatic GC.
sudo incus query -X DELETE "/1.0/instances/$mcp_quota_name/logs/exec-output/exec_$mcp_native.stdout?project=$project"   > "$work/mcp-quota-native-log-delete.json"
sudo test ! -f "$mcp_output_file"
# Native log loss alone now has a private 1 MiB archive fallback; verify it.
s2_mcp_quota archive > "$work/mcp-quota-native-archived.txt"
grep -Fqx 'S2_MCP_EXACT_ORIGINAL_NATIVE_LOG_MISSING_ARCHIVE_PREFIX=PASS' "$work/mcp-quota-native-archived.txt"
echo "S2_MCP_EXACT_ORIGINAL_NATIVE_LOG_MISSING_ARCHIVE_PREFIX=PASS"
# Deliberately remove ONLY the exact disposable original archive, to retain
# the old real no-native/no-archive typed-unavailable negative.
mcp_archive_file="$mcp_quota_db.exec-archive/$mcp_original.stdout"
sudo test -f "$mcp_archive_file"
test "$(sudo stat -c '%s' "$mcp_archive_file")" -eq 1048576
sudo -u boxweave rm -- "$mcp_archive_file"
sudo test ! -e "$mcp_archive_file"
s2_mcp_quota missing > "$work/mcp-quota-native-missing.txt"
grep -Fqx 'S2_MCP_EXACT_ORIGINAL_NATIVE_LOG_MISSING_TYPED_UNAVAILABLE=PASS'   "$work/mcp-quota-native-missing.txt"
echo "S2_MCP_NATIVE_OUTPUT_404_TYPED_WITHOUT_FALSE_EXPIRY=PASS"

# Remove only disposable quota preallocation, preserve original logs until
# public MCP cleanup removes the verified exact UUID, never an UNKNOWN Box.
sudo rm -f -- "$mcp_filler"
sudo test ! -e "$mcp_filler"
sudo incus query "$mcp_state_url" > "$work/mcp-quota-reclaimed.json"
mcp_reclaimed="$(python3 - "$work/mcp-quota-reclaimed.json" <<'PY'
import json,sys
v=json.load(open(sys.argv[1]))
print(int(v.get("metadata",v)["usage"]["used"]))
PY
)"
test "$mcp_reclaimed" -lt $((mcp_after - mcp_fill_bytes / 2))
