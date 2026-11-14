#!/bin/sh
# On a verified one-use GitHub Ubuntu runner, test a *single* parent-level
# BPF devices interception change. Never run on retained Incus or L0/L1 host.
set -eu
test "${GITHUB_ACTIONS:-}" = true
test "${RUNNER_ENVIRONMENT:-}" = github-hosted
test "$(id -u)" -eq 0
. /etc/os-release
test "$ID" = ubuntu && test "$VERSION_ID" = 24.04
test -n "${BH019_REPORT_DIR:-}"
case "${BH019_PROXY_URL:-}" in
  http://10.*:18466|http://172.*:18466|http://192.168.*:18466) ;;
  *) echo BH019_UNAPPROVED_PROXY >&2; exit 83 ;;
esac
test -f scripts/bh019_inner_setup.sh
mkdir -p "$BH019_REPORT_DIR"
for case_name in control bpf-parent; do
  name="bh019-${case_name}-1114"
  test -z "$(incus list --format csv -c n | grep -Fx "$name" || true)"
  incus init images:ubuntu/24.04 "$name" -c security.nesting=true -c limits.cpu=1 -c limits.memory=2048MiB
  if [ "$case_name" = bpf-parent ]; then
    incus config set "$name" security.syscalls.intercept.bpf=true
    incus config set "$name" security.syscalls.intercept.bpf.devices=true
    test "$(incus config get "$name" security.syscalls.intercept.bpf)" = true
    test "$(incus config get "$name" security.syscalls.intercept.bpf.devices)" = true
  else
    test "$(incus config get "$name" security.syscalls.intercept.bpf)" != true
    test "$(incus config get "$name" security.syscalls.intercept.bpf.devices)" != true
  fi
  test "$(incus config get "$name" security.privileged)" != true
  incus config show "$name" --expanded > "$BH019_REPORT_DIR/${case_name}-expanded.yml"
  if ! incus start "$name" > "$BH019_REPORT_DIR/${case_name}-start.log" 2>&1; then
    cat "$BH019_REPORT_DIR/${case_name}-start.log"
    echo "BH019_PARENT_START=BLOCKED case=$case_name" | tee "$BH019_REPORT_DIR/${case_name}-result.txt"
    incus info --show-log "$name" >> "$BH019_REPORT_DIR/${case_name}-start.log" 2>&1 || true
    continue
  fi
  echo "BH019_PARENT_START=PASS case=$case_name" | tee "$BH019_REPORT_DIR/${case_name}-result.txt"
  incus file push scripts/bh019_inner_setup.sh "$name/tmp/bh019_inner_setup.sh"
  if incus exec "$name" -- /bin/sh /tmp/bh019_inner_setup.sh "$case_name" "$BH019_PROXY_URL" > "$BH019_REPORT_DIR/${case_name}-inner.log" 2>&1; then
    cat "$BH019_REPORT_DIR/${case_name}-inner.log"
  else
    cat "$BH019_REPORT_DIR/${case_name}-inner.log"
    echo "BH019_L2_START=BLOCKED_SETUP case=$case_name" | tee -a "$BH019_REPORT_DIR/${case_name}-result.txt"
    incus exec "$name" -- /usr/bin/journalctl -u incus.service -n 35 --no-pager >> "$BH019_REPORT_DIR/${case_name}-inner.log" 2>&1 || true
  fi
  if grep -q "BH019_L2_START=" "$BH019_REPORT_DIR/${case_name}-inner.log"; then
    grep "BH019_L2_START=" "$BH019_REPORT_DIR/${case_name}-inner.log" | tee -a "$BH019_REPORT_DIR/${case_name}-result.txt"
  fi
done
test -s "$BH019_REPORT_DIR/control-result.txt"
test -s "$BH019_REPORT_DIR/bpf-parent-result.txt"
if grep -q 'BH019_L2_START=BLOCKED_SETUP' "$BH019_REPORT_DIR/control-result.txt" "$BH019_REPORT_DIR/bpf-parent-result.txt"; then
  echo BH019_BPF_A_B=NOT_EVALUATED_SETUP_BLOCKED >&2
  exit 84
fi
echo BH019_BPF_CONTROL_AND_PARENT_INTERCEPT_EVIDENCE_RECORDED=PASS
# This means both experiments yielded evidence. It is deliberately NOT a
# claim that either L2 worked or that weaker policy is safe for BoxWeave.
