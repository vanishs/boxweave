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
- An explicitly triggered **pre-release** publication may create immutable, checksum-verified `v1.X.Y-rc.N` Release assets only after green same-source jobs. No auto-upgrade of existing hosts, no silent overwrite/re-tag, no stable-pointer movement.
- **Stable** `v1.X.Y` release still requires S5 formal installed native/nested acceptance, independent verifier, source/artifact identity and all release criteria. Only a separately reviewed owner-approved PR may promote `V1_PUBLISHED_RELEASE` to that existing published stable tag; publishing does not change it.
- Every release has one pair of ELF assets `boxweave-v1.X.Y-linux-amd64` and `boxweave-v1.X.Y-linux-arm64`, `release.json`, `SHA256SUMS` and an installer helper; the provenance/limitations are stated clearly. GitHub Release asset hashes are not an independent trust anchor by themselves: always bind to the official repository/tag/source and review the operator's SHA when needed.

## 5. Entry points, ownership and exclusions

`docs/development/HOST_GO_THREE_STAGE_RUNBOOK.md` Round 3 is now **the bounded exact-host pilot + formal postflight**, *not* an additional prerequisite before any host can be installed. A request to fix docs/build distribution tooling is not an authorization to run Round 3 or install on a specific machine. The `PILOT_DEPLOY_GO` policy is a permitted **conditional path**, not an assertion that the unnamed target is already safe. Historical evidence remains immutable and failures do not disappear.

Do not merge main into v1, import main's Python/Containarium/REST/bootstrap architecture, ship a generic host shell MCP Tool, make installer a server endpoint, configure Cloudflare Portal/Tunnel automatically, or run release publication from PR #133 merely because a source job is green.
