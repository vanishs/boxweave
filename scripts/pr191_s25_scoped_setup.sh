#!/bin/sh
# One-use S25 test entry. Reuse the same reviewed native bootstrap and
# supported Scoped wrapper as N2. No persistent bypass or test-only API.
set -eu
umask 077
test "$(id -u)" = 0
test "${P1B_S25_SCOPED:-}" = 1
case "${BW_S25_SETUP_LAYER:-}" in
 outer)
   test "${GITHUB_ACTIONS:-}" = true
   test "${RUNNER_ENVIRONMENT:-}" = github-hosted
   ;;
 inner)
   test "${BW_S25_ALLOWED_NESTED:-}" = 1
   test "$(hostname)" = bw-s25-mcp-l1
   # Reject parent daemon bind mounts rather than inheriting their authority.
   awk '$5 ~ /^\/(var\/lib|run)\/incus(\/|$)/ {bad=1} END {exit bad}' /proc/self/mountinfo
   ;;
 *) exit 88 ;;
esac
test "$#" = 5
action="$1"; candidate="$2"; digest="$3"; target="$4"; work="$5"
case "$candidate" in /usr/local/bin/boxweave-s25-candidate) ;; *) exit 88 ;; esac
case "$work" in /root|/tmp/bw-v1-s25-lab.*/private) ;; *) exit 88 ;; esac
test "$(sha256sum "$candidate" | cut -d' ' -f1)" = "$digest"
test ! -e /var/lib/boxweave
test ! -e /usr/local/bin/boxweave
test ! -e /var/lib/boxweave-scoped-peer-v2
test ! -e /var/lib/boxweave-native-prereq-v1.json
case "$action" in
 bootstrap)
   test ! -e /var/lib/incus
   ! id boxweave >/dev/null 2>&1
   "$candidate" native-plan -incus-package-version "$target" > "$work/s25-scoped-native-plan.json"
   review="$(python3 -c 'import json,sys;v=json.load(open(sys.argv[1]));assert v["mode"]=="fresh-install-proposed";print(v["plan_sha256"])' "$work/s25-scoped-native-plan.json")"
   "$candidate" native-bootstrap -incus-package-version "$target" -native-plan-sha256 "$review" -approve-native-bootstrap > "$work/s25-scoped-native-apply.json"
   python3 - "$work/s25-scoped-native-apply.json" <<'CHECK'
import ipaddress,json,sys
x=json.load(open(sys.argv[1]))
assert x["mode"]=="new-daemon-ready"
assert ipaddress.ip_interface(x["ipv4_bridge_prefix"]).network.prefixlen<=21 and x["ipv4_address_pool"]>=2045
CHECK
   echo "P1B_S25_SCOPED_NATIVE_BOOTSTRAP=${BW_S25_SETUP_LAYER}:PASS"
   ;;
 install)
   project="$target"
   test "$project" = "user-$(id -u boxweave)"
   test -z "$(incus list --all-projects --format csv -c n)"
   # Exact root-reviewed wrapper is placed alongside the private helper.
   wrapper="$work/s25-install.sh"
   test "$(sha256sum "$wrapper" | cut -d' ' -f1)" = "${BW_S25_INSTALL_SHA:?}"
   sh "$wrapper" --plan --scoped-peer-fresh --local-binary "$candidate" --sha256 "$digest" > "$work/s25-scoped-plan.txt"
   grep -Fxq "project=$project" "$work/s25-scoped-plan.txt"
   host="$(sed -n 's/^host_machine_id_sha256=//p' "$work/s25-scoped-plan.txt")"
   review="$(sed -n 's/^scoped_fresh_review_sha256=//p' "$work/s25-scoped-plan.txt")"
   printf '%s\n' "$host" | grep -Ex '[0-9a-f]{64}' >/dev/null
   printf '%s\n' "$review" | grep -Ex '[0-9a-f]{64}' >/dev/null
   if sh "$wrapper" --apply --scoped-peer-fresh --project "$project" --local-binary "$candidate" --sha256 "$digest" --confirm-host-sha256 "$host" --scoped-fresh-review-sha256 "$(printf '%064d' 0)" > "$work/s25-scoped-stale.log" 2>&1; then
     echo S25_FAIL_STALE_SCOPED_INSTALL_ACCEPTED >&2; exit 89
   fi
   grep -Fq 'scoped fresh review digest changed' "$work/s25-scoped-stale.log"
   test ! -e /usr/local/bin/boxweave
   test ! -e /var/lib/boxweave
   test ! -e /var/lib/boxweave-scoped-peer-v2
   echo "P1B_S25_SCOPED_STALE_REVIEW_REFUSED=${BW_S25_SETUP_LAYER}:PASS"
   sh "$wrapper" --apply --scoped-peer-fresh --project "$project" --local-binary "$candidate" --sha256 "$digest" --confirm-host-sha256 "$host" --scoped-fresh-review-sha256 "$review" > "$work/s25-scoped-install.log"
   test "$(sha256sum /usr/local/bin/boxweave | cut -d' ' -f1)" = "$digest"
   systemctl is-active --quiet boxweave.service boxweave-scoped-guard.service boxweave-scoped-lifecycle.service
   /usr/local/bin/boxweave install-status -project "$project" > "$work/s25-scoped-status.json"
   python3 - "$work/s25-scoped-status.json" "$project" "$digest" <<'CHECK'
import json,sys
s=json.load(open(sys.argv[1]))
assert s["scoped_peer_mode"]=="SCOPED_PEER_FRESH_V2"
assert s["native_project"]==sys.argv[2] and s["binary_sha256"]==sys.argv[3]
assert s["service_active"] and s["restart_required"] is False
CHECK
   echo "P1B_S25_SCOPED_FRESH_INSTALL=${BW_S25_SETUP_LAYER}:PASS"
   ;;
 *) exit 88 ;;
esac
