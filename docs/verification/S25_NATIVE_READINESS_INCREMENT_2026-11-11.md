# S2.5 — native prerequisite readiness implementation and bounded negative evidence

**Status: SOURCE_AND_LOCAL_NEGATIVE_CHECKS_ONLY / BRIDGEHEAD NOT PASSED (2026-11-11 PT).** The source increment is on the topic branch `feat/v1-s25-native-readiness-20261111` from protected `v1` baseline `379b50b0a209bd598d83edc9f9ca7d8bbb13d7d8`. It does **not** complete BH-001–BH-017, establish `LAB_PASS`, authorize a dedicated physical host, or prove a v1 controller was installed on a nested host.

## Implemented, deliberately narrow

- `host_status`/`doctor` native readiness now requires the same locally approved Ubuntu 24.04 image alias and image properties that native `Create` already required. Missing/wrong images refuse the **first** Box at admission rather than presenting misleading readiness.
- Added root-local, read-only `native-prereq-plan -project user-UID` and `native-prereq-verify -project user-UID -native-prereq-sha256 REVIEW -approve-local`. The review uses the **official Incus Go SDK**, does not prepare/modify native resources and binds the approval to a host hash, relevant native object readbacks/ETags, approved image fingerprint and exact checked identities. No host name, pool, ACL, image URL or Project is accepted from public MCP Tools.
- Review requires an exclusively scoped non-root `boxweave` actor, a restricted/nesting-enabled Project, one root disk without host mounts, one private default-deny IPv4/IPv6 anti-spoof NIC, a real separately mounted ext4/XFS `dir` pool with `prjquota`, a managed bridge, an ACL `used_by` limited to the owned Project, and a preexisting Ubuntu alias. An unknown ACL usage URL or absent mount evidence is **BLOCKED**, not guessed safe.
- Added the read-only L1 filesystem precheck `sh scripts/s25_l1_storage_precheck.sh <pool>`. No alternate storage driver is enabled and no quota check is disabled.
- Prepared a one-use Ubuntu 24.04 Actions rehearsal that invokes the new operator-readback gate on the existing **real S2 native fixture** and reruns the S2 installed MCP/native tests. It is explicitly a *final validation job*; a green job would still not be BH network packet, dedicated-host, or nested L2 acceptance.

**Approval scope:** The preflight checks **operator-prepared configuration only**, not automatic provisioning. `config_ready=true` means only that the source-visible configuration prerequisites pass. `packet_witness_required=true` is intentionally always emitted; real IPv4/IPv6 ACL packet and anti-spoof, positive egress, independent foreign Project denial, original native UUID/operation and EDQUOT evidence still require separate installed acceptance. It is invalid to promote a plan digest or a configuration-only review to `LAB_PASS` or `HOST_GO`.

## Local reproducible observations (not phase acceptance)

| Scope | Exact observation | Tier / result |
|---|---|---|
| Source build | Ubuntu 24.04 managed test Box `boxweave-test-ser2`, Go 1.26.9, Incus 7.6 client/daemon, CGO enabled. `go test ./internal/install ./internal/native ./cmd/boxweave` passed during implementation. | T PASS |
| Developer quality | `make verify` on the working source before its final documentation update completed with exit 0, including full Go race packages, vet, format, module checks and build. Re-run on final staged snapshot. | T PASS (prior snapshot) |
| Actual unprepared inner native | Root-local candidate `native-prereq-plan -project user-1001` observed native extensions but **seven not-ready prerequisites**, reported `config_ready=false`; incorrect 64-byte digest refused; even the exact reviewed digest refused verification because prerequisites were not ready. Non-root plan invocation refused. No native write was issued. | N read-only negative PASS; I/V not attempted |
| Unprivileged L1 storage | `sh scripts/s25_l1_storage_precheck.sh default` returned exit **83**, `S25_L1_STORAGE_FILESYSTEM=bcachefs`, `S25_L1_NATIVE_DIR_QUOTA=NESTED_STORAGE_BLOCKED`. Root namespace lacks a `/dev/loop-control` device. A nested Incus daemon running is not proof of a quota-backed L2. | N precondition BLOCKED **on this Box only** |
| ACL packets, real L2, exact candidate service, crash/UNKNOWN, verifier, dedicated host | Not performed in this increment; no supported positive native v1 installation or Agent path claim. | BH I/N/V UNVERIFIED |

This developer Box already contains unrelated native state; it was used for **read-only discovery and Go development**, not as a disposable installation. The existing Box, Incus guests, ACLs, bridges and the three unrelated original UNKNOWN operations were not modified or deleted.

## First final CI feedback and source correction

The first exact-head disposable [native S2.5 job](https://github.com/vanishs/boxweave/actions/runs/43147286231) on source `6a640ae` reached the one-use installed Incus 7.6 fixture. **Seven of eight** native prerequisite checks were positive (actual extensions, restricted Project/nesting, root/NIC, quota-backed pool, managed bridge, exclusive ACL used_by, approved image), while the service identity check returned BLOCKED, so the native job correctly **failed** rather than claiming acceptance. The S2 acceptance fixture explicitly cleared inherited `INCUS_CONF`/`INCUS_CONFIG_DIR`/`XDG_CONFIG_HOME` and supplied `HOME=/home/boxweave`, but the reused host-local root operator helper ran `runuser` without reproducing this bounded client environment. This candidate corrects the invocation and adds an argument-level regression test. The correction is **pending fresh exact-head native verification** at this documentary checkpoint; do not relabel the first red run as a PASS.

## Remaining mandatory S2.5 work

1. On a **separate approved disposable native LAB**, establish actual storage backing and native network before any create. Obtain an explicitly reviewed operator-owned plan for Project/Profile/ACL/image/pool; either implement guarded local apply+rollback or retain independently audited operator provisioning with exact native ownership evidence. A simple S2 CI fixture is not itself an installer.
2. Make BH-003's source preflight actually pass on real approved pristine native config. Validate ACL project/usage URLs against real Incus 7.6 API, native source mount, required image and independent denied foreign native effects. On any refusal, preserve original resources and report the exact missing prerequisite.
3. Prove EDQUOT for root and Exec spool, and actual private IPv4/IPv6/anti-spoof packet paths before calling `LAB_PASS`; find a **genuinely quota-capable** unprivileged L1 for BH-010 or report `NESTED_STORAGE_BLOCKED` without relaxing the native contract.
4. Complete the BH-001–BH-016 and TC-011–TC-014 original-operation continuity and independent L0→L1→L2 Verifier matrix on one exact installed candidate SHA/digest. Only then consider an **explicitly approved exact dedicated physical host** and BH-017. Old `main` is never coinstalled with v1.

**No new public MCP tools.** This increment only adds host-local CLI inspection and safety admission. S3/S4/S5 and S4 deferred dependent Delete remain unchanged.
