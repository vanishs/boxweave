#!/bin/sh
# Read-only L1 nested storage feasibility probe. No mount, pool creation,
# daemon restart, host filesystem change or quota waiver is performed.
set -eu
if [ "$#" -ne 1 ]; then
  echo "usage: sh scripts/s25_l1_storage_precheck.sh <exact-nested-pool-name>" >&2
  exit 80
fi
pool="$1"
case "$pool" in
  ''|.|..|*[!a-zA-Z0-9_-]*) echo "S25_L1_INVALID_POOL=BLOCKED" >&2; exit 81 ;;
esac
path="/var/lib/incus/storage-pools/$pool"
if [ ! -d "$path" ]; then
  echo "S25_L1_POOL_PATH=UNAVAILABLE"
  exit 82
fi
fstype="$(findmnt -n -o FSTYPE --target "$path")"
opts="$(findmnt -n -o OPTIONS --target "$path")"
printf 'S25_L1_STORAGE_FILESYSTEM=%s\n' "$fstype"
quota_evidence=UNVERIFIED
case "$fstype" in
  ext4|xfs)
    case ",$opts," in
      *,prjquota,*|*,pquota,*) quota_evidence=PRECONDITION_PRESENT ;;
    esac
    ;;
esac
if [ "$quota_evidence" = PRECONDITION_PRESENT ]; then
  echo "S25_L1_PROJECT_QUOTA_MOUNT=PRECONDITION_PRESENT"
  echo "S25_L1_NATIVE_DIR_QUOTA=REQUIRES_REAL_EDQUOT_AND_L2_PROOF"
else
  # Incus may log a skipped size limit; inner dir still remains usable.
  echo "S25_L1_PROJECT_QUOTA_MOUNT=UNVERIFIED"
  echo "S25_L1_NATIVE_DIR_QUOTA=UNVERIFIED_CHECK_INCUS_LOGS_AND_TEST_LIMITS"
fi
