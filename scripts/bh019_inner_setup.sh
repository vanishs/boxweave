#!/bin/sh
# Only copied into a one-use Incus parent on a GitHub-hosted Ubuntu runner.
# BPF syscall interception is configured on that parent, not from here.
set -eu
case_name="${1:-}"
case "$case_name" in control|bpf-parent) ;; *) exit 81 ;; esac
proxy="${2:-}"
case "$proxy" in
  http://10.*:18466|http://172.*:18466|http://192.168.*:18466) ;;
  *) echo BH019_UNAPPROVED_PROXY >&2; exit 82 ;;
esac
export http_proxy="$proxy" HTTP_PROXY="$proxy"
export https_proxy="$proxy" HTTPS_PROXY="$proxy"
export no_proxy=localhost,127.0.0.1 NO_PROXY=localhost,127.0.0.1
test "$(id -u)" -eq 0
. /etc/os-release
test "$ID" = ubuntu && test "$VERSION_ID" = 24.04
test ! -e /var/lib/incus
printf 'BH019_INNER_CASE=%s\n' "$case_name"
cat /proc/self/attr/current
cat /proc/self/uid_map
cat /proc/self/gid_map
if [ -e /sys/kernel/security/apparmor/.ns_name ]; then cat /sys/kernel/security/apparmor/.ns_name; fi
if [ -e /sys/kernel/security/apparmor/.ns_stacked ]; then cat /sys/kernel/security/apparmor/.ns_stacked; fi
for sources in /etc/apt/sources.list /etc/apt/sources.list.d/ubuntu.sources; do
  [ ! -f "$sources" ] && continue
  sed -i -e 's#http://archive.ubuntu.com#https://archive.ubuntu.com#g' \
    -e 's#http://security.ubuntu.com#https://security.ubuntu.com#g' "$sources"
done
if grep -ER '^(deb |URIs:).*http://' /etc/apt/sources.list /etc/apt/sources.list.d 2>/dev/null; then
  echo BH019_PLAINTEXT_APT_SOURCE_REFUSED >&2
  exit 83
fi
printf 'Acquire::https::Proxy "%s";\n' "$proxy" > /etc/apt/apt.conf.d/98-bh019-relay
apt-get -o Acquire::Retries=0 update -qq
DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends ca-certificates curl gpg dnsmasq-base >/dev/null
curl -fsSI --connect-timeout 6 --max-time 25 https://pkgs.zabbly.com/key.asc >/dev/null
echo "BH019_SCOPED_SIGNED_PACKAGE_PROXY=PASS case=$case_name"
curl -fsSL --max-time 45 https://pkgs.zabbly.com/key.asc -o /tmp/bh019-zabbly.asc
fpr="$(gpg --show-keys --with-colons /tmp/bh019-zabbly.asc | awk -F: '$1=="fpr" {print $10; exit}')"
test "$fpr" = 4EFC590696CB15B87C73A3AD82CC8797C838DCFD
install -d -m 0755 /etc/apt/keyrings
install -m 0644 /tmp/bh019-zabbly.asc /etc/apt/keyrings/bh019-zabbly.asc
cat > /etc/apt/sources.list.d/bh019-zabbly.sources <<'SIGNED'
Enabled: yes
Types: deb
URIs: https://pkgs.zabbly.com/incus/stable
Suites: noble
Components: main
Architectures: amd64
Signed-By: /etc/apt/keyrings/bh019-zabbly.asc
SIGNED
apt-get update -qq
pkg="$(apt-cache madison incus | awk '/pkgs.zabbly.com/ && $3 ~ /:7[.]6[-.]/ {print $3; exit}')"
test -n "$pkg"
DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends "incus=$pkg" "incus-client=$pkg" "incus-base=$pkg" >/dev/null
systemctl start incus.service
systemctl is-active --quiet incus.service
incus admin init --auto --storage-backend=dir
incus config set core.proxy_https="$proxy"
test "$(incus config get core.proxy_https)" = "$proxy"
incus version
echo "BH019_NESTED_SIGNED_INCUS_76=PASS case=$case_name"
# Copy only the exact already-cached parent image from the disposable
# runner. No inner image-server request: a transient upstream 503 cannot
# masquerade as an AppArmor or cgroup-device denial.
if [ ! -s /root/bh019-l2-image-metadata ]; then
  echo "BH019_LOCAL_IMAGE_METADATA_MISSING_AFTER_INSTALL=BLOCKED case=$case_name" >&2
  ls -l /root/bh019-l2-image-* 2>&1 || true
  exit 86
fi
if [ -s /root/bh019-l2-image-rootfs ]; then
  incus image import /root/bh019-l2-image-metadata /root/bh019-l2-image-rootfs \
    --alias bh019-l2-cached-image > /tmp/bh019-image-import.out 2>&1 || {
      cat /tmp/bh019-image-import.out
      echo "BH019_L2_IMAGE_IMPORT=BLOCKED case=$case_name"
      exit 85
    }
else
  incus image import /root/bh019-l2-image-metadata \
    --alias bh019-l2-cached-image > /tmp/bh019-image-import.out 2>&1 || {
      cat /tmp/bh019-image-import.out
      echo "BH019_L2_IMAGE_IMPORT=BLOCKED case=$case_name"
      exit 85
    }
fi
incus image info bh019-l2-cached-image >/dev/null
echo "BH019_L2_IMAGE_LOCAL_IMPORT=PASS case=$case_name"
set +e
incus launch bh019-l2-cached-image bh019-l2-native-child > /tmp/bh019-child-start.out 2>&1
status=$?
set -e
cat /tmp/bh019-child-start.out
if [ "$status" -eq 0 ]; then
  incus info bh019-l2-native-child
  if incus exec bh019-l2-native-child -- /bin/true; then
    echo "BH019_L2_START=PASS case=$case_name"
  else
    echo "BH019_L2_START=BLOCKED_EXEC case=$case_name"
  fi
else
  echo "BH019_L2_START=BLOCKED_START case=$case_name code=$status"
  incus info --show-log bh019-l2-native-child || true
  journalctl -u incus.service -n 35 --no-pager -o cat || true
fi
