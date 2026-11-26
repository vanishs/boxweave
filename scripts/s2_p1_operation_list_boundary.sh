#!/bin/sh
# Exact original installed binary under its own service OS identity in GDB.
# Derive the unique source boundary after durable ListOperations SQL and before
# privateReadReply. Revocation/expiry is performed by an independent actor.
# No production hook or mock handler; this fixture is never shipped.
set -eu
if [ "${GITHUB_ACTIONS:-}" != true ] ||
   [ "${RUNNER_ENVIRONMENT:-}" != github-hosted ] ||
   [ "$(id -u)" -eq 0 ] || [ "$#" -ne 3 ]; then exit 91; fi
work="$1"; project="$2"; socket="$3"
case "$work" in /tmp/bw-v1-s2-ghci.*) ;; *) exit 92 ;; esac
test "$project" = "user-$(id -u boxweave)"
test "$socket" = /var/lib/incus/unix.socket.user
test -z "$(sudo incus list --all-projects --format csv -c n)"
if test "${BW_S2_INSTALLED_SCOPED_AUTH:-0}" = 1; then
  test -n "${N2_EXACT_HEAD:?}"
  test "$(git rev-parse HEAD)" = "$N2_EXACT_HEAD"
  test "$(sudo sha256sum /usr/local/bin/boxweave | cut -d' ' -f1)" = "${BW_S2_AUTH_CANDIDATE_SHA:?}"
  test "$(sudo systemctl show boxweave.service -p MainPID --value)" = 0
  sudo systemctl is-active --quiet boxweave-scoped-lifecycle.service boxweave-scoped-guard.service
  binary=/usr/local/bin/boxweave
  scoped_flag=-scoped-peer-fresh
else
  binary="$work/boxweave"
  scoped_flag=""
fi
if ! command -v gdb >/dev/null 2>&1; then
 sudo DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends gdb >/dev/null
fi
command -v gdb >/dev/null 2>&1
actor() {
 sudo runuser -u boxweave -- env -u INCUS_CONF -u INCUS_CONFIG_DIR -u XDG_CONFIG_HOME \
   GITHUB_ACTIONS=true RUNNER_ENVIRONMENT=github-hosted \
   BW_S2_INSTALLED_SCOPED_AUTH="${BW_S2_INSTALLED_SCOPED_AUTH:-0}" \
   BW_S2_TEST_UID="$(id -u boxweave)" \
   HOME=/home/boxweave XDG_CONFIG_HOME=/home/boxweave/.config "$@"
}
# A historical fixed line can silently move into Tool registration when new
# helpers are added. Require the exact post-query return and its error guard;
# source drift or ambiguity must fail before any debugger fixture is started.
boundary_line="$(python3 - internal/control/operation_tools.go <<'PY_LIST_BOUNDARY'
from pathlib import Path
import sys
lines=Path(sys.argv[1]).read_text(encoding="utf-8").splitlines()
anchor="return privateReadReply(ctx, s, principal.TokenID, operationListResult{TrackingID: tracking, Page: page})"
matches=[i for i,line in enumerate(lines) if line.strip()==anchor]
assert len(matches)==1, "unique final operation_list response boundary missing"
at=matches[0]
assert [line.strip() for line in lines[at-3:at]]==[
    "if err != nil {", "return nil, operationListResult{}, err", "}"
], "operation_list post-query error guard changed"
queries=[i for i,line in enumerate(lines[:at])
         if line.strip()=="page, err := s.Store.ListOperations(ctx, state.OperationQuery{"]
assert len(queries)==1 and at-queries[0]<=12, "final boundary is not immediately after ListOperations"
print(at+1)
PY_LIST_BOUNDARY
)"
echo "S4_A_OPERATION_LIST_POSTQUERY_BOUNDARY_LINE=$boundary_line"
# The source now passes the SDK-derived principal.TokenID directly.
# Source-line PCs may still be ambiguous under Go optimization.
# Stop at the unique actual privateReadReply CALL in the installed bytes,
# after successful ListOperations, before the final credential recheck.
go tool objdump -s 'registerOperationReadTools\.func2$' "$binary" \
  > "$work/p1-list-boundary.disassembly"
boundary_pc="$(python3 - "$binary" "$work/p1-list-boundary.disassembly" "$boundary_line" <<'PY_LIST_PC'
from pathlib import Path
import re,struct,sys
binary=Path(sys.argv[1]).read_bytes()
assert binary[:4]==b"\x7fELF" and binary[5] in (1,2), "unknown executable address format"
endian="<" if binary[5]==1 else ">"
assert struct.unpack(endian+"H",binary[16:18])[0]==2, "relocatable binary needs an independently verified relocated PC"
lines=Path(sys.argv[2]).read_text(encoding="utf-8").splitlines()
assert len([line for line in lines if line.startswith("TEXT ")])==1, "ambiguous compiled list handler"
matches=[]
for line in lines:
    if "CALL " in line and "control.privateReadReply[" in line and "state.OperationPage" in line:
        match=re.match(r"\s*operation_tools\.go:(\d+)\s+(0x[0-9a-f]+)\s+",line)
        assert match and int(match[1])==int(sys.argv[3]), "compiled call differs from exact source boundary"
        matches.append(match[2])
assert len(matches)==1, "unique compiled privateReadReply call missing"
print(matches[0])
PY_LIST_PC
)"
echo "S4_A_OPERATION_LIST_POSTQUERY_BOUNDARY_PC=$boundary_pc"
# Each mode has a fresh inferior, private ledger, and one-shot instruction
# breakpoint. A persistent software breakpoint can stop again at the same PC
# while GDB resumes a Go process (e.g. a signal/context switch). That is not
# evidence of a second MCP request. Do not silently discard extra requests:
# the probe issues one CallTool and the independent service log must contain
# exactly one completed tools/call for each isolated inferior.
cat > "$work/p1-list-boundary-hook.sh" <<'SH_LIST_HOOK'
#!/bin/sh
set -eu
lab="$1"
mode="$2"
case "$mode" in revoke|expire) ;; *) exit 1 ;; esac
if [ "$(cat "$lab/armed")" != "$mode" ]; then
 printf '%s\n' UNEXPECTED >> "$lab/breakpoint-hits"
 exit 1
fi
printf '%s\n' "$mode" >> "$lab/breakpoint-hits"
if [ -e "$lab/held" ] || [ -e "$lab/release" ]; then
 printf '%s\n' DUPLICATE_OR_STALE >> "$lab/breakpoint-hits"
 exit 1
fi
printf '%s\n' "$mode" > "$lab/held.tmp"
mv "$lab/held.tmp" "$lab/held"
if ! timeout 30 /bin/sh -c 'until test -e "$1/release"; do sleep 0.1; done' sh "$lab"; then
 printf '%s\n' HOLD_TIMEOUT >> "$lab/breakpoint-hits"
 exit 1
fi
SH_LIST_HOOK
chmod 0644 "$work/p1-list-boundary-hook.sh"

debug_pid=""
cleanup() {
 if [ -n "$debug_pid" ]; then
  sudo pkill -TERM -u boxweave -f "^$binary serve -db $db -incus-socket $socket -project $project -listen 127.0.0.1:18667" || :
  kill "$debug_pid" 2>/dev/null || :
  wait "$debug_pid" 2>/dev/null || :
  debug_pid=""
 fi
}
report_exit() {
 status=$?
 trap - EXIT
 if [ "$status" -ne 0 ]; then
  echo "S2_P1_LIST_BOUNDARY_FAIL: mode=$mode exit=$status" >&2
  if [ -f "$results" ]; then cat "$results" >&2; fi
  if sudo test -f "$lab/breakpoint-hits"; then
   echo "S2_P1_LIST_BOUNDARY_RECORDED_HITS:" >&2
   sudo cat "$lab/breakpoint-hits" >&2
  fi
  if [ -f "$log" ]; then tail -100 "$log" >&2; fi
 fi
 cleanup
 exit "$status"
}

run_case() {
 mode="$1"
 case "$mode" in revoke|expire) ;; *) exit 93 ;; esac
 lab="$work/p1-list-boundary-$mode"
 db="$lab/ledger.sqlite"
 if test -n "$scoped_flag"; then db=/var/lib/boxweave/ledger.sqlite; fi
 log="$work/p1-list-debugger-$mode.redacted.log"
 results="$work/p1-list-boundary-$mode-results.txt"
 gdb_script="$work/p1-list-boundary-$mode.gdb"
 # Refuse stale fixture/port before starting another independent inferior.
 test -z "$(ss -H -ltn 'sport = :18667')"
 mkdir -m 0700 "$lab"
 sudo chown boxweave:incus "$lab"
 trap report_exit EXIT
 # Seed only this mode's ledger. No native mutation or shared credential.
 if test -z "$scoped_flag"; then
   actor "$binary" token-create -db "$db" > "$work/p1-list-debugger-$mode-admin.secret"
   chmod 0600 "$work/p1-list-debugger-$mode-admin.secret"
 fi
 cat > "$gdb_script" <<EOF
set pagination off
set confirm off
set debuginfod enabled off
set breakpoint pending off
set print thread-events off
handle SIGURG nostop noprint pass
handle SIGPIPE nostop noprint pass
tbreak *$boundary_pc
commands 1
silent
printf "S4_A_OPERATION_LIST_COMPILED_CALL_BREAKPOINT_HIT\\n"
shell /bin/sh "$work/p1-list-boundary-hook.sh" "$lab" "$mode"
continue
end
run
EOF
 sudo chown boxweave:incus "$gdb_script"
 actor gdb -nx -q --batch -x "$gdb_script" --args \
  "$binary" serve -db "$db" -incus-socket "$socket" \
  -project "$project" -listen 127.0.0.1:18667 $scoped_flag > "$log" 2>&1 &
 debug_pid=$!
 attempt=0
 until [ "$attempt" -ge 60 ]; do
  if ! kill -0 "$debug_pid" 2>/dev/null; then
   echo "S2_P1_FAIL: debugger failed before source-boundary reachability" >&2
   exit 95
  fi
  code="$(curl -sS -o /dev/null -w '%{http_code}' http://127.0.0.1:18667/mcp 2>/dev/null || :)"
  if [ "$code" = 401 ]; then break; fi
  attempt=$((attempt+1))
  sleep 1
 done
 test "$attempt" -lt 60
 if ! grep -Eq 'Temporary breakpoint 1 at|Temporary breakpoint 1,|Temporary breakpoint 1 \(' "$log"; then
  echo "S2_P1_FAIL: candidate lacks one-shot compiled boundary" >&2
  exit 96
 fi
 actor BW_S2_LIST_LAB="$lab" BW_S2_LIST_PROJECT="$project" BW_S2_LIST_MODE="$mode" \
  "$work/s2-p1-list-debugger-probe" > "$results" 2>&1 || {
  echo "S2_P1_FAIL: in-flight operation_list auth" >&2
  exit 97
 }
 case "$mode" in revoke) label=REVOKED ;; expire) label=EXPIRED ;; esac
 grep -Fqx "S2_P1_INSTALLED_OPERATION_LIST_POST_QUERY_PRE_RESPONSE_${label}_NO_LEAK=PASS" "$results"
 cat "$results"
 # Wait for the independent service completion log, not another request.
 attempt=0
 until [ "$(grep -c 'mcp_request method=tools/call ' "$log" || :)" -ge 1 ] || [ "$attempt" -ge 50 ]; do
  attempt=$((attempt+1))
  sleep 0.1
 done
 cleanup
 if [ "$(grep -c 'mcp_request method=tools/call ' "$log" || :)" -ne 1 ]; then
  echo "S2_P1_FAIL: expected exactly one independent service tool-call receipt ($mode)" >&2
  exit 98
 fi
 if [ "$(sudo cat "$lab/breakpoint-hits")" != "$mode" ]; then
  echo "S2_P1_FAIL: exact single breakpoint witness mismatch ($mode)" >&2
  exit 98
 fi
 echo "S4_A_OPERATION_LIST_ONE_SHOT_WITNESS=$mode"
 echo "S4_A_OPERATION_LIST_SERVICE_CALL_COUNT=$mode:1"
 trap - EXIT
 # Account for asynchronous inferior exit without touching any other process.
 attempt=0
 until [ -z "$(ss -H -ltn 'sport = :18667')" ] || [ "$attempt" -ge 50 ]; do
  attempt=$((attempt+1))
  sleep 0.1
 done
 test -z "$(ss -H -ltn 'sport = :18667')"
}
run_case revoke
run_case expire
# Join the independent files in execution order; neither mode can reuse a
# witness, process, token, or breakpoint from the other.
expected="$(printf 'revoke\nexpire\n')"
actual="$(sudo cat "$work/p1-list-boundary-revoke/breakpoint-hits" "$work/p1-list-boundary-expire/breakpoint-hits")"
test "$actual" = "$expected"
echo S2_P1_INSTALLED_OPERATION_LIST_TWO_INFLIGHT_AUTH_RACES=PASS
echo "S4_A_OPERATION_LIST_BREAKPOINT_STDOUT_COUNT=$(grep -h -Fxc S4_A_OPERATION_LIST_COMPILED_CALL_BREAKPOINT_HIT "$work/p1-list-debugger-revoke.redacted.log" "$work/p1-list-debugger-expire.redacted.log" | awk '{s+=$1} END {print s}')"
echo "S4_A_OPERATION_LIST_BREAKPOINT_FILE_ORDERED_HITS=revoke,expire"
echo "S4_A_OPERATION_LIST_COMPILED_CALL_TWO_INDEPENDENT_AUTH_CUTS=PASS"
test -z "$(sudo incus list --all-projects --format csv -c n)"
echo "S2_P1_REAL_INSTALLED_OPERATION_LIST_INFLIGHT_REVOCATION_AND_EXPIRY_NO_LEAK=PASS"
