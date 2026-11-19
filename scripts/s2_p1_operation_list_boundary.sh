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
if ! command -v gdb >/dev/null 2>&1; then
 sudo DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends gdb >/dev/null
fi
command -v gdb >/dev/null 2>&1
lab="$work/p1-list-boundary"
mkdir -m 0700 "$lab"
sudo chown boxweave:incus "$lab"
db="$lab/ledger.sqlite"
actor() {
 sudo runuser -u boxweave -- env -u INCUS_CONF -u INCUS_CONFIG_DIR -u XDG_CONFIG_HOME \
   GITHUB_ACTIONS=true RUNNER_ENVIRONMENT=github-hosted \
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
anchor="return privateReadReply(ctx, s, operationCredentialID(req), operationListResult{TrackingID: tracking, Page: page})"
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
# Optimized Go inlines credential extraction into the return expression.
# Its first source-line PC can belong only to an invalid-credential branch.
# Stop at the unique actual privateReadReply CALL in the installed bytes,
# after successful ListOperations, before the final credential recheck.
go tool objdump -s 'registerOperationReadTools\.func2$' "$work/boxweave" \
  > "$work/p1-list-boundary.disassembly"
boundary_pc="$(python3 - "$work/boxweave" "$work/p1-list-boundary.disassembly" "$boundary_line" <<'PY_LIST_PC'
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
# Populate exactly this test's private ledger; zero native mutation.
actor "$work/boxweave" token-create -db "$db" > "$work/p1-list-debugger-admin.secret"
chmod 0600 "$work/p1-list-debugger-admin.secret"
cat > "$work/p1-list-boundary.gdb" <<EOF
set pagination off
set confirm off
set debuginfod enabled off
set breakpoint pending off
set print thread-events off
handle SIGURG nostop noprint pass
handle SIGPIPE nostop noprint pass
break *$boundary_pc
commands 1
silent
printf "S4_A_OPERATION_LIST_COMPILED_CALL_BREAKPOINT_HIT\\n"
shell /bin/sh -c "if test -e $lab/armed; then touch $lab/held; timeout 30 /bin/sh -c 'until test -e $lab/release; do sleep 0.1; done'; fi"
continue
end
run
EOF
sudo chown boxweave:incus "$work/p1-list-boundary.gdb"
debug_pid=""
cleanup() {
 sudo pkill -TERM -u boxweave -f "^$work/boxweave serve -db $db" || :
 if [ -n "$debug_pid" ]; then
  kill "$debug_pid" 2>/dev/null || :
  wait "$debug_pid" 2>/dev/null || :
 fi
}
trap cleanup EXIT
actor gdb -nx -q --batch -x "$work/p1-list-boundary.gdb" --args \
  "$work/boxweave" serve -db "$db" -incus-socket "$socket" \
  -project "$project" -listen 127.0.0.1:18667 \
  > "$work/p1-list-debugger.redacted.log" 2>&1 &
debug_pid=$!
attempt=0
until [ "$attempt" -ge 60 ]; do
 if ! kill -0 "$debug_pid" 2>/dev/null; then
  cat "$work/p1-list-debugger.redacted.log"
  echo "S2_P1_FAIL: debugger failed before source-boundary reachability" >&2
  exit 95
 fi
 code="$(curl -sS -o /dev/null -w '%{http_code}' http://127.0.0.1:18667/mcp 2>/dev/null || :)"
 if [ "$code" = 401 ]; then break; fi
 attempt=$((attempt+1))
 sleep 1
done
test "$attempt" -lt 60
if ! grep -Eq 'Breakpoint 1 at|Breakpoint 1,|Breakpoint 1 \(' "$work/p1-list-debugger.redacted.log"; then
 cat "$work/p1-list-debugger.redacted.log"
 echo "S2_P1_FAIL: candidate lacks source-mapped boundary; cannot accept race" >&2
 exit 96
fi
actor BW_S2_LIST_LAB="$lab" BW_S2_LIST_PROJECT="$project" \
 "$work/s2-p1-list-debugger-probe" > "$work/p1-list-boundary-results.txt" 2>&1 || {
 cat "$work/p1-list-boundary-results.txt"
 tail -100 "$work/p1-list-debugger.redacted.log"
 echo "S2_P1_FAIL: in-flight operation_list auth" >&2
 exit 97
}
grep -Fqx S2_P1_INSTALLED_OPERATION_LIST_POST_QUERY_PRE_RESPONSE_REVOKED_NO_LEAK=PASS \
 "$work/p1-list-boundary-results.txt"
grep -Fqx S2_P1_INSTALLED_OPERATION_LIST_POST_QUERY_PRE_RESPONSE_EXPIRED_NO_LEAK=PASS \
 "$work/p1-list-boundary-results.txt"
grep -Fqx S2_P1_INSTALLED_OPERATION_LIST_TWO_INFLIGHT_AUTH_RACES=PASS \
 "$work/p1-list-boundary-results.txt"
test "$(grep -Fxc S4_A_OPERATION_LIST_COMPILED_CALL_BREAKPOINT_HIT "$work/p1-list-debugger.redacted.log")" = 2
echo "S4_A_OPERATION_LIST_COMPILED_CALL_TWO_INDEPENDENT_AUTH_CUTS=PASS"
test -z "$(sudo incus list --all-projects --format csv -c n)"
echo "S2_P1_REAL_INSTALLED_OPERATION_LIST_INFLIGHT_REVOCATION_AND_EXPIRY_NO_LEAK=PASS"
