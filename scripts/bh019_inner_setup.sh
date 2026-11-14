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
    # Evidence only: distinguish an actual L2 Guest profile label, UserNS
    # and AppArmor policy namespace from mere 'security.nesting=true' or
    # a successful native container start. This is read-only and cannot
    # grant inner policy-admin authority or make an unconfined parent safe.
    daemon_pid="$(systemctl show -p MainPID --value incus.service)"
    if [ -n "$daemon_pid" ] && [ "$daemon_pid" != 0 ]; then
      printf 'BH019_INNER_DAEMON_AA_LABEL='
      cat "/proc/$daemon_pid/attr/current" || echo UNAVAILABLE
      printf 'BH019_INNER_DAEMON_USERNS='
      readlink "/proc/$daemon_pid/ns/user" || echo UNAVAILABLE
    fi
    if incus exec bh019-l2-native-child -- /bin/sh -c '
      printf "BH019_L2_PID1_AA_LABEL="
      cat /proc/1/attr/current || echo UNAVAILABLE
      printf "BH019_L2_EXEC_AA_LABEL="
      cat /proc/self/attr/current || echo UNAVAILABLE
      printf "BH019_L2_USERNS="
      readlink /proc/self/ns/user || echo UNAVAILABLE
      printf "BH019_L2_USERNS_MAP="
      tr "\\n" ";" </proc/self/uid_map; echo
      printf "BH019_L2_CAPEFF="
      sed -n "s/^CapEff:[[:space:]]*//p" /proc/self/status
      if [ -r /sys/kernel/security/apparmor/.ns_name ]; then
        printf "BH019_L2_POLICY_NS="
        cat /sys/kernel/security/apparmor/.ns_name
      else
        echo "BH019_L2_POLICY_NS=UNAVAILABLE"
      fi
      if [ -r /sys/kernel/security/apparmor/.ns_stacked ]; then
        printf "BH019_L2_POLICY_STACKED="
        cat /sys/kernel/security/apparmor/.ns_stacked
      else
        echo "BH019_L2_POLICY_STACKED=UNAVAILABLE"
      fi
    '; then
      echo "BH019_L2_READONLY_AA_USERNS_WITNESS=RECORDED case=$case_name"
    else
      echo "BH019_L2_READONLY_AA_USERNS_WITNESS=UNVERIFIED case=$case_name"
    fi
    # Controlled policy-admin experiment only in the owned ephemeral GH
    # parent AppArmor namespace. The synthetic profile has NO attachment
    # and is never applied to a running command or system service.
    # This tests native policy load/remove rights, NOT L2's independent
    # namespace, confinement negative, or physical-host repair.
    kernel_release="$(uname -r)"
    echo "BH019_PARENT_KERNEL_RELEASE=$kernel_release case=$case_name"
    policy_ns="$(cat /sys/kernel/security/apparmor/.ns_name 2>/dev/null || :)"
    policy_stacked="$(cat /sys/kernel/security/apparmor/.ns_stacked 2>/dev/null || :)"
    case "$policy_ns" in
      incus-bh019-control-1114_*|incus-bh019-bpf-parent-1114_*) ;;
      *) echo "BH019_PARENT_POLICY_NS_SCOPE=BLOCKED case=$case_name"; exit 87 ;;
    esac
    if [ "$policy_stacked" != yes ]; then
      echo "BH019_PARENT_POLICY_NOT_STACKED=BLOCKED case=$case_name"
      exit 87
    fi
    policy_path="/root/bh019-ci-inert-$case_name.profile"
    test ! -e "$policy_path"
    umask 077
    cat >"$policy_path" <<EOF
profile s25-bh019-ci-inert-$case_name {
  deny /tmp/bh019-ci-unused-marker r,
}
EOF
    if ! command -v apparmor_parser >/dev/null 2>&1; then
      echo "BH019_PARENT_POLICY_ADMIN=NOT_TESTED_PARSER_MISSING case=$case_name"
    elif apparmor_parser -Q -K "$policy_path" >/tmp/bh019-policy-syntax.log 2>&1; then
      echo "BH019_PARENT_POLICY_PARSE=PASS case=$case_name"
      if timeout 15s apparmor_parser -a -K "$policy_path" >/tmp/bh019-policy-load.log 2>&1; then
        echo "BH019_PARENT_POLICY_LOAD=PASS case=$case_name"
        if timeout 15s apparmor_parser -R -K "$policy_path" >/tmp/bh019-policy-remove.log 2>&1; then
          echo "BH019_PARENT_POLICY_REMOVE=PASS case=$case_name"
        else
          echo "BH019_PARENT_POLICY_REMOVE=BLOCKED case=$case_name"
          cat /tmp/bh019-policy-remove.log
          exit 88
        fi
      else
        rc=$?
        echo "BH019_PARENT_POLICY_LOAD=DENIED_OR_FAILED case=$case_name rc=$rc"
        cat /tmp/bh019-policy-load.log
      fi
    else
      echo "BH019_PARENT_POLICY_PARSE=BLOCKED case=$case_name"
      cat /tmp/bh019-policy-syntax.log
    fi
    rm -f "$policy_path"

    # Next-depth *negative* authority control: probe the actual L2 UserNS,
    # which differs from the L1 daemon UserNS but reports the SAME inherited
    # policy namespace. No unrestricted host admin socket or policy changes.
    # Only try to load an inert, unattached profile if the L2 Guest already
    # has a parser and its observed AA namespace is this exact owned parent.
    if incus exec bh019-l2-native-child -- /bin/sh -c 'command -v apparmor_parser >/dev/null 2>&1'; then
      l2_policy="/root/bh019-ci-l2-inert-$case_name.profile"
      test ! -e "$l2_policy"
      cat >"$l2_policy" <<EOF
profile s25-bh019-ci-l2-inert-$case_name {
  deny /tmp/bh019-ci-l2-unused-marker r,
}
EOF
      incus file push "$l2_policy" "bh019-l2-native-child/root/bh019-ci-l2-inert.profile"
      rm -f "$l2_policy"
      incus exec bh019-l2-native-child -- /bin/sh -c '
        set -eu
        expected="$1"
        actual="$(cat /sys/kernel/security/apparmor/.ns_name)"
        if [ "$actual" != "$expected" ]; then
          echo "BH019_L2_POLICY_NS_SCOPE=BLOCKED"
          exit 87
        fi
        test "$(cat /sys/kernel/security/apparmor/.ns_stacked)" = yes
        file=/root/bh019-ci-l2-inert.profile
        if ! apparmor_parser -Q -K "$file" >/tmp/bh019-l2-policy-syntax.log 2>&1; then
          echo "BH019_L2_POLICY_PARSE=BLOCKED"
          cat /tmp/bh019-l2-policy-syntax.log
          exit 88
        fi
        echo "BH019_L2_POLICY_PARSE=PASS"
        if timeout 15s apparmor_parser -a -K "$file" >/tmp/bh019-l2-policy-load.log 2>&1; then
          echo "BH019_L2_POLICY_LOAD=PASS"
          if timeout 15s apparmor_parser -R -K "$file" >/tmp/bh019-l2-policy-remove.log 2>&1; then
            echo "BH019_L2_POLICY_REMOVE=PASS"
          else
            echo "BH019_L2_POLICY_REMOVE=BLOCKED"
            cat /tmp/bh019-l2-policy-remove.log
            exit 89
          fi
        else
          rc=$?
          echo "BH019_L2_POLICY_LOAD=DENIED_OR_FAILED rc=$rc"
          cat /tmp/bh019-l2-policy-load.log
        fi
        rm -f "$file"
      ' -- "$policy_ns"
    else
      echo "BH019_L2_POLICY_LOAD=NOT_TESTED_PARSER_MISSING case=$case_name"
    fi
    echo "BH019_LAYERED_APPARMOR_INDEPENDENT_POLICY=UNVERIFIED case=$case_name"
    echo "BH019_CGROUP_DEVICE_BPF_INNER_LOAD=UNVERIFIED case=$case_name"
  else
    echo "BH019_L2_START=BLOCKED_EXEC case=$case_name"
  fi
else
  echo "BH019_L2_START=BLOCKED_START case=$case_name code=$status"
  incus info --show-log bh019-l2-native-child || true
  journalctl -u incus.service -n 35 --no-pager -o cat || true
fi
