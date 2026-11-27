#!/bin/sh
# One-use GitHub-hosted S2.5 LAB. No shared Incus/production effects.
set -eu
test "$GITHUB_ACTIONS" = true
test "$RUNNER_ENVIRONMENT" = github-hosted
test "$(id -u)" != 0
sudo -n true
grep -q '^ID=ubuntu$' /etc/os-release
grep -Eq '^VERSION_ID="?24\.04"?$' /etc/os-release
if [ -e /var/lib/incus ] || id boxweave >/dev/null 2>&1; then
 echo S25_LAB_BLOCKED_FOREIGN_NATIVE_STATE >&2; exit 82
fi
work="$(mktemp -d -p /tmp bw-v1-s25-lab.XXXXXX)"
chmod 0711 "$work"
mkdir -m 0700 "$work/staging" "$work/private"
proof="$RUNNER_TEMP/s25-evidence"
mkdir -p "$proof"
echo "S25_LAB_SOURCE_SHA=$(git rev-parse HEAD)"
echo S25_LAB_DISPOSABLE_HOST=PASS
case "${P1B_S25_SCOPED:-0}" in 0|1) ;; *) exit 88 ;; esac
if [ "${P1B_S25_SCOPED:-0}" = 1 ]; then
 test -z "${S5B1_MODE:-}${S3_FILE_INSTALLED_PHASE2:-}"
 test ! -e /usr/local/bin/boxweave-s25-candidate
 test "$(git rev-parse HEAD)" = "${P1B_S25_EXACT_HEAD:?}"
 test ! -e /var/lib/boxweave
 test ! -e /usr/local/bin/boxweave
 # Same isolated runner prerequisite as N2; never applied to a shared Host.
 sudo python3 - <<'P1B_S25_PARENT'
import os,stat
for name in ("/","/usr","/usr/local","/usr/local/bin"):
    s=os.lstat(name)
    assert stat.S_ISDIR(s.st_mode) and s.st_uid==0
    if name in ("/usr/local","/usr/local/bin"):
        os.chmod(name,stat.S_IMODE(s.st_mode)&~0o022)
    assert not os.lstat(name).st_mode&0o022
P1B_S25_PARENT
 case "${GITHUB_REF:-}" in
   refs/heads/release/*)
     release_version="$(python3 scripts/pr191_s2_ci_gate.py --release-version)"
     CGO_ENABLED=1 go build -trimpath -buildvcs=false -ldflags "-X main.version=$release_version -X main.commit=$(git rev-parse HEAD) -buildid=" -o "$work/staging/boxweave" ./cmd/boxweave
     ;;
   *) CGO_ENABLED=1 go build -trimpath -ldflags "-X main.version=1.0.25-s25lab -X main.commit=$(git rev-parse HEAD)" -o "$work/staging/boxweave" ./cmd/boxweave
     ;;
 esac
 sudo install -m 0755 "$work/staging/boxweave" /usr/local/bin/boxweave-s25-candidate
 cp scripts/pr191_s25_scoped_setup.sh "$work/staging/pr191_s25_scoped_setup.sh"
 cp install.sh "$work/staging/s25-install.sh"
 cp install.sh "$work/private/s25-install.sh"
 setup_helper_sha="$(sha256sum "$work/staging/pr191_s25_scoped_setup.sh" | cut -d' ' -f1)"
 install_sha="$(sha256sum "$work/staging/s25-install.sh" | cut -d' ' -f1)"
 candidate_sha="$(sha256sum "$work/staging/boxweave" | cut -d' ' -f1)"
fi

# CI package selection: unrelated optional repository is not needed.
# Required repository signatures and exact package checks are unchanged.
sudo env GITHUB_ACTIONS="$GITHUB_ACTIONS" RUNNER_ENVIRONMENT="$RUNNER_ENVIRONMENT" python3 scripts/pr191_ci_apt_sources.py
sudo apt-get -o APT::Update::Error-Mode=any -o Acquire::Retries=2 update -qq
sudo DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends ca-certificates curl gpg python3 iproute2 nftables conntrack dnsmasq-base >/dev/null
sudo install -d -m 0755 /etc/apt/keyrings
curl -fsSL --max-time 35 https://pkgs.zabbly.com/key.asc -o "$work/staging/zabbly.asc"
fp="$(gpg --show-keys --with-colons "$work/staging/zabbly.asc" | awk -F: '$1=="fpr"{print $10;exit}')"
test "$fp" = 4EFC590696CB15B87C73A3AD82CC8797C838DCFD
sudo install -m 0644 "$work/staging/zabbly.asc" /etc/apt/keyrings/s25-zabbly.asc
sudo tee /etc/apt/sources.list.d/s25-zabbly.sources >/dev/null <<'S25_APT_SOURCE'
Enabled: yes
Types: deb
URIs: https://pkgs.zabbly.com/incus/stable
Suites: noble
Components: main
Architectures: amd64
Signed-By: /etc/apt/keyrings/s25-zabbly.asc
S25_APT_SOURCE
sudo apt-get -o APT::Update::Error-Mode=any -o Acquire::Retries=2 update -qq
pkg="$(apt-cache madison incus | awk '/pkgs.zabbly.com/ && $3 ~ /:7[.]6[-.]/ {print $3;exit}')"
test -n "$pkg"
if [ "${P1B_S25_SCOPED:-0}" = 1 ]; then
 sudo env P1B_S25_SCOPED=1 BW_S25_SETUP_LAYER=outer GITHUB_ACTIONS=true RUNNER_ENVIRONMENT=github-hosted \
   sh scripts/pr191_s25_scoped_setup.sh bootstrap /usr/local/bin/boxweave-s25-candidate "$candidate_sha" "$pkg" "$work/private"
else
sudo DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends "incus=$pkg" "incus-client=$pkg" "incus-base=$pkg" >/dev/null
sudo systemctl start incus.service
sudo systemctl is-active --quiet incus.service
test -z "$(sudo incus list --all-projects --format csv -c n)"
sudo incus admin init --auto --storage-backend=dir
sudo useradd --create-home --gid incus --shell /usr/sbin/nologin boxweave
fi
echo "S25_LAB_INCUS_APT=$pkg"
echo S25_LAB_NATIVE_INCUS_7_6_INITIALIZED=PASS
uid="$(id -u boxweave)"
project="user-$uid"
sudo runuser -u boxweave -- env -u INCUS_CONF -u INCUS_CONFIG_DIR -u XDG_CONFIG_HOME HOME=/home/boxweave XDG_CONFIG_HOME=/home/boxweave/.config incus project list --format csv >/dev/null
test "$(sudo incus project get "$project" restricted)" = true
if sudo runuser -u boxweave -- env -u INCUS_CONF -u INCUS_CONFIG_DIR -u XDG_CONFIG_HOME HOME=/home/boxweave XDG_CONFIG_HOME=/home/boxweave/.config incus project show default >/dev/null 2>&1; then
 echo S25_FAIL_FOREIGN_PROJECT_ACCESS >&2; exit 83
fi
echo S25_LAB_FOREIGN_NATIVE_PROJECT_DENIED=PASS
sudo incus image copy images:ubuntu/24.04 local: --target-project "$project" --alias boxweave-ubuntu-24.04 >/dev/null
if [ "${P1B_S25_SCOPED:-0}" = 1 ]; then
 # Reuse the exact image already acquired by the authorized disposable L0.
 # No new CONNECT hosts, parent socket, image server or production capability.
 image_fingerprint="$(sudo incus query "/1.0/images/aliases/boxweave-ubuntu-24.04?project=$project" | python3 -c 'import json,sys;print(json.load(sys.stdin)["target"])')"
 printf '%s\n' "$image_fingerprint" | grep -Ex '[0-9a-f]{64}' >/dev/null
 test ! -e "$work/staging/s25-ubuntu.meta"
 test ! -e "$work/staging/s25-ubuntu.meta.root"
 # Incus 7.6 split-image export to explicit nonexistent file writes
 # metadata and <file>.root. Refuse a different format rather than guessing.
 sudo incus image export "$image_fingerprint" "$work/staging/s25-ubuntu.meta" --project "$project" --quiet
 sudo test -s "$work/staging/s25-ubuntu.meta"
 sudo test -s "$work/staging/s25-ubuntu.meta.root"
 sudo chown "$(id -u):$(id -g)" "$work/staging/s25-ubuntu.meta" "$work/staging/s25-ubuntu.meta.root"
 chmod 0600 "$work/staging/s25-ubuntu.meta" "$work/staging/s25-ubuntu.meta.root"
 image_meta_sha="$(sha256sum "$work/staging/s25-ubuntu.meta" | cut -d' ' -f1)"
 image_root_sha="$(sha256sum "$work/staging/s25-ubuntu.meta.root" | cut -d' ' -f1)"
 test "$(cat "$work/staging/s25-ubuntu.meta" "$work/staging/s25-ubuntu.meta.root" | sha256sum | cut -d' ' -f1)" = "$image_fingerprint"
 printf 'image_fingerprint=%s\nimage_metadata_sha256=%s\nimage_rootfs_sha256=%s\n' "$image_fingerprint" "$image_meta_sha" "$image_root_sha" > "$proof/image-identity.txt"
 echo P1B_S25_OUTER_EXACT_IMAGE_EXPORTED=PASS
fi

net="incusbr-$uid"
acl="bw-s25-standalone-owned"
sudo incus network show "$net" >/dev/null
if [ "${P1B_S25_SCOPED:-0}" = 1 ]; then
  test "$(sha256sum "$work/staging/boxweave" | cut -d' ' -f1)" = "$candidate_sha"
elif [ -n "${S5B1_MODE:-}" ]; then
  test "$(git rev-parse HEAD)" = a3e02b4750edcabb80427e3311a19614d82b0029
  CGO_ENABLED=1 go build -trimpath -buildvcs=false -ldflags "-X main.version=1.0.0-rc.10 -X main.commit=$(git rev-parse HEAD) -buildid=" -o "$work/staging/boxweave" ./cmd/boxweave
else
  CGO_ENABLED=1 go build -trimpath -ldflags "-X main.version=1.0.25-s25lab -X main.commit=$(git rev-parse HEAD)" -o "$work/staging/boxweave" ./cmd/boxweave
fi
CGO_ENABLED=1 go build -trimpath -o "$work/staging/s25-lab-agent" ./scripts/s25_lab_agent.go
cp scripts/s25_inner_setup.sh "$work/staging/s25_inner_setup.sh"
cp scripts/s25_simwan6.py "$work/staging/s25_simwan6.py"
chmod 0755 "$work/staging/boxweave" "$work/staging/s25-lab-agent"
binary_sha="$(sha256sum "$work/staging/boxweave" | awk '{print $1}')"
echo "S25_LAB_CANDIDATE_SHA256=$binary_sha"
agent_sha="$(sha256sum "$work/staging/s25-lab-agent" | awk '{print $1}')"
setup_sha="$(sha256sum "$work/staging/s25_inner_setup.sh" | awk '{print $1}')"
simwan_sha="$(sha256sum "$work/staging/s25_simwan6.py" | awk '{print $1}')"
if [ "${P1B_S25_SCOPED:-0}" = 0 ]; then
 sudo install -m 0755 "$work/staging/boxweave" /usr/local/bin/boxweave-s25-candidate
fi

printf 'source_commit=%s\ncandidate_binary_sha256=%s\nagent_binary_sha256=%s\ninner_setup_sha256=%s\nincus_apt=%s\n' "$(git rev-parse HEAD)" "$binary_sha" "$agent_sha" "$setup_sha" "$pkg" > "$proof/identity.txt"
printf 'simwan6_helper_sha256=%s\n' "$simwan_sha" >> "$proof/identity.txt"

if [ "${P1B_S25_SCOPED:-0}" = 1 ]; then
 sudo env P1B_S25_SCOPED=1 BW_S25_SETUP_LAYER=outer GITHUB_ACTIONS=true RUNNER_ENVIRONMENT=github-hosted BW_S25_INSTALL_SHA="$install_sha" \
   sh scripts/pr191_s25_scoped_setup.sh install /usr/local/bin/boxweave-s25-candidate "$binary_sha" "$project" "$work/private"
 printf 'scoped_helper_sha256=%s\ninstaller_sha256=%s\n' "$setup_helper_sha" "$install_sha" >> "$proof/identity.txt"
else
sudo /usr/local/bin/boxweave-s25-candidate native-prepare-plan -project "$project" -pool default -network "$net" -acl "$acl" > "$work/private/plan.json"
sha="$(python3 -c 'import json,sys;print(json.load(open(sys.argv[1]))["review_sha256"])' "$work/private/plan.json")"
if sudo /usr/local/bin/boxweave-s25-candidate native-prepare-apply -project "$project" -pool default -network "$net" -acl "$acl" -native-prepare-sha256 0000000000000000000000000000000000000000000000000000000000000000 -approve-local >/dev/null 2>&1; then
 echo S25_FAIL_WRONG_PREP_SHA_ACCEPTED >&2;exit 84
fi
sudo /usr/local/bin/boxweave-s25-candidate native-prepare-apply -project "$project" -pool default -network "$net" -acl "$acl" -native-prepare-sha256 "$sha" -approve-local > "$work/private/applied.json"
sudo /usr/local/bin/boxweave-s25-candidate native-prereq-plan -project "$project" > "$proof/native-prereq.json"
python3 - "$proof/native-prereq.json" <<'S25_PREREQ_ASSERT'
import sys,json
d=json.load(open(sys.argv[1]))
assert d["config_ready"] and len(d["checks"])==8 and all(x["ok"] for x in d["checks"]),d
S25_PREREQ_ASSERT
echo S25_LAB_NATIVE_PREREQ_EIGHT_CHECKS=PASS

sudo /usr/local/bin/boxweave-s25-candidate install-stage -project "$project" -source-binary /usr/local/bin/boxweave-s25-candidate -expect-sha256 "$binary_sha" -approve-stage > "$work/private/stage.json"
fi
for name in admin1 admin2 admin3 peer; do
 sudo runuser -u boxweave -- env -u INCUS_CONF -u INCUS_CONFIG_DIR -u XDG_CONFIG_HOME HOME=/home/boxweave XDG_CONFIG_HOME=/home/boxweave/.config /usr/local/bin/boxweave token-create > "$work/private/$name.secret"
 chmod 0600 "$work/private/$name.secret"
done
if [ "${P1B_S25_SCOPED:-0}" = 0 ]; then
 sudo /usr/local/bin/boxweave install-activate -project "$project" -approve-local > "$work/private/activated.json"
fi
# Doctor is observation-only. Explicit token creation initializes the ledger;
# activate the exact staged service before checking installed readiness.
sudo runuser -u boxweave -- env -u INCUS_CONF -u INCUS_CONFIG_DIR -u XDG_CONFIG_HOME HOME=/home/boxweave XDG_CONFIG_HOME=/home/boxweave/.config /usr/local/bin/boxweave doctor -project "$project" > "$proof/doctor.json" || {
 python3 - "$proof/doctor.json" <<'S25_DOCTOR_FAILURE'
import json,sys
s=json.load(open(sys.argv[1]))
d=s.get("diagnostic",{})
print("S25_DOCTOR_FAILED", {k:d.get(k,{}).get("reason") for k in ("database","native","mcp")}, "native_error",s.get("native",{}).get("error_class"))
S25_DOCTOR_FAILURE
 exit 1
}
python3 - "$proof/doctor.json" <<'S25_DOCTOR_ASSERT'
import json,sys
s=json.load(open(sys.argv[1]))
assert s["ready"] and s["native"]["ready"],s
S25_DOCTOR_ASSERT
sudo systemctl is-active --quiet boxweave.service
test "$(curl -s -o /dev/null -w '%{http_code}' --max-time 5 -X POST http://127.0.0.1:18650/mcp -H 'Content-Type: application/json' -d '{}')" = 401
echo S25_LAB_LOOPBACK_MCP_UNAUTH_DENIAL=PASS
# Independent HTTP boundary negatives, no issued token material.
for path in /not-mcp /; do
 code="$(curl -sS -o /dev/null -w '%{http_code}' --max-time 5 -X POST    "http://127.0.0.1:18650$path" -H 'Content-Type: application/json' -d '{}')"
 test "$code" = 404
done
for field in 'Origin: https://attacker.invalid' 'Host: attacker.invalid'; do
 code="$(curl -sS -o /dev/null -w '%{http_code}' --max-time 5 -X POST    http://127.0.0.1:18650/mcp -H "$field"    -H 'Content-Type: application/json' -d '{}')"
 test "$code" = 403
done
code="$(curl -sS -o /dev/null -w '%{http_code}' --max-time 5 -X POST  http://127.0.0.1:18650/mcp -H 'Authorization: Bearer definitely-invalid'  -H 'Content-Type: application/json' -d '{}')"
test "$code" = 401
echo S25_HTTP_BAD_HOST_ORIGIN_PATH_AND_TOKEN_DENIAL=PASS

# S5B1 test-only branch, using the same guarded disposable VM and exact
# installed product. This does not alter the ordinary S25/S3 scenarios.
if [ -n "${S5B1_MODE:-}" ]; then
  sh scripts/s5b1_installed_closeout.sh "$work" "$project" "$net" "$acl" "$binary_sha"
  exit 0
fi


# S3-only isolated installed File Tool acceptance entry. Reuse the existing
# pristine GitHub runner's signed native preparation, restricted nonroot
# installed systemd/SQLite MCP, then exit before the old S2.5 deep LAB.
if [ "$(printenv S3_FILE_INSTALLED_PHASE2 2>/dev/null || true)" = true ]; then
  test "$GITHUB_ACTIONS" = true
  test "$RUNNER_ENVIRONMENT" = github-hosted
  CGO_ENABLED=1 go build -trimpath -o "$work/staging/s3-file-installed-agent" ./scripts/s3_file_installed_probe.go
  export BW_S3_MCP_URL=http://127.0.0.1:18650/mcp
  export BW_S3_ADMIN_TOKEN_FILE="$work/private/admin1.secret"
  export BW_S3_PEER_TOKEN_FILE="$work/private/peer.secret"
  export BW_S3_FILE_DISPOSABLE=approved-ephemeral-vm
  echo "S3_INSTALLED_TEST_SOURCE_SHA=$(git rev-parse HEAD)"
  installed_bin_sha="$(sudo sha256sum /usr/local/bin/boxweave | awk '{print $1}')"
  test "$installed_bin_sha" = "$binary_sha"
  echo "S3_INSTALLED_EXACT_BINARY_SHA256=$installed_bin_sha"
  if [ "$(printenv S3_FILE_INSTALLED_PHASE3 2>/dev/null || true)" = true ]; then
    # Same installed candidate & exactly one supervised original Agent.
    # Same strict HTTPS-only relay used by the already-accepted S2.5 lab:
    # the disposable Guest has no direct public WAN package permission.
    gateway="$(sudo incus network get "$net" ipv4.address | cut -d/ -f1)"
    case "$gateway" in
      10.*|172.*|192.168.*) ;;
      *) echo S3_PACKAGE_RELAY_GATEWAY_NOT_PRIVATE >&2; exit 71;;
    esac
    export BW_S3_PACKAGE_PROXY_URL="http://$gateway:18466"
    python3 scripts/s25_package_proxy.py "$gateway" 18466 \
      > "$work/private/s3-package-proxy.log" 2>&1 &
    s3_pkg_relay_pid=$!
    trap 'kill "$s3_pkg_relay_pid" 2>/dev/null || :; wait "$s3_pkg_relay_pid" 2>/dev/null || :' EXIT
    s3_relay_ready=no
    for attempt in 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15; do
      if curl -4 -fsSI --noproxy '' --proxy "$BW_S3_PACKAGE_PROXY_URL" \
          --connect-timeout 2 --max-time 10 https://pkgs.zabbly.com/key.asc \
          >/dev/null 2>&1; then
        s3_relay_ready=yes
        break
      fi
      sleep 1
    done
    test "$s3_relay_ready" = yes
    echo S3_PHASE3_SCOPED_HTTPS_PACKAGE_RELAY=PASS
    export BW_S3_PHASE3_MODE=stage
    export BW_S3_PHASE3_CHECKPOINT="$work/private/s3-phase3-original.json"
    if [ "${BW_S5B_TC015:-false}" = true ]; then
      handoff="$work/private/s5b-handoff"
      mkdir -m0700 "$handoff"
      git init -q --bare --initial-branch=main "$handoff/remote.git"
      git init -q -b main "$handoff/a"
      git -C "$handoff/a" config user.name "S5B Actor A fixture"
      git -C "$handoff/a" config user.email "actor-a@example.invalid"
      export BW_S5B_HANDOFF_CHECKPOINT="$handoff/a/checkpoint.json"
    fi
    if ! "$work/staging/s3-file-installed-agent"; then
      # Diagnostic only on disposable CI runner: the original Tool error is
      # preserved. Do NOT redispatch a native mutation or alter the Guest.
      echo S3_PHASE3_ORIGINAL_MCP_FAILURE_NATIVE_READ_ONLY_DIAGNOSTIC
      sudo incus config get s3-file-phase2-ci volatile.uuid --project "$project" || :
      sudo incus exec s3-file-phase2-ci --project "$project" -- /bin/sh -c '
        if test -d /root/s3-file-phase2; then
          echo S3_PHASE3_NATIVE_GUEST_FIXTURE_DIR_PRESENT
        else
          echo S3_PHASE3_NATIVE_GUEST_FIXTURE_DIR_MISSING
        fi
        if test -f /root/s3-file-phase2/victim; then
          echo S3_PHASE3_NATIVE_GUEST_VICTIM_PRESENT
        else
          echo S3_PHASE3_NATIVE_GUEST_VICTIM_MISSING
        fi
      ' || :
      sudo incus file pull s3-file-phase2-ci/root/s3-file-phase2/victim \
        "$work/staging/s3-phase3-failed-victim-read-only" --project "$project" || :
      exit 83
    fi
    if [ "${BW_S5B_TC015:-false}" = true ]; then
      test -s "$BW_S5B_HANDOFF_CHECKPOINT"
      git -C "$handoff/a" add checkpoint.json
      git -C "$handoff/a" commit -qm "record Actor A failing original"
      git -C "$handoff/a" push -q "$handoff/remote.git" HEAD:main
      initial_revision="$(git -C "$handoff/a" rev-parse HEAD)"
      git clone -q --branch main "$handoff/remote.git" "$handoff/b"
      test "$(git -C "$handoff/b" rev-parse HEAD)" = "$initial_revision"
      git -C "$handoff/b" config user.name "S5B Actor B fixture"
      git -C "$handoff/b" config user.email "actor-b@example.invalid"
      export BW_S5B_HANDOFF_CHECKPOINT="$handoff/b/checkpoint.json"
      export BW_S3_PHASE3_MODE=handoff
      "$work/staging/s3-file-installed-agent"
      test -s "$BW_S3_PHASE3_CHECKPOINT"
      # Independent verifier reads only nonsecret original identity fields.
      sudo python3 - "$handoff/a/checkpoint.json" "$BW_S3_PHASE3_CHECKPOINT" <<'S5B_ACTOR_WITNESS'
import json, sqlite3, sys
before=json.load(open(sys.argv[1])); after=json.load(open(sys.argv[2]))
for k in ("box_uuid","setup_op","fail_op","native_setup_op","native_fail_op","initial_git_commit"):
 assert before[k]==after[k], k
with sqlite3.connect("file:/var/lib/boxweave/ledger.sqlite?mode=ro",uri=True) as db:
 rows=[]
 for k in ("setup","fail","fix","pass"):
  row=db.execute("select actor_id,layer,box_uuid,native_operation_id,state from operations where id=?",(after[k+"_op"],)).fetchone()
  assert row and row[2]==after["box_uuid"] and row[3]==after["native_"+k+"_op"] and row[4]=="SUCCEEDED",k
  rows.append(row)
 assert rows[0][0]==rows[1][0] and rows[2][0]==rows[3][0] and rows[0][0]!=rows[2][0]
 assert len({r[1] for r in rows})==1
print("S5B_TC015_DISTINCT_AUTHENTICATED_ACTORS_AND_NATIVE_ORIGINALS=PASS")
S5B_ACTOR_WITNESS
      cp "$BW_S3_PHASE3_CHECKPOINT" "$handoff/b/checkpoint.json"
      git -C "$handoff/b" add checkpoint.json
      git -C "$handoff/b" commit -qm "Actor B verified repaired original"
      git -C "$handoff/b" push -q origin HEAD:main
      winner="$(git -C "$handoff/b" rev-parse HEAD)"
      printf 'stale review fixture\n' > "$handoff/a/review.txt"
      git -C "$handoff/a" add review.txt
      git -C "$handoff/a" commit -qm "stale Actor A competing checkpoint"
      if git -C "$handoff/a" push "$handoff/remote.git" HEAD:main >"$handoff/stale-push.log" 2>&1; then
        echo S5B_TC016_UNEXPECTED_STALE_WRITE >&2; exit 95
      fi
      test "$(git --git-dir="$handoff/remote.git" rev-parse refs/heads/main)" = "$winner"
      git --git-dir="$handoff/remote.git" show refs/heads/main:checkpoint.json | cmp - "$BW_S3_PHASE3_CHECKPOINT"
      echo S5B_TC016_EXTERNAL_GIT_STALE_WRITER_REFUSED_WINNER_PRESERVED=PASS
    fi
    test -s "$BW_S3_PHASE3_CHECKPOINT"
    # Restart only the installed service on this disposable GitHub runner.
    # A separate official MCP client observes the SAME original operation IDs;
    # it cannot POST a second Git command or reapply a file change.
    sudo systemctl restart boxweave.service
    sudo systemctl is-active --quiet boxweave.service
    s3_restart_ready=no
    for attempt in 1 2 3 4 5 6 7 8 9 10; do
      status="$(curl -s -o /dev/null -w '%{http_code}' --max-time 2 -X POST http://127.0.0.1:18650/mcp -H 'Content-Type: application/json' -d '{}' || :)"
      if [ "$status" = 401 ]; then
        s3_restart_ready=yes
        break
      fi
      sleep 1
    done
    test "$s3_restart_ready" = yes
    export BW_S3_PHASE3_MODE=verify
    "$work/staging/s3-file-installed-agent"
    echo S3_PHASE3_RESTART_FRESH_MCP_ORIGINAL_OPERATIONS_VERIFIED=PASS
    # Independent host-side Incus CLI is strictly read-only: check original
    # UUID and final Git bytes/HEAD, never inject an alternate Guest edit.
    native_uuid="$(sudo incus config get s3-file-phase2-ci volatile.uuid --project "$project")"
    test -n "$native_uuid"
    saved_uuid="$(python3 - "$BW_S3_PHASE3_CHECKPOINT" <<'S3_CP_UUID'
import json,sys
x=json.load(open(sys.argv[1]))
assert x["native_box_name"] == "s3-file-phase2-ci"
assert len(x["box_uuid"]) == 36
assert len(x["initial_git_commit"]) == 40
assert len(x["final_git_commit"]) == 40
assert x["initial_git_commit"] != x["final_git_commit"]
assert x["final_git_clean"] is True
for key in ["native_setup_op","native_fail_op","native_fix_op","native_pass_op"]:
    assert len(x[key]) == 36
print(x["box_uuid"])
S3_CP_UUID
)"
    test "$saved_uuid" = "$native_uuid"
    sudo incus file pull s3-file-phase2-ci/root/s3-guest-git/calc.py \
      "$work/staging/s3-phase3-native-calc.py" --project "$project"
    sudo incus file pull s3-file-phase2-ci/root/s3-guest-git/test_calc.py \
      "$work/staging/s3-phase3-native-test.py" --project "$project"
    sudo chown "$(id -u):$(id -g)" "$work/staging/s3-phase3-native-calc.py" "$work/staging/s3-phase3-native-test.py"
    native_git_head="$(sudo incus exec s3-file-phase2-ci --project "$project" -- \
      git -C /root/s3-guest-git rev-parse HEAD)"
    native_git_count="$(sudo incus exec s3-file-phase2-ci --project "$project" -- \
      git -C /root/s3-guest-git rev-list --count HEAD)"
    test "$native_git_count" = 2
    native_git_dirty="$(sudo incus exec s3-file-phase2-ci --project "$project" -- \
      git -C /root/s3-guest-git status --porcelain)"
    test -z "$native_git_dirty"
    echo S3_PHASE3_INDEPENDENT_NATIVE_GIT_WORKTREE_CLEAN=PASS
    python3 - "$BW_S3_PHASE3_CHECKPOINT" "$work/staging/s3-phase3-native-calc.py" \
      "$work/staging/s3-phase3-native-test.py" "$native_git_head" <<'S3_VERIFY_NATIVE'
import hashlib,json,sys
cp=json.load(open(sys.argv[1]))
source=open(sys.argv[2],"rb").read()
test_source=open(sys.argv[3],"rb").read()
assert source == b"def double(value):\n    return value * 2\n"
assert hashlib.sha256(source).hexdigest() == cp["source_sha256"]
assert b"self.assertEqual(double(2), 4)" in test_source
assert sys.argv[4] == cp["final_git_commit"] != cp["initial_git_commit"]
assert cp["final_git_clean"] is True
for field in ["setup_op","fail_op","fix_op","pass_op","file_create_op"]:
 assert len(cp[field]) == 32,field
print("S3_PHASE3_INDEPENDENT_NATIVE_SOURCE_GIT_HEAD_AND_ORIGINAL_IDS=PASS")
S3_VERIFY_NATIVE
    export BW_S3_PHASE3_MODE=cleanup
    "$work/staging/s3-file-installed-agent"
    if [ "${BW_S5B_TC015:-false}" = true ]; then
      echo S5B_TC015_TWO_AUTHENTICATED_SDK_PROCESSES_NATIVE_VERIFIED_AND_CLEANUP=PASS
    else
      echo S3_PHASE3_SINGLE_AGENT_GIT_NATIVE_VERIFIED_AND_MCP_CLEANUP=PASS
    fi
  else
    "$work/staging/s3-file-installed-agent"
  fi
  if sudo incus info s3-file-phase2-ci --project "$project" >/dev/null 2>&1; then
    echo S3_INSTALLED_CLEANUP_NATIVE_BOX_STILL_PRESENT >&2
    exit 91
  fi
  echo S3_INSTALLED_INDEPENDENT_NATIVE_BOX_ABSENCE=PASS
  echo S3_INSTALLED_SCOPED_HTTP_MCP_NATIVE_INTEGRATION=PASS
  exit 0
fi

# Read-only binary staging on the disposable private Incus bridge only;
# credentials and SQLite are in a different 0700 directory.
gateway="$(sudo incus network get "$net" ipv4.address | cut -d/ -f1)"
test -n "$gateway"
port=18465
proxy_port=18466
python3 -m http.server "$port" --bind "$gateway" --directory "$work/staging" > "$work/private/stage-http.log" 2>&1 &
stage_pid=$!
# LAB-only bridge relay allows only explicitly listed public HTTPS hosts.
python3 scripts/s25_package_proxy.py "$gateway" "$proxy_port" > "$work/private/proxy.log" 2>&1 &
proxy_pid=$!
trap 'if [ -f "$work/private/outer-checkpoint.json" ]; then cp "$work/private/outer-checkpoint.json" "$proof/outer-checkpoint.json" || :; fi; if [ -e "$work/private/simwan6/simwan6.json" ]; then sudo env BW_S25_SIMWAN_LAB=1 python3 scripts/s25_simwan6.py stop --workdir "$work/private/simwan6" || :; fi; kill "$stage_pid" "$proxy_pid" 2>/dev/null || :; wait "$stage_pid" "$proxy_pid" 2>/dev/null || :' EXIT
i=0
while [ "$i" -lt 30 ]; do
 if curl -fsSI --connect-timeout 1 --max-time 2 "http://$gateway:$port/boxweave" >/dev/null 2>&1;then break;fi
 i=$((i+1));sleep 1
done
test "$i" -lt 30
echo S25_LAB_PRIVATE_BRIDGE_SHA_BOUND_ARTIFACTS_READY=PASS
i=0
while [ "$i" -lt 15 ]; do
 if curl -4 -fsSI --proxy "http://$gateway:$proxy_port" --connect-timeout 2    --max-time 8 https://pkgs.zabbly.com/key.asc >/dev/null 2>&1; then break; fi
 i=$((i+1));sleep 1
done
test "$i" -lt 15
if curl -fsS -I --proxy "http://$gateway:$proxy_port" --noproxy ''   --max-time 4 https://127.0.0.1/ >/dev/null 2>&1; then
 echo S25_LAB_FAIL_PACKAGE_PROXY_ALLOWED_LOOPBACK >&2;exit 87
fi
echo S25_LAB_SCOPED_PACKAGE_PROXY_HTTPS_AND_DENIAL=PASS

sudo env BW_S25_SIMWAN_LAB=1 python3 scripts/s25_simwan6.py start --bridge "$net" --workdir "$work/private/simwan6"
sudo cp "$work/private/simwan6/simwan6.crt" "$work/staging/s25_simwan6.crt"
sudo chown "$(id -u):$(id -g)" "$work/staging/s25_simwan6.crt"
chmod 0644 "$work/staging/s25_simwan6.crt"
echo S25_L0_SIMWAN6_TLS_NAMESPACE_READY=PASS

if [ "${P1B_S25_SCOPED:-0}" = 1 ]; then
 export P1B_S25_SCOPED=1 BW_S25_SCOPED_HELPER_SHA="$setup_helper_sha" BW_S25_INSTALL_SHA="$install_sha"
 export BW_S25_IMAGE_FINGERPRINT="$image_fingerprint" BW_S25_IMAGE_META_SHA="$image_meta_sha" BW_S25_IMAGE_ROOT_SHA="$image_root_sha"
fi
export BW_S25_URL=http://127.0.0.1:18650/mcp
export BW_S25_PROJECT="$project"
export BW_S25_OUTER_DB=/var/lib/boxweave/ledger.sqlite
export BW_S25_ADMIN1_FILE="$work/private/admin1.secret"
export BW_S25_ADMIN2_FILE="$work/private/admin2.secret"
export BW_S25_ADMIN3_FILE="$work/private/admin3.secret"
export BW_S25_PEER_FILE="$work/private/peer.secret"
export BW_S25_CHECKPOINT="$work/private/outer-checkpoint.json"
export BW_S25_STAGE_URL="http://$gateway:$port"
export BW_S25_PACKAGE_PROXY_URL="http://$gateway:$proxy_port"
export BW_S25_SETUP_SHA256="$setup_sha"
export BW_S25_AGENT_SHA256="$agent_sha"
export BW_S25_BINARY_SHA256="$binary_sha"
export BW_S25_SIMWAN6_HELPER_SHA="$simwan_sha"
export BW_S25_SIMWAN6_URL='https://[2001:db8:25:2026::2]:18479/witness'
export BW_S25_SIMWAN6_CERT_FILE="$work/staging/s25_simwan6.crt"
export BW_S25_SIMWAN6_WORKDIR="$work/private/simwan6"
"$work/staging/s25-lab-agent" admit
"$work/staging/s25-lab-agent" recover
python3 scripts/s25_independent_verifier.py l1
"$work/staging/s25-lab-agent" wan6
python3 scripts/s25_independent_verifier.py wan6outer
"$work/staging/s25-lab-agent" wan6-clean
python3 scripts/s25_independent_verifier.py wan6outerclean
cp "$work/private/outer-checkpoint.json" "$proof/outer-checkpoint.json"
"$work/staging/s25-lab-agent" double-agent
python3 scripts/s25_independent_verifier.py agents
# One-shot *real* lost MCP HTTP ACK after native Exec dispatch:
# distinct Admin3 Agent has its response deliberately withheld only
# AFTER actual candidate HTTP response; Admin2 independently recovers
# the single original via persisted work/agent refs and native/SQLite V.
"$work/staging/s25-lab-agent" r12-lost-reply-admit
"$work/staging/s25-lab-agent" r12-lost-reply-recover
python3 scripts/s25_independent_verifier.py r12-lost-reply
admin3_id="$(sed -n 's/^id=\([0-9a-f]*\) .*/\1/p' "$work/private/admin3.secret")"
test -n "$admin3_id"
sudo runuser -u boxweave -- env -u INCUS_CONF -u INCUS_CONFIG_DIR -u XDG_CONFIG_HOME HOME=/home/boxweave XDG_CONFIG_HOME=/home/boxweave/.config /usr/local/bin/boxweave token-revoke -id "$admin3_id" >/dev/null
python3 - "$work/private/admin3.secret" <<'S25_R12_REVOKED_ADMIN'
import urllib.error,urllib.request,sys
credential=next(x[6:].strip() for x in open(sys.argv[1]) if x.startswith("token="))
req=urllib.request.Request("http://127.0.0.1:18650/mcp",data=b"{}",
 headers={"Authorization":"Bearer "+credential,"Content-Type":"application/json"},method="POST")
try:
 urllib.request.urlopen(req,timeout=8)
except urllib.error.HTTPError as e:
 assert e.code==401,e.code
else:
 raise AssertionError("revoked Admin still had MCP HTTP authority")
print("S25_R12_REVOKED_ORIGINAL_AGENT_HTTP_401=PASS")
S25_R12_REVOKED_ADMIN
"$work/staging/s25-lab-agent" r12-lost-reply-post-revoke
python3 scripts/s25_independent_verifier.py r12-lost-reply
echo S25_R12_REAL_LOST_MCP_ACK_TWO_AGENT_ROTATE_NO_REPLAY=PASS
"$work/staging/s25-lab-agent" quiet-long-poll
python3 scripts/s25_independent_verifier.py quiet
cp "$work/private/outer-checkpoint.json" "$proof/outer-checkpoint.json"
# Installer/runtime durability requires *same SQLite* surviving actual
# systemd service restart, independently recovered by separate SDK process.
sudo systemctl restart boxweave.service
sudo systemctl is-active --quiet boxweave.service
# Type=simple systemd ACTIVE alone proves a process was started; the MCP
# loopback socket may still be unbound. Re-read the actual auth boundary
# before dispatching any Agent request after this deliberate restart.
restart_ready=no
restart_attempt=0
while [ "$restart_attempt" -lt 40 ]; do
 sudo systemctl is-active --quiet boxweave.service
 code="$(curl -s -o /dev/null -w '%{http_code}' --max-time 2 -X POST   http://127.0.0.1:18650/mcp -H 'Content-Type: application/json' -d '{}' || :)"
 if [ "$code" = 401 ]; then restart_ready=yes; break; fi
 restart_attempt=$((restart_attempt+1))
 sleep 1
done
if [ "$restart_ready" != yes ]; then
 echo S25_RESTART_HTTP_AUTH_BOUNDARY_UNREADY=BLOCKED >&2
 exit 89
fi
echo S25_RESTART_AUTHENTIC_HTTP_READY=PASS
"$work/staging/s25-lab-agent" post-restart
python3 scripts/s25_independent_verifier.py agents
echo S25_ONE_USE_CONTROLLER_RESTART_NATIVE_SQLITE_RECONCILED=PASS
# Round 2 fault: kill the exact disposable L0 BoxWeave systemd main process,
# NOT Incus and not a shared/physical host. Original L1 and peer lifecycle
# receipts must remain immutable and be independently recovered after the
# actual process has been replaced. Active systemd alone is not HTTP ready.
old_main="$(sudo systemctl show boxweave.service -p MainPID --value)"
test "$old_main" -gt 1
sudo systemctl kill --kill-whom=main --signal=SIGKILL boxweave.service
kill_recovered=no
attempt=0
while [ "$attempt" -lt 40 ]; do
 new_main="$(sudo systemctl show boxweave.service -p MainPID --value)"
 state="$(sudo systemctl show boxweave.service -p ActiveState --value)"
 code="$(curl -s -o /dev/null -w '%{http_code}' --max-time 2 -X POST http://127.0.0.1:18650/mcp -H 'Content-Type: application/json' -d '{}' || :)"
 if [ "$new_main" -gt 1 ] && [ "$new_main" != "$old_main" ] && [ "$state" = active ] && [ "$code" = 401 ]; then
  kill_recovered=yes; break
 fi
 attempt=$((attempt+1))
 sleep 1
done
test "$kill_recovered" = yes
echo S25_R2_REAL_SERVICE_SIGKILL_DISTINCT_PID_HTTP_AUTH_RECOVERY=PASS
"$work/staging/s25-lab-agent" post-restart
python3 scripts/s25_independent_verifier.py agents
echo S25_R2_SIGKILL_ORIGINAL_TWO_AGENT_SQLITE_AND_NATIVE_NO_REPLAY=PASS
# Token revocation is a direct dedicated-user local CLI action; the old
# HTTP bearer is tested by a separate process but never logged/exported.
peer_id="$(sed -n 's/^id=\([0-9a-f]*\) .*/\1/p' "$work/private/peer.secret")"
test "$peer_id" != ""
sudo runuser -u boxweave -- env -u INCUS_CONF -u INCUS_CONFIG_DIR -u XDG_CONFIG_HOME  HOME=/home/boxweave XDG_CONFIG_HOME=/home/boxweave/.config  /usr/local/bin/boxweave token-revoke -id "$peer_id" >/dev/null
python3 - "$work/private/peer.secret" <<'S25_REVOKED_HTTP'
import sys,urllib.request,urllib.error
token=next(x[6:].strip() for x in open(sys.argv[1]) if x.startswith("token="))
req=urllib.request.Request("http://127.0.0.1:18650/mcp",data=b"{}",
  headers={"Authorization":"Bearer "+token,"Content-Type":"application/json"},method="POST")
try:
  res=urllib.request.urlopen(req,timeout=8)
except urllib.error.HTTPError as exc:
  assert exc.code==401,exc.code
else:
  raise AssertionError("revoked bearer unexpectedly returned HTTP "+str(res.status))
print("S25_REVOKED_PEER_TOKEN_HTTP_DENIED=PASS")
S25_REVOKED_HTTP
sudo runuser -u boxweave -- env -u INCUS_CONF -u INCUS_CONFIG_DIR -u XDG_CONFIG_HOME  HOME=/home/boxweave XDG_CONFIG_HOME=/home/boxweave/.config  /usr/local/bin/boxweave token-create > "$work/private/peer.rotated"
chmod 0600 "$work/private/peer.rotated"
export BW_S25_PEER_FILE="$work/private/peer.rotated"
"$work/staging/s25-lab-agent" rotated-peer
"$work/staging/s25-lab-agent" bootstrap
python3 scripts/s25_independent_verifier.py wan6inner
# Evidence is captured *before* any original SDK cleanup. These outputs are
# public test CA/packet metadata, exact native NIC and original IDs only.
# No Token, private key, SQLite DB or unbounded Guest output is exported.
sudo cat "$work/private/simwan6/server.log" > "$proof/simwan6-l0-upstream.log"
sudo incus exec bw-s25-mcp-l1 --project "$project" -- cat /root/s25simwan6/server.log > "$proof/simwan6-l1-upstream.log"
sudo incus exec bw-s25-mcp-l1 --project "$project" -- cat /root/s25-inner-checkpoint.json > "$proof/simwan6-l1-originals.json"
sudo ip -s link show s25s6host > "$proof/simwan6-l0-veth-packets.txt"
sudo incus exec bw-s25-mcp-l1 --project "$project" -- ip -s link show s25s6host > "$proof/simwan6-l1-veth-packets.txt"
sudo incus config show bw-s25-mcp-l1 --project "$project" --expanded > "$proof/simwan6-l0-native-nic.yaml"
inner_uid="$(sudo incus exec bw-s25-mcp-l1 --project "$project" -- id -u boxweave)"
inner_name=bw-s25-mcp-l2
if [ "${P1B_S25_SCOPED:-0}" = 1 ]; then inner_name=bw-s25-mcp-l1; fi
sudo incus exec bw-s25-mcp-l1 --project "$project" -- incus config show "$inner_name" --project "user-$inner_uid" --expanded > "$proof/simwan6-l1-native-nic.yaml"
grep -q 'SIM_WAN6_REMOTE=' "$proof/simwan6-l0-upstream.log"
grep -q 'SIM_WAN6_REMOTE=' "$proof/simwan6-l1-upstream.log"
echo S25_SIMWAN6_BOTH_LAYER_RAW_PACKET_AND_NATIVE_EVIDENCE_ARCHIVED=PASS
python3 scripts/s25_independent_verifier.py l2
# This marker is emitted only once independent native L2 UUID, CPU and
# original Update/Stop/Start receipts were all verified, not from
# truncated Guest stdout inside the outer L1 bootstrap Exec.
echo S25_L1_MCP_L2_UPDATE_STOP_START_SAME_UUID=PASS
cp "$work/private/outer-checkpoint.json" "$proof/outer-checkpoint.json"
echo S25_TWO_LAYER_V1_MCP_CREATE_EXEC=PASS
if [ "${P1B_S25_SCOPED:-0}" = 1 ]; then
 # Nonsecret IDs are exchanged only between the two independent SDK clients.
 # The nested invocation uses its own local test credential and loopback MCP.
 inner_cp="$work/private/dual-inner-original.json"
 sudo incus exec bw-s25-mcp-l1 --project "$project" -- cat /root/s25-inner-checkpoint.json > "$inner_cp"
 outer_uuid="$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["uuid"])' "$BW_S25_CHECKPOINT")"
 outer_original="$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["create_op"])' "$BW_S25_CHECKPOINT")"
 inner_uuid="$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["uuid"])' "$inner_cp")"
 inner_original="$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["create_op"])' "$inner_cp")"
 test "$outer_uuid" != "$inner_uuid"
 test "$outer_original" != "$inner_original"
 BW_S25_FOREIGN_UUID="$inner_uuid" BW_S25_FOREIGN_ORIGINAL="$inner_original" "$work/staging/s25-lab-agent" dual-scope
 sudo incus exec bw-s25-mcp-l1 --project "$project" -- env P1B_S25_SCOPED=1 BW_S25_ALLOWED_NESTED=1 BW_S25_INNER=1 \
   BW_S25_URL=http://127.0.0.1:18650/mcp BW_S25_ADMIN2_FILE=/root/s25-admin2.secret BW_S25_CHECKPOINT=/root/s25-inner-checkpoint.json \
   BW_S25_FOREIGN_UUID="$outer_uuid" BW_S25_FOREIGN_ORIGINAL="$outer_original" /usr/local/bin/s25-lab-agent dual-scope
 python3 scripts/s25_dual_auth.py "$project"
 python3 scripts/s25_independent_verifier.py dual-scope
fi
# Additional P0 BH-009/BH-011/BH-015: cross-Project raw IPv4/IPv6
# packets and parent native Unix-socket isolation. This test uses ONLY
# a freshly created foreign guest in the disposable INNER L1 daemon.
# Native setup is a separate operator/Verifier actor, not v1 client.
foreign_probe_sha="$(sha256sum scripts/s25_r12_foreign_l1.py | awk '{print $1}')"
sudo incus file push scripts/s25_r12_foreign_l1.py bw-s25-mcp-l1/root/s25_r12_foreign_l1.py --project "$project"
test "$(sudo incus exec bw-s25-mcp-l1 --project "$project" -- sha256sum /root/s25_r12_foreign_l1.py | awk '{print $1}')" = "$foreign_probe_sha"
printf 'foreign_probe_sha256=%s\n' "$foreign_probe_sha" >> "$proof/identity.txt"
sudo incus exec bw-s25-mcp-l1 --project "$project" -- env BW_S25_R12_DISPOSABLE=1 P1B_S25_SCOPED="${P1B_S25_SCOPED:-0}" python3 /root/s25_r12_foreign_l1.py setup > "$proof/r12-foreign-inner-setup.txt"
grep -qx 'S25_R12_L1_FOREIGN_PROJECT_V4_V6_POSITIVE_AND_BIDIRECTIONAL_PACKET_DENIAL=PASS' "$proof/r12-foreign-inner-setup.txt"
grep -qx 'S25_R12_L1_PARENT_SOCKET_AND_FOREIGN_PROJECT_AUTH_ISOLATION=PASS' "$proof/r12-foreign-inner-setup.txt"
python3 scripts/s25_independent_verifier.py r12-foreign
sudo incus exec bw-s25-mcp-l1 --project "$project" -- cat /root/s25-r12-foreign-state.json > "$proof/r12-foreign-before-cleanup.json"
sudo incus exec bw-s25-mcp-l1 --project "$project" -- env BW_S25_R12_DISPOSABLE=1 P1B_S25_SCOPED="${P1B_S25_SCOPED:-0}" python3 /root/s25_r12_foreign_l1.py cleanup
python3 scripts/s25_independent_verifier.py r12-foreign-clean
sudo incus exec bw-s25-mcp-l1 --project "$project" -- cat /root/s25-r12-foreign-state.json > "$proof/r12-foreign-after-cleanup.json"
echo S25_R12_L1_FOREIGN_PACKET_AND_SOCKET_INDEPENDENT_V=PASS
"$work/staging/s25-lab-agent" guest-git
python3 scripts/s25_independent_verifier.py git
cp "$work/private/outer-checkpoint.json" "$proof/outer-checkpoint.json"
"$work/staging/s25-lab-agent" inner-clean
python3 scripts/s25_independent_verifier.py wan6innerclean
python3 scripts/s25_independent_verifier.py l2clean
"$work/staging/s25-lab-agent" cleanup
python3 scripts/s25_independent_verifier.py clean
cp "$work/private/outer-checkpoint.json" "$proof/outer-checkpoint.json"
test -z "$(sudo incus list --project "$project" --format csv -c n)"
if [ "${P1B_S25_SCOPED:-0}" = 1 ]; then
 test -z "$(sudo incus list --all-projects --format csv -c n)"
 echo P1B_S25_SCOPED_STARTUP_LIFECYCLE=PASS
 echo P1B_S25_SCOPED_FOREIGN_PACKET=PASS
fi
# Round 2 independent operator rollback: run only *after* original SDK
# cleanup. The upgrade canary uses identical source with only a distinct
# build label; always verify actual install digest and preserve original
# SQLite receipt history. No quota stress and no native guest writes.
mkdir -m 0755 "$work/canary-readable"
CGO_ENABLED=1 go build -trimpath \
  -ldflags "-X main.version=1.0.25-hostgo-r2-canary -X main.commit=$(git rev-parse HEAD)" \
  -o "$work/canary-readable/boxweave-r2-canary" ./cmd/boxweave
canary_sha="$(sha256sum "$work/canary-readable/boxweave-r2-canary" | awk '{print $1}')"
test "$canary_sha" != "$binary_sha"
if [ "${P1B_S25_SCOPED:-0}" = 1 ]; then
 # Scoped fresh-only has no successful Legacy upgrade/rollback. Keep
 # wrong/correct-digest and no-backup negatives, offline operator access,
 # and exact original receipt recovery; do not fake the old PASS markers.
 export BW_S25_CANARY_SHA="$canary_sha"
 ctl=boxweave.service
 before_pid="$(sudo systemctl show "$ctl" -p MainPID --value)"
 test "$before_pid" -gt 1
 python3 scripts/s25_independent_verifier.py scoped-maintenance-snapshot > "$work/private/scoped-maint-before.json"
 refused_scoped_maintenance() {
   label="$1"; shift
   if sudo /usr/local/bin/boxweave "$@" > "$work/private/scoped-$label.log" 2>&1; then
     echo "P1B_S25_SCOPED_MAINTENANCE_ACCEPTED=$label" >&2; exit 95
   fi
   if ! grep -Fxq 'boxweave: scoped fresh only: legacy installation mutation unsupported' "$work/private/scoped-$label.log"; then
     echo "P1B_S25_SCOPED_MAINTENANCE_UNEXPECTED_REFUSAL=$label" >&2
     return 96
   fi
 }
 refused_scoped_maintenance rollback rollback -project "$project" -approve-local
 refused_scoped_maintenance wrong-digest upgrade -project "$project" -source-binary "$work/canary-readable/boxweave-r2-canary" -expect-sha256 "$(printf '%064d' 0)" -approve-local
 refused_scoped_maintenance exact-digest upgrade -project "$project" -source-binary "$work/canary-readable/boxweave-r2-canary" -expect-sha256 "$canary_sha" -approve-local
 test "$(sudo systemctl show "$ctl" -p MainPID --value)" = "$before_pid"
 python3 scripts/s25_independent_verifier.py scoped-maintenance-snapshot > "$work/private/scoped-maint-active.json"
 cmp "$work/private/scoped-maint-before.json" "$work/private/scoped-maint-active.json"
 echo P1B_S25_SCOPED_ACTIVE_LEGACY_MAINTENANCE_REFUSED_NO_EFFECT=PASS
 sudo systemctl stop "$ctl"
 test "$(sudo systemctl is-active "$ctl")" = inactive
 code="$(curl -s -o /dev/null -w '%{http_code}' --max-time 2 -X POST http://127.0.0.1:18650/mcp || :)"
 test "$code" = 000
 refused_scoped_maintenance offline-rollback rollback -project "$project" -approve-local
 refused_scoped_maintenance offline-upgrade upgrade -project "$project" -source-binary "$work/canary-readable/boxweave-r2-canary" -expect-sha256 "$canary_sha" -approve-local
 test "$(sudo systemctl is-active "$ctl")" = inactive
 python3 scripts/s25_independent_verifier.py scoped-maintenance-snapshot > "$work/private/scoped-maint-offline.json"
 cmp "$work/private/scoped-maint-before.json" "$work/private/scoped-maint-offline.json"
 echo P1B_S25_SCOPED_OFFLINE_LEGACY_MAINTENANCE_REFUSED_NO_EFFECT=PASS
 sudo systemctl start "$ctl"
 ready=no
 for attempt in $(seq 1 40); do
   after_pid="$(sudo systemctl show "$ctl" -p MainPID --value)"
   code="$(curl -s -o /dev/null -w '%{http_code}' --max-time 2 -X POST http://127.0.0.1:18650/mcp -H 'Content-Type: application/json' -d '{}' || :)"
   if [ "$after_pid" -gt 1 ] && [ "$after_pid" != "$before_pid" ] && [ "$code" = 401 ]; then ready=yes; break; fi
   sleep 1
 done
 test "$ready" = yes
 "$work/staging/s25-lab-agent" scoped-maintenance-readback
 python3 scripts/s25_independent_verifier.py scoped-maintenance-readback
 python3 scripts/s25_independent_verifier.py scoped-maintenance-snapshot > "$proof/scoped-maintenance-final.json"
 cmp "$work/private/scoped-maint-before.json" "$proof/scoped-maintenance-final.json"
 sudo /usr/local/bin/boxweave install-status -project "$project" > "$proof/scoped-final-status.json"
 cp "$work/private/outer-checkpoint.json" "$proof/outer-checkpoint.json"
 printf 'maintenance_original_binary_sha256=%s\nmaintenance_canary_binary_sha256=%s\n' "$binary_sha" "$canary_sha" > "$proof/scoped-maintenance-binaries.txt"
 echo P1B_S25_SCOPED_MAINTENANCE_NO_EFFECT_ORIGINALS=PASS
 echo P1B_S25_LEGACY_SUCCESSFUL_UPGRADE_ROLLBACK=OUT_OF_SCOPE_170
 # Retained dual-Agent verifier completed, including scoped refusal,
 # offline/readback and all preceding original lifecycle assertions.
 echo S25_TARGETED_MCP_DUAL_AGENT_RECOVERY_VERIFIER=PASS
 echo S25_ALL_BH_AND_TC_FORMAL_SIGNOFF=NOT_YET
 exit 0
fi
test ! -e /var/lib/boxweave-installer-backup
if sudo /usr/local/bin/boxweave rollback -project "$project" -approve-local > "$work/private/unbacked-rollback.json" 2>&1; then
 echo S25_R2_FAIL_UNBACKED_ROLLBACK_ACCEPTED >&2; exit 95
fi
if sudo /usr/local/bin/boxweave upgrade -project "$project" \
  -source-binary "$work/canary-readable/boxweave-r2-canary" \
  -expect-sha256 0000000000000000000000000000000000000000000000000000000000000000 \
  -approve-local > "$work/private/wrong-upgrade.json" 2>&1; then
 echo S25_R2_FAIL_WRONG_UPGRADE_SHA_ACCEPTED >&2; exit 95
fi
test "$(sha256sum /usr/local/bin/boxweave | awk '{print $1}')" = "$binary_sha"
sudo systemctl is-active --quiet boxweave.service
echo S25_R2_UNBACKED_ROLLBACK_AND_BAD_UPGRADE_DIGEST_REFUSED=PASS
sudo /usr/local/bin/boxweave upgrade -project "$project" \
  -source-binary "$work/canary-readable/boxweave-r2-canary" \
  -expect-sha256 "$canary_sha" -approve-local > "$work/private/upgraded.json"
test "$(sha256sum /usr/local/bin/boxweave | awk '{print $1}')" = "$canary_sha"
sudo systemctl is-active --quiet boxweave.service
sudo test -d /var/lib/boxweave-installer-backup
echo S25_R2_EXACT_SOURCE_CANARY_UPGRADE_WITH_LEDGER_BACKUP=PASS
sudo systemctl stop boxweave.service
test "$(sudo systemctl is-active boxweave.service)" = inactive
code="$(curl -s -o /dev/null -w '%{http_code}' --max-time 2 -X POST http://127.0.0.1:18650/mcp || :)"
test "$code" != 401
echo S25_R2_OPERATOR_FALLBACK_WHILE_MCP_UNAVAILABLE=PASS
sudo /usr/local/bin/boxweave rollback -project "$project" -approve-local > "$work/private/rollback.json"
test "$(sha256sum /usr/local/bin/boxweave | awk '{print $1}')" = "$binary_sha"
rollback_ready=no
attempt=0
while [ "$attempt" -lt 40 ]; do
 sudo systemctl is-active --quiet boxweave.service
 code="$(curl -s -o /dev/null -w '%{http_code}' --max-time 2 -X POST http://127.0.0.1:18650/mcp || :)"
 if [ "$code" = 401 ]; then rollback_ready=yes; break; fi
 attempt=$((attempt+1));sleep 1
done
test "$rollback_ready" = yes
echo S25_R2_OPERATOR_ROOT_LOCAL_EXACT_BINARY_ROLLBACK=PASS
"$work/staging/s25-lab-agent" round2-postrollback
python3 scripts/s25_independent_verifier.py round2-postrollback
cp "$work/private/outer-checkpoint.json" "$proof/outer-checkpoint.json"
printf 'round2_original_binary_sha256=%s\nround2_canary_binary_sha256=%s\n' "$binary_sha" "$canary_sha" > "$proof/round2-binary-identities.txt"
echo S25_R2_ORIGINAL_SQLITE_NATIVE_CLEAN_AFTER_ROLLBACK=PASS
echo S25_TARGETED_MCP_DUAL_AGENT_RECOVERY_VERIFIER=PASS
echo S25_ALL_BH_AND_TC_FORMAL_SIGNOFF=NOT_YET
