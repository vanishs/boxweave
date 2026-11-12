#!/bin/sh
# One-use GitHub VM only. An isolated routed TLS endpoint behind its own
# namespace/veth is a **SIMULATED WAN**, never a real public Internet PASS.
set -eu
[ "${GITHUB_ACTIONS:-}" = true ] && [ "${RUNNER_ENVIRONMENT:-}" = github-hosted ]
[ "$(id -u)" -ne 0 ] && [ "$#" -eq 2 ]
mode="$1"; work="$2"
case "$work" in /tmp/bw-v1-s2-ghci.*) ;; *) exit 88 ;; esac
ns="bw-s2-simwan"
uplink="bw-s2-simwan-host"
peer="bw-s2-simwan-peer"
cert="$work/s2-simwan-cert.pem"
key="$work/s2-simwan-key.pem"
log="$work/s2-simwan-hits.log"
case "$mode" in
setup)
  test ! -e "/run/netns/$ns"
  test -z "$(sudo ip link show "$uplink" 2>/dev/null || :)"
  command -v openssl >/dev/null
  command -v python3 >/dev/null
  sudo openssl req -x509 -newkey rsa:2048 -nodes -days 1 \
    -keyout "$key" -out "$cert" \
    -subj "/CN=s2-wan.example.test" \
    -addext "subjectAltName=DNS:s2-wan.example.test" \
    >/dev/null 2>&1
  sudo chmod 0600 "$key"
  sudo ip netns add "$ns"
  sudo ip link add "$uplink" type veth peer name "$peer"
  sudo ip link set "$peer" netns "$ns"
  sudo ip addr add 198.18.0.1/30 dev "$uplink"
  sudo ip link set "$uplink" up
  sudo ip netns exec "$ns" ip addr add 198.18.0.2/30 dev "$peer"
  sudo ip netns exec "$ns" ip link set "$peer" up
  sudo ip netns exec "$ns" ip link set lo up
  sudo sysctl -w net.ipv4.ip_forward=1 >/dev/null
  sudo touch "$log"
  sudo chmod 0644 "$log"
  sudo ip netns exec "$ns" python3 "$work/s2_p1_simulated_wan_server.py" \
    "$cert" "$key" "$log" > "$work/s2-simwan-server.log" 2>&1 &
  echo "$!" > "$work/s2-simwan.pid"
  attempt=0
  while [ "$attempt" -lt 35 ]; do
    if curl -4 --noproxy '*' -fsS --max-time 2 --connect-timeout 1 \
      --cacert "$cert" \
      --resolve s2-wan.example.test:18443:198.18.0.2 \
      https://s2-wan.example.test:18443/s2-probe 2>/dev/null |
      grep -qx S2_SIMULATED_WAN_HTTPS_OK; then
      break
    fi
    attempt=$((attempt+1))
    sleep 1
  done
  test "$attempt" -lt 35
  # Discard one independent host-side TLS bootstrap request. Native guest
  # must account for exactly two later GETs: before and after disabled egress.
  sudo truncate -s 0 "$log"
  echo "S2_NATIVE_SIMULATED_WAN_NAMESPACE_ROUTE_VERIFIED_TLS=PASS"
  ;;
verify)
  test -f "$log"
  test "$(wc -l < "$log")" -eq 2
  test "$(grep -c '^S2_SIMULATED_WAN_HTTPS_GET$' "$log")" -eq 2
  echo "S2_NATIVE_SIMULATED_WAN_SERVER_INDEPENDENT_ON_OFF_ON_TWO_HITS=PASS"
  ;;
cleanup)
  if [ -e "/run/netns/$ns" ]; then
    sudo ip netns pids "$ns" | xargs -r sudo kill -TERM || :
    sudo ip netns del "$ns"
  fi
  sudo ip link del "$uplink" 2>/dev/null || :
  echo "S2_NATIVE_SIMULATED_WAN_NAMESPACE_CLEANUP=PASS"
  ;;
*) exit 89 ;;
esac
