# S4-B2 Stage 3 — root helper restart hardening (partial)

2026-11-17 PT. Sole Draft PR #153. Baseline `a8479796c0008f4cf2326b6bde5a04ef80975196`. **S4_B2_PARTIAL**, **B3_NOT_STARTED**.

## Source-exact evidence

- GitHub Actions at baseline: [quality #44220527981](https://github.com/vanishs/boxweave/actions/runs/44220527981), [dual architecture #44220527987](https://github.com/vanishs/boxweave/actions/runs/44220527987), [standalone native B2 #44220527993](https://github.com/vanishs/boxweave/actions/runs/44220527993), [installed S4-A #44220528033](https://github.com/vanishs/boxweave/actions/runs/44220528033) SUCCESS. Native B2 runner explicitly reports `DESIGN_CANDIDATE_ONLY_NOT_INSTALLED_PRODUCT`.
- Fresh disposable L0 RC7 Box `072909ed-38c4-4e4e-8c69-53abab056ed4`, original Create `09e0564a765f58dbb13f76cbac401983`, native `baec1811-e262-4234-bd9e-ad4f177d0572`. Exact source cloned; baseline `make verify` passed.
- New fix: after a root helper dies, an approved restart may remove only the dead, unchanged `/run/boxweave-native-peer-root.sock` inode with exact root UID, approved service GID and mode 0660, **only** when kernel connect returns ECONNREFUSED. Active listener, wrong owner/group/mode, symlink/foreign file, changed inode and uncertain kernel responses still refuse. Neither the per-original UNKNOWN journal nor the exact operation/SHA approval is touched. This repair **does not recover or rearm an UNKNOWN operation**.
- Added `TestS4B2RootSocketCrashRestartOnlyExactStaleInode` plus existing root peer credential/parameter tests PASS. After gofmt and git diff check, L0 full `make verify` PASS: BW original Exec `0bd01679ab3912290f97e6dff3c11f38`, native `bb1bb743-48b5-44e2-a013-5caae3f6e5f4`; includes race, vet, tidy diff and formatting. This proves source/socket behavior in a disposable guest, **not** an installed systemd lifecycle.

## Mandatory blockers before B2 can pass

1. Installed nonroot controller → approved root service actual activation/restart and durable original; hook real stop/quiesce, dual-stack ACL ETag DROP and independent per-NIC nft kernel and foreign-packet verifier. Current worker still accepts private isolate/verify callbacks only; the installed product does not wire them.
2. One complete installed live IPv4/IPv6 TCP/UDP grant/revoke with exact conntrack and independently proven foreign-Project established traffic survival, plus overlapping address/unmanaged route refusal.
3. Crash, power-loss surrogate, lost ACK, partial ACL/nft faults: verified no-replay, independently reconciled recovery and fresh authorized original. Stale socket removal is not that recovery.
4. Installed Scriptlet restricted TLS Project/Profiles/Files/Exec/ACL parity through restart, failed activation and exact rollback.
5. Same **final** SHA with installed B2, S4-A, dual-stack native, complete race/build, independent raw verifier and exact cleanup.

No P0 was declared PASS merely from standalone native CI or unit tests. No release, merge, B3 or shared L0 host mutation. The one-use L0 Box must be stopped and deleted after completing this work.
