#!/usr/bin/env bash
# S2.5 host-local read-only data collection. Inspect before running with sudo.
# Does not restart, rollback, recover, delete, unlock, or modify host/Incus.
set -uo pipefail
export LC_ALL=C
say(){ printf '%s=%s\n' "$1" "$2"; }
ok(){ if "$@" >/dev/null 2>&1; then printf YES; else printf NO; fi; }
say S25_READONLY_PROBE V1
if [ "$(id -u)" -ne 0 ]; then say ROOT_READ_ACCESS REQUIRED; exit 2; fi
say KERNEL "$(uname -r)"
say ARCH "$(uname -m)"
if command -v incus >/dev/null 2>&1; then
  say INCUS "$(incus version 2>/dev/null | tr '\n' ';' | cut -c 1-120)"
  say PROJECT_USER1001_READABLE "$(ok incus project show user-1001)"
else say INCUS UNAVAILABLE; fi
say NONADMIN_SOCKET_PRESENT "$(ok test -S /var/lib/incus/unix.socket.user)"
if id boxweave >/dev/null 2>&1; then
  groups="$(id -nG boxweave 2>/dev/null || true)"
  case " $groups " in *" incus "*) say INCUS_GROUP YES;; *) say INCUS_GROUP NO;; esac
  case " $groups " in *" incus-admin "*|*" sudo "*|*" docker "*|*" lxd "*) say FORBIDDEN_GROUP YES;; *) say FORBIDDEN_GROUP NO;; esac
else say SERVICE_USER UNAVAILABLE; fi
say BOXWEAVE_SERVICE_ACTIVE "$(ok systemctl is-active --quiet boxweave.service)"
say UNIT_USER "$(systemctl show boxweave.service -p User --value 2>/dev/null | tr -cd 'A-Za-z0-9_-' | cut -c 1-40)"
say UNIT_NO_NEW_PRIVILEGES "$(systemctl show boxweave.service -p NoNewPrivileges --value 2>/dev/null | tr -cd 'A-Za-z' | cut -c 1-20)"
bw="/usr/local/bin/boxweave"
if [ -x "$bw" ]; then
  say BINARY_SHA256 "$(sha256sum "$bw" | awk '{print $1}')"
  say BUILD_INFO "$("$bw" build-info 2>/dev/null | tr -d '\n' | cut -c 1-180)"
  if command -v python3 >/dev/null 2>&1; then
    if data="$("$bw" install-status -project user-1001 2>/dev/null)"; then
      say INSTALL_INTEGRITY PASS
      printf '%s\n' "$data" | python3 -c 'import json,sys;d=json.load(sys.stdin);print("MANIFEST_COMMIT="+str(d.get("build_commit","?")));print("MANIFEST_ACTIVE="+str(d.get("service_active","?")));print("MANIFEST_RESTART_REQUIRED="+str(d.get("restart_required","?")))' || say INSTALL_MANIFEST PARSE_ERROR
    else say INSTALL_INTEGRITY FAILED_OR_UNAVAILABLE; fi
    if data="$("$bw" native-prepare-status 2>/dev/null)"; then
      printf '%s\n' "$data" | python3 -c 'import json,sys;d=json.load(sys.stdin);print("NATIVE_PREPARE_STATE="+str(d.get("status","?")))' || say NATIVE_PREPARE_STATE PARSE_ERROR
    else say NATIVE_PREPARE_STATE NO_READABLE_JOURNAL; fi
    if data="$("$bw" native-prereq-plan -project user-1001 2>/dev/null)"; then
      printf '%s\n' "$data" | python3 -c 'import json,sys;d=json.load(sys.stdin);c=d.get("checks",[]);print("NATIVE_PREREQ_READY="+str(d.get("config_ready","?")));print("NATIVE_PREREQ_OK="+str(sum(1 for x in c if x.get("ok") is True))+"/"+str(len(c)))' || say NATIVE_PREREQ PARSE_ERROR
    else say NATIVE_PREREQ UNAVAILABLE; fi
  else say PYTHON3 UNAVAILABLE; fi
else say BINARY_SHA256 MISSING; fi
say INSTALLER_BACKUP_DIR "$(ok test -d /var/lib/boxweave-installer-backup)"
say CPU_CORES "$(nproc 2>/dev/null || printf '?')"
say MEMORY_AVAILABLE_KIB "$(awk '$1=="MemAvailable:"{print $2}' /proc/meminfo)"
for path in / /var/lib/boxweave /var/lib/incus; do
  tag="$(printf '%s' "$path" | tr '/' '_')"
  if [ -d "$path" ]; then
    say "$(printf 'CAPACITY_%s_FREE_KIB' "$tag")" "$(df -Pk "$path" | awk 'NR==2{print $4}')"
    say "$(printf 'CAPACITY_%s_FREE_INODES' "$tag")" "$(df -Pi "$path" | awk 'NR==2{print $4}')"
    say "$(printf 'CAPACITY_%s_USED_PERCENT' "$tag")" "$(df -Pk "$path" | awk 'NR==2{print $5}')"
  fi
done
say ORIGINAL_UNKNOWN_MUTATED NO
say HOST_STOP_OR_ROLLBACK_TESTED NO_READ_ONLY
say RESULT EVIDENCE_ONLY_NOT_AUTOMATIC_PASS
