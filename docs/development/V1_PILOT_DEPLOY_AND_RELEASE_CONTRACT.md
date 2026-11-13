# v1 — nested pilot, dedicated-host pilot, installer and Release contract (2026-11-13)

**Owner decision / scope:** resolve the chicken-and-egg gate. Do not require evidence obtainable only **after** a dedicated physical-host installation as a precondition for attempting a **reversible, owner-authorized development-only host pilot**. This corrects the prior "HOST_GO PASS before any physical install" circular gate; it does **not** waive P0 safety prerequisites, install on an unnamed host, merge PR #133, publish a supported v1 release, or allow coexistence with legacy main on that host.

## 1. Gates (distinct, no retroactive PASS)

| Gate | Meaning | May do next |
| --- | --- | --- |
| `ROUND1_PASS`, `ROUND2_PASS`, `LAB_PASS` | Already signed bounded isolated native/network/fault work; retain accepted exceptions and original receipts | Isolated live-ChatGPT rehearsal |
| `NESTED_PILOT_CORE_PASS` | Real owner-connected ChatGPT OAuth → new v1 HTTP MCP → original UUID-bound Create/Update/Exec/output/Stop/Delete, real isolated Incus readback, Git test-fail/fix-pass, IPv4 egress off/on/off, credential/Host negatives, exact cleanup | Assess whether bounded physical pilot is admissible |
| `PILOT_DEPLOY_GO=CONDITIONAL` | Prior two gates + approved known-risk register + **separately identified dedicated host**, non-mutating preflight, explicit owner approval for host-specific effects, safe rollback/stop and data preservation | Only that host's initial reversible dev-only installation |
| `HOST_POSTFLIGHT_PASS` | After authorized install: exact same source/binary on that physical host, Incus Project/ACL/pool/runtime, Agent original receipts + native readback, fault/recovery/rollback, host network packets, independent Verifier | Assess final HOST_GO |
| `HOST_GO=PASS` | HOST_POSTFLIGHT_PASS **and** remaining mandatory BH/TC P0 negative/network/installed/independent evidence + same-candidate required Actions reconciled | Dedicated dev-host usage; **not** S5 production/release readiness |
| `HOST_NO_GO` / `HOST_POSTFLIGHT_NO_GO` | Missing/failed mandatory P0, unsafe host, unconfirmed original native UNKNOWN, mismatched binary, unrecoverable rollback or privilege isolation | Stop before further writes; preserve receipts; independent fallback |

**Current facts vs verdict:** PR #133 `b8f0a8d` real ChatGPT v1 OAuth read/write and Git/IPv4 subtests PASS; [PR evidence](https://github.com/vanishs/boxweave/pull/133#issuecomment-6851880060) shows original 15/15 operation receipts, 0 new UNKNOWN, exact guest cleanup. Independent main-connected native/SQLite readback exists, **but not a separately signed independent Verifier**; full simulated routed IPv6 anti-spoof/foreign suite and exact-head native/nested Actions remain to be reconciled. Do not relabel `NESTED_PILOT_CORE_PASS` as all BH/TC PASS. Formal `HOST_GO=PASS` remains pending.

## 2. Pilot-stage authority and rollback

1. Owner names the exact dedicated development host and explicitly permits the **particular** OS/service-user, signed Incus version/Project/pool/ACL/loopback listener and any installer/native-prepare changes. First run read-only host fingerprint, available disk/inodes, native daemon/version/socket, unexpected consumer/foreign instances and previous `boxweave` binary/service/state inventory. **No legacy main + v1 co-install.**
2. Prepare/review native plan and precise digest under existing v1 root-local `native-prereq-plan` / `native-prepare-plan` interfaces. Never relax ACL, expose Incus admin socket, silently upgrade Incus or touch foreign resources. Preserve original gateway/Incus `UNKNOWN`; do not retry them.
3. Deploy a pinned, SHA256-verified **exact binary** with the external installer **only after** explicit approval. Installation uses the built-in `install-stage`, `install-activate` or `upgrade` and its own manifest/backup transaction; an external helper must not rewrite SQLite/units or replace the native rollback state machine.
4. Verify locally first (non-root restricted identity, Incus 7.6 extension/Project/ACL readback, loopback `/mcp` unauthorized 401), then owner-configured temporary HTTPS/Tunnel and approved Agent. Reject/stop on unexpected public/foreign access. Independent verifier witnesses original UUID/native operations, negative packets, recovery, backup/rollback and exact cleanup. Unverified `dir` quotas remain a warning, **never** a claimed hard disk limit.
5. After host postflight and remaining required native/security/Actions crosswalk, explicitly sign `HOST_POSTFLIGHT_PASS` and `HOST_GO`. Pilot failure does not justify silently erasing or replacing state. Physical installation is **not** retroactively proof of any skipped LAB requirement.

## 3. One Go application binary, but not a complete installable deployment

The v1 **runtime application artifact** is one CGO-enabled Linux `boxweave` executable (Go MCP server + local CLI; bundled SQLite C dependency). It is **not** a self-contained replacement for OS, signed Incus, Project/Profile/ACL, service account, systemd or approved credentials. The installer helper, checksums, release manifest, CI smoke scripts and GitHub Actions are **distribution tooling outside the running controller**, not more MCP tools, services, or a second privileged application.

External `install.sh` provides:
- `--plan` (default, no writes), `--apply` (explicit exact-host authorization), and `--version v1.X.Y` / `--version v1.X.Y-rc.N` for an immutable Release. Without `--version` it reads the **v1-only** `V1_PUBLISHED_RELEASE` pointer from the protected `v1` branch; `UNPUBLISHED` fails closed. **Never use GitHub's repository-wide latest Release endpoint:** main v0.x coexists here.
- Local pre-release pilot: `--local-binary ABSOLUTE_PATH --sha256 EXACT_DIGEST`. Require candidate `build-info` and host-local review. This is not a formal tagged Release and cannot auto-advance the stable pointer.
- Linux `amd64` and `arm64` assets selected only after machine-architecture check; fetch bounded HTTPS bytes; match immutable Release tag, manifest/source commit, GitHub-reported artifact digest and SHA256SUMS. Refuse mismatched/draft/unknown assets, unexpected existing state and downgrade/upgrade without review. Native prerequisite setup is **not silently performed**.
- Script coordinates (does not implement itself) Go `install-stage/activate/upgrade/status/rollback/install-commit`. Explicit `--apply` must not auto-finalize the previous rollback backup, modify existing Incus, select a public hostname or enable a tunnel.

## 4. Release pipeline: candidate artifacts vs supported publication

- Always compile Linux **amd64 + arm64** with **CGO_ENABLED=1** and the correct native target compiler/runner. Stamp **actual** source SHA and version in `build-info`; build each platform natively to avoid a broken ARM64 SQLite cross-build.
- Candidate CI builds are **not** automatically stable Releases. Produce named binaries, digests and source-bound machine-readable manifest; at least execute on-platform binary `build-info` and SQLite CLI smoke, with full Go race on supported runner. Source `VERSION=1.0.0` today is **not** a published tag.
- An explicitly triggered **one-shot RC release branch** publication may create immutable, checksum-verified `v1.X.Y-rc.N` Release assets only after green same-source jobs. No auto-upgrade of existing hosts, no silent overwrite/re-tag, no stable-pointer movement.
- **Stable** `v1.X.Y` release still requires S5 formal installed native/nested acceptance, independent verifier, source/artifact identity and all release criteria. Only a separately reviewed owner-approved PR may promote `V1_PUBLISHED_RELEASE` to that existing published stable tag; publishing does not change it.
- Every release has one pair of ELF assets `boxweave-v1.X.Y-linux-amd64` and `boxweave-v1.X.Y-linux-arm64`, `release.json`, `SHA256SUMS` and an installer helper; the provenance/limitations are stated clearly. GitHub Release asset hashes are not an independent trust anchor by themselves: always bind to the official repository/tag/source and review the operator's SHA when needed.

## 5. Operator commands — explicit local pilot or immutable Release

The distribution files are now implemented on Draft PR #133: [external `install.sh`](../../install.sh), [v1-only pointer](../../V1_PUBLISHED_RELEASE), [GitHub Actions amd64/arm64 + source-bound RC workflow](../../.github/workflows/v1-release.yml), [manifest assembler](../../scripts/v1_release_manifest.py) and [negative tests](../../scripts/test_v1_release_manifest.py). The current source `VERSION=1.0.0` is **not** a published v1 Release; `V1_PUBLISHED_RELEASE=UNPUBLISHED` makes default selection fail closed.

Before first install, provision and review signed Incus, `boxweave` service user restricted to the `incus` group, exact `user-1001`-like Project/default Profile/ACL/image/pool with the existing v1 `native-prepare-plan/apply`. Confirm that the target host is dedicated and has no old main service. Do not treat this helper as an Incus/bootstrap installer.

Read-only exact local candidate plan on the approved host:

```sh
sh install.sh --plan --project user-1001 \
  --local-binary /absolute/path/to/reviewed/boxweave \
  --sha256 <reviewed-64-hex-binary-digest>
```

After separately identifying and approving that machine's printed `host_machine_id_sha256`, run as root **only on that host** (not in this documentation task):

```sh
sudo sh install.sh --apply --project user-1001 \
  --local-binary /absolute/path/to/reviewed/boxweave \
  --sha256 <reviewed-64-hex-binary-digest> \
  --confirm-host-sha256 <exact-sha256-from-plan>
```

The verified helper calls Go `install-stage` then Go `install-activate` for a fresh, previously reviewed host. To update an **existing verified** installation, add `--upgrade`; the Go backup must have been independently reviewed/committed and no incompatible SQLite schema is silently adopted. An interrupted stage may require the original Go `install-status` and explicit operator recovery; the helper does **not** delete state or auto-retry an UNKNOWN.

For a **published development prerelease only** (after one is actually published), choose exact tag:

```sh
sh install.sh --plan --project user-1001 \
  --version v1.0.0-rc.1
```

Explicit RC naming selects the prerelease directly; no additional `--allow-prerelease` switch is required. The v1-only **default** pointer remains stable-only and will never select an RC. A future signed-off stable `v1.X.Y` can be selected using `--version v1.X.Y`. Once the reviewed v1-only pointer is updated to that published stable tag, omitting `--version` selects it; **not** the repository-wide main/v1 latest GitHub Release.

Release workflow intentionally separates three actions: ordinary PR/push runs build+manifest artifacts only; **after accepted PR merge into protected `v1`**, the operator creates exactly one branch `release/v1.X.Y-rc.N` from the latest reviewed `v1` HEAD. That push triggers **two separate clean GitHub-hosted Ubuntu jobs** (S2 native Incus regression and S2.5 nested verifier). Never run S2 then S2.5 sequentially on one runner: S2 installs native Incus and S2.5 correctly refuses existing native state (`S25_LAB_BLOCKED_FOREIGN_NATIVE_STATE`). Both job receipts, both native-architecture builds, source checks and manifest must PASS before creating an immutable **development prerelease**; GitHub's default branch is legacy `main`, so `workflow_dispatch` scoped only to `v1` would not reliably work (do not copy the v1 product to `main`); a **stable S5-qualified Release publication job is not yet implemented** and requires S5 acceptance/independent V. The current Script/Go installer owns staged artifacts and rollback, never automatically enables Cloudflare or starts a real physical pilot.

**2026-11-13 proof:** On isolated `boxweave-test-ser2`, the external script's `sh -n`, embedded Python compile and 4 manifest/negative tests passed; read-only `--plan --local-binary` accepted the exact previously installed `b8f0a8d` binary SHA256 `4b83e06ae5d58f221ba4f63bf17413cfd2e5f23ab44b182aeb447365305935a4` and rejected a wrong digest. No `--apply` was invoked. [Same-source Actions run 43423024906](https://github.com/vanishs/boxweave/actions/runs/43423024906): Go native CGO `amd64` / `arm64`, source tests and candidate manifest all SUCCESS; publishing and real native prerelease tests correctly SKIPPED on a normal PR push. Required quality/S1 source at that commit also SUCCESS. This is distribution-build QA, **not** real host installation, native S5 release or final HOST_GO.

## 6. Entry points, ownership and exclusions

`docs/development/HOST_GO_THREE_STAGE_RUNBOOK.md` Round 3 is now **the bounded exact-host pilot + formal postflight**, *not* an additional prerequisite before any host can be installed. A request to fix docs/build distribution tooling is not an authorization to run Round 3 or install on a specific machine. The `PILOT_DEPLOY_GO` policy is a permitted **conditional path**, not an assertion that the unnamed target is already safe. Historical evidence remains immutable and failures do not disappear.

Do not merge main into v1, import main's Python/Containarium/REST/bootstrap architecture, ship a generic host shell MCP Tool, make installer a server endpoint, configure Cloudflare Portal/Tunnel automatically, or run release publication from PR #133 merely because a source job is green.
