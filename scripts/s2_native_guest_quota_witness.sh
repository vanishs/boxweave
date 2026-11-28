#!/bin/sh
# Sourced only by the guarded disposable S2/N2 lab after exact fixture Create.
# The caller owns lifecycle and supplies the restricted native SDK function.
set -eu
test "${GITHUB_ACTIONS:-}" = true
test "${RUNNER_ENVIRONMENT:-}" = github-hosted
test "$(id -u)" != 0
case "$work" in /tmp/bw-v1-s2-ghci.*|/tmp/bw-n2-installed.*) ;; *) exit 88 ;; esac
test "$project" = "user-$(id -u boxweave)"
test "$quota_name" = bw-s2-quota-native-110h
test "$(sudo incus config get "$quota_name" volatile.uuid --project "$project")" = "$quota_uuid"
test "$(findmnt -n -o FSTYPE --mountpoint "$work/mounted")" = ext4
findmnt -n -o OPTIONS --mountpoint "$work/mounted" | grep -Eq '(^|,)prjquota(,|$)'
test "$(sudo incus storage get bw-s2-native-pool source)" = "$work/mounted/native-pool"
# Wait for read-only guest shell readiness, never retry a native mutation.
ready=0
for attempt in $(seq 1 40); do
  if sudo incus exec "$quota_name" --project "$project" -- /bin/true >/dev/null 2>&1; then
    ready=1
    break
  fi
  sleep 1
done
test "$ready" -eq 1
s2_quota_native seed > "$work/quota-guest-seed.txt"
grep -Fqx "S2_NATIVE_QUOTA_SPOOL_SEEDED=PASS" "$work/quota-guest-seed.txt"

native_quota_spool="$(printf '/var/lib/incus/containers/%s_%s/exec-output' "$project" "$quota_name")"
sudo test -d "$native_quota_spool"
test "$(sudo stat -Lc '%d' "$native_quota_spool")" = "$(sudo stat -Lc '%d' "$work/mounted/native-pool")"
test "$(sudo stat -Lc '%d' "$native_quota_spool")" != "$(sudo stat -Lc '%d' /)"
sudo lsattr -p -d "$native_quota_spool"
quota_state_url="/1.0/storage-pools/bw-s2-native-pool/volumes/container/$quota_name/state?project=$project"
sudo incus query "$quota_state_url" > "$work/quota-guest-before.json"
quota_values="$(python3 - "$work/quota-guest-before.json" <<'PY'
import json,sys
v=json.load(open(sys.argv[1]))
v=v.get("metadata",v)["usage"]
print(int(v["used"]),int(v["total"]))
PY
)"
quota_used="$(printf '%s\n' "$quota_values" | awk '{print $1}')"
quota_limit="$(printf '%s\n' "$quota_values" | awk '{print $2}')"
test "$quota_limit" -eq 2147483648
test "$quota_used" -gt 10485760
quota_headroom=50331648
quota_fill_bytes=$((quota_limit - quota_used - quota_headroom))
test "$quota_fill_bytes" -gt 1073741824
physical_available="$(df -B1 --output=avail "$work/mounted" | tail -n1 | tr -d ' ')"
test "$physical_available" -gt $((quota_fill_bytes + 268435456))
host_available_before_fill="$(df -B1 --output=avail / | tail -n1 | tr -d ' ')"
test "$host_available_before_fill" -gt $((quota_fill_bytes + 4294967296))
quota_filler="$native_quota_spool/.boxweave-s2-guest-quota-filler"
sudo test ! -e "$quota_filler"
sudo timeout 30s setpriv --bounding-set=-sys_resource --nnp -- \
  fallocate -l "$quota_fill_bytes" "$quota_filler"
sudo incus query "$quota_state_url" > "$work/quota-guest-filled.json"
quota_used_filled="$(python3 - "$work/quota-guest-filled.json" <<'PY'
import json,sys
v=json.load(open(sys.argv[1]))
print(int(v.get("metadata",v)["usage"]["used"]))
PY
)"
test "$quota_used_filled" -gt $((quota_limit - 100663296))
test "$quota_used_filled" -le "$quota_limit"
echo "S2_NATIVE_GUEST_EXEC_QUOTA_NEAR_LIMIT_PRECONDITION=PASS"

# This original Incus operation is dispatched ONCE by the test-only native
# SDK helper. It tries to record 128MiB with only ~48MiB of quota left.
s2_quota_native flood > "$work/quota-guest-flood.txt"
quota_exec_id="$(sed -n 's/^S2_NATIVE_QUOTA_OP_ID=//p' "$work/quota-guest-flood.txt")"
echo "$quota_exec_id" | grep -Eq '^[0-9a-f-]{36}$'
grep -Fq 'S2_NATIVE_QUOTA_OP_STATUS=' "$work/quota-guest-flood.txt"
grep -Fq 'S2_NATIVE_QUOTA_OP_STATUS_CODE=' "$work/quota-guest-flood.txt"
# Only original Incus native status, operation ID and observed guest exit;
# no auth material, guest output or arbitrary host paths are logged.
cat "$work/quota-guest-flood.txt"
quota_stdout="$(printf '%s/exec_%s.stdout' "$native_quota_spool" "$quota_exec_id")"
sudo test -f "$quota_stdout"
quota_stdout_bytes="$(sudo stat -c '%s' "$quota_stdout")"
echo "S2_NATIVE_GUEST_QUOTA_PARTIAL_STDOUT_BYTES=$quota_stdout_bytes"
test "$quota_stdout_bytes" -gt 0
test "$quota_stdout_bytes" -lt 134217728
# Native quota recovery may briefly overlap Incus's accounting refresh.
# A failed read is NOT proof of safe quota; bounded read-only retries leave
# the original native Exec, SQLite and output files completely untouched.
quota_read_pass=0
for quota_attempt in 1 2 3 4; do
  if sudo incus query "$quota_state_url" > "$work/quota-guest-after-output.json"; then
    quota_read_pass=1
    break
  fi
  echo "S2_NATIVE_GUEST_QUOTA_STATE_TRANSIENT_READ_FAIL_ATTEMPT=$quota_attempt"
  sleep 1
done
if [ "$quota_read_pass" -ne 1 ]; then
  echo "S2_NATIVE_FAIL: actual native post-quota used/total unavailable" >&2
  exit 121
fi
quota_used_output="$(python3 - "$work/quota-guest-after-output.json" <<'PY'
import json,sys
v=json.load(open(sys.argv[1]))
print(int(v.get("metadata",v)["usage"]["used"]))
PY
)"
echo "S2_NATIVE_GUEST_QUOTA_USAGE_AFTER_OUTPUT_BYTES=$quota_used_output"
echo "S2_NATIVE_GUEST_QUOTA_VERIFIED_LIMIT_BYTES=$quota_limit"
if [ "$quota_used_output" -gt "$quota_limit" ]; then
  echo "S2_NATIVE_FAIL: actual native quota usage exceeded real limit" >&2
  exit 122
fi
echo "S2_NATIVE_REAL_GUEST_RECORDED_EXEC_QUOTA_PARTIAL_OUTPUT=PASS"

# Remove only the unique quota-test filler, NEVER original native
# stdout/stderr or an unresolved task. Independently confirm recovered space.
sudo rm -f -- "$quota_filler"
sudo test ! -e "$quota_filler"
sudo incus query "$quota_state_url" > "$work/quota-guest-reclaimed.json"
quota_used_reclaimed="$(python3 - "$work/quota-guest-reclaimed.json" <<'PY'
import json,sys
v=json.load(open(sys.argv[1]))
print(int(v.get("metadata",v)["usage"]["used"]))
PY
)"
test "$quota_used_reclaimed" -lt $((quota_used_output - quota_fill_bytes / 2))
test "$(sudo incus config get "$quota_name" volatile.uuid --project "$project")" = "$quota_uuid"
# Independent kernel refusal on the same exact owned volume after reclaim.
# The legacy 3GiB probe remains unchanged. Only the installed one-use
# Scoped runner executes this additional oracle, with physical headroom.
if [ "${P1B_S2_PHYSICAL_QUOTA:-0}" = 1 ]; then
  test "${PR178_SCOPED_N2:-0}" = 1
  test "$(sudo incus config device get "$quota_name" root size --project "$project")" = 2GiB
  test "$(sudo stat -Lc '%d' "$native_quota_spool")" = "$(sudo stat -Lc '%d' "$work/mounted/native-pool")"
  edquot_bytes=$((quota_limit + 4194304))
  physical_available="$(df -B1 --output=avail "$work/mounted" | tail -n1 | tr -d ' ')"
  host_available="$(df -B1 --output=avail / | tail -n1 | tr -d ' ')"
  test "$physical_available" -gt $((edquot_bytes + 268435456))
  test "$host_available" -gt $((edquot_bytes + 4294967296))
  edquot_file="$native_quota_spool/.boxweave-p1b-independent-edquot"
  sudo test ! -e "$edquot_file"
  sudo test ! -L "$edquot_file"
  edquot_rc=0
  sudo env LC_ALL=C timeout 40s setpriv --bounding-set=-sys_resource --nnp -- fallocate -l "$edquot_bytes" "$edquot_file" > "$work/direct-edquot.log" 2>&1 || edquot_rc=$?
  if [ "$edquot_rc" -eq 0 ] || ! grep -Fq 'Disk quota exceeded' "$work/direct-edquot.log"; then
    echo P1B_SCOPED_INDEPENDENT_EDQUOT_NOT_PROVED >&2
    exit 106
  fi
  sudo rm -- "$edquot_file"
  sudo test ! -e "$edquot_file"
  sudo incus query "$quota_state_url" > "$work/direct-edquot-after.json"
  python3 - "$work/direct-edquot-after.json" "$quota_limit" "$quota_used_reclaimed" <<'EDQUOT'
import json,sys
v=json.load(open(sys.argv[1]))
u=v.get("metadata",v)["usage"]
total,used,limit,reclaimed=int(u["total"]),int(u["used"]),int(sys.argv[2]),int(sys.argv[3])
print("P1B_SCOPED_EDQUOT_ACCOUNTING"
      f" total={total} used={used} expected_limit={limit}"
      f" reclaimed={reclaimed} delta={used-reclaimed} allowed_delta=1048576",flush=True)
assert total==limit, "EDQUOT changed the reported quota limit"
assert used<=limit, "EDQUOT postimage exceeded the quota limit"
assert used<=reclaimed+1048576, "EDQUOT postimage exceeded reclaimed usage tolerance"
print("P1B_SCOPED_INDEPENDENT_EDQUOT_REAL_KERNEL_REFUSAL=PASS")
EDQUOT
fi
