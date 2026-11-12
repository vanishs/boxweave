#!/bin/sh
# One-use GitHub VM only. An isolated routed TLS endpoint behind its own
# namespace/veth is a **SIMULATED WAN**, never a real public Internet PASS.
set -eu
[ "${GITHUB_ACTIONS:-}" = true ] && [ "${RUNNER_ENVIRONMENT:-}" = github-hosted ]
[ "$(id -u)" -ne 0 ] && [ "$#" -eq 3 ]
mode="$1"; work="$2"; project="$3"
test "$project" = "user-$(id -u boxweave)"
bridge="$(sudo incus profile device get default eth0 network --project "$project")"
printf '%s' "$bridge" | grep -Eq '^[a-zA-Z0-9_.-]{1,15}
case "$work" in /tmp/bw-v1-s2-ghci.*) ;; *) exit 88 ;; esac
ns="bw-s2-simwan"
uplink="bws2wanhost"
peer="bws2wanpeer"
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
  sudo ip netns exec "$ns" ip route add default via 198.18.0.1
  sudo sysctl -w net.ipv4.ip_forward=1 >/dev/null
  # The native bridge ACL controls guest egress. These narrowly scoped
  # forwarding/NAT prerequisites make a real off-bridge route usable on
  # the single-use CI VM; neither rule authorizes other destinations nor
  # bypasses Incus per-NIC IPv4/IPv6 ACLs. OFF must still prove denial.
  sudo iptables -I FORWARD 1 -i "$bridge" -o "$uplink" -d 198.18.0.2/32 -j ACCEPT
  sudo iptables -I FORWARD 1 -i "$uplink" -o "$bridge" -s 198.18.0.2/32 \
    -m conntrack --ctstate ESTABLISHED,RELATED -j ACCEPT
  sudo iptables -t nat -I POSTROUTING 1 -o "$uplink" -d 198.18.0.2/32 -j MASQUERADE
  sudo touch "$log"
  sudo chmod 0644 "$log"
  sudo ip netns exec "$ns" python3 "$(pwd)/scripts/s2_p1_simulated_wan_server.py" \
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
  if [ "$attempt" -ge 35 ]; then
    echo "S2_SIMULATED_WAN_TEST_HARNESS_FAIL: no host-side verified HTTPS baseline" >&2
    sudo ip -brief address show "$uplink" >&2 || :
    sudo ip route get 198.18.0.2 >&2 || :
    sudo ip netns exec "$ns" ip -brief addr >&2 || :
    sudo ip netns exec "$ns" ip route >&2 || :
    cat "$work/s2-simwan-server.log" >&2 || :
    curl -4 --noproxy '*' -v --max-time 4 --connect-timeout 2 \
      --cacert "$cert" --resolve s2-wan.example.test:18443:198.18.0.2 \
      https://s2-wan.example.test:18443/s2-probe > /dev/null 2> "$work/s2-simwan-curl.last-error" || :
    sed -n '1,45p' "$work/s2-simwan-curl.last-error" >&2 || :
    exit 105
  fi
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
  sudo iptables -t nat -D POSTROUTING -o "$uplink" -d 198.18.0.2/32 -j MASQUERADE 2>/dev/null || :
  sudo iptables -D FORWARD -i "$uplink" -o "$bridge" -s 198.18.0.2/32 \
    -m conntrack --ctstate ESTABLISHED,RELATED -j ACCEPT 2>/dev/null || :
  sudo iptables -D FORWARD -i "$bridge" -o "$uplink" -d 198.18.0.2/32 -j ACCEPT 2>/dev/null || :
  if [ -e "/run/netns/$ns" ]; then
    sudo ip netns pids "$ns" | xargs -r sudo kill -TERM || :
    sudo ip netns del "$ns"
  fi
  sudo ip link del "$uplink" 2>/dev/null || :
  echo "S2_NATIVE_SIMULATED_WAN_NAMESPACE_CLEANUP=PASS"
  ;;
*) exit 89 ;;
esac

sudo ip link show "$bridge" >/dev/null
command -v iptables >/dev/null 2>&1
case "$work" in /tmp/bw-v1-s2-ghci.*) ;; *) exit 88 ;; esac
ns="bw-s2-simwan"
uplink="bws2wanhost"
peer="bws2wanpeer"
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
  sudo ip netns exec "$ns" python3 "$(pwd)/scripts/s2_p1_simulated_wan_server.py" \
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
  if [ "$attempt" -ge 35 ]; then
    echo "S2_SIMULATED_WAN_TEST_HARNESS_FAIL: no host-side verified HTTPS baseline" >&2
    sudo ip -brief address show "$uplink" >&2 || :
    sudo ip route get 198.18.0.2 >&2 || :
    sudo ip netns exec "$ns" ip -brief addr >&2 || :
    sudo ip netns exec "$ns" ip route >&2 || :
    cat "$work/s2-simwan-server.log" >&2 || :
    curl -4 --noproxy '*' -v --max-time 4 --connect-timeout 2 \
      --cacert "$cert" --resolve s2-wan.example.test:18443:198.18.0.2 \
      https://s2-wan.example.test:18443/s2-probe > /dev/null 2> "$work/s2-simwan-curl.last-error" || :
    sed -n '1,45p' "$work/s2-simwan-curl.last-error" >&2 || :
    exit 105
  fi
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
