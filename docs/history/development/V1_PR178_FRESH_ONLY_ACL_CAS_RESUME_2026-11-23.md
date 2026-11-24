# PR #178 — fresh-only scoped ACL CAS and guarded effect planning / reconnect checkpoint

2026-11-23 (America/Los_Angeles). **Draft only. No installed new Root dispatcher or final native acceptance.** The [superseding Owner decision](V1_PEER_MCP_AUTOSTART_NONDISRUPTIVE_OWNER_DECISION_2026-11-23.md) limits formal acceptance to fresh installation. Shared ACL online migration is explicitly out of scope.

## Recovery and provenance

Remote starting head `6a4fb80cc23866ee81a93af6f08ffa851113b7aa` (26 commits, 4/4 workflows successful: 45290441818, 45290441780, 45290441772, 45290441906). Existing L0 RC13 remained intact. Isolated Box `pr178-lab-1123` UUID `ac585d2c-37ae-4879-a2e6-3cecc338d4d7`.

- `/root/pr178/repo` is a symlink to `boxweave-fix-rc13-peer-blast-radius-lifecycle-20261123`. That older source has no valid Git HEAD and 832 staged files; no meaningful uncommitted source delta versus the remote except files absent from that older tree.
- Actual recovery source `/root/pr178/new-4cbdb5f` also has an unborn Git HEAD. Four new Go files and the changed transition type were recovered. Its progress doc was older than remote and it lacked the remote preflight report: do not overwrite either remote document with old copies. Python caches are excluded.
- Full non-destructive archive `/root/pr178/recovery-20261123/original-worktrees.tar.gz`, SHA256 `077ca1f4cce19abae2856ec5220deec5b9505859bbf5bfb653d6cedcc9b6e9b7`, operation `c14709d185dd8924ccdd53c435176b0f`, Guest exit 0.
- Reconstructed independent candidate `/root/pr178/recovery-20261123/candidate` from exact remote tarball plus five reviewed source changes. Local Git initialized only for make verify's tracked YAML requirement; its index is not a valid upstream commit ancestry. No reset, clean, original directory replacement, L0 network change or #177 edit.

## Delivered source slice

- `network_peer_scoped_acl_cas.go`: package-private Incus SDK ACL adapter uses default-project ACL Get + UpdateNetworkACL ETag, exact UUID-derived ACL name, singleton UsedBy, complete expected ACL content, boot preference, three guarded readbacks and a one-attempt receipt. Unknown PUT or post-readback returns unverified; never retries. Its caller must still fsync the original-specific intent and continuously hold claims/locks.
- `network_peer_scoped_effect_sequence.go`: private plan only. Grant/regrant: prepare source ACL and target ACL while guard denies the selected new initiation; exact scoped conntrack step under protection; **guard allow last**. Revoke: guard deny first; exact scoped conntrack step; source and target ACL cleanup under deny. All stages carry required guard preimage digest. The final worker must prove guard postimage, and all other native state, before success.
- Hardened against an allow/revoke polarity mismatch (guard JSON rule delta must add/remove exactly one four-rule group in intended direction); complete exact per-UUID ACL and boot-key membership, UUID-derived ACL names, valid exact boot values; ACL CAS also calls sequence validation. C/D maps must deep-equal. These checks are defense in depth, not actor authorization or a replacement for root-collected proof.
- Tests: fake SDK ETag competition, foreign ACL/UsedBy/UUID/project, altered C/D or boot map, guard drift before/at/after PUT, applied-but-ACK-lost and bad post-readback; no speculative replay and no unrelated ACL writes. No actual native ACL mutation by this candidate slice was executed.

## Evidence

| Evidence | Operation | Result |
| --- | --- | --- |
| Prior agent targeted race (original sequence, before hardening) | `3e8c6abbdc8b0b3c118bc5b4ff6701a7` | PASS, exit 0 |
| Prior agent full make verify (older tree) | `f13c43531e29937cd43b9e8ad6257b4e` | PASS, exit 0, original output recovered |
| Restored candidate targeted race | `8fb87ddb87f0b6e143b8f290f94772c7` | PASS, exit 0 |
| Hardened candidate targeted race, six top-level tests plus negative subcases | `3584ef5d4d04e0bffaa72495abe7bf0b` | PASS, exit 0 |
| Hardened candidate full make verify | `07ef01deb64deee8c4a4f27a0913c47a` | Check terminal Guest exit and full output, not Operation state alone; latest result recorded in progress/PR checkpoint |

Go 1.26.9 linux/amd64. Candidate source file SHA256 (exact transfer from Box readback):
- ACL CAS: a8dd16366ae4bda59a262df7b1acfde62d0ebdc5e30da9361b3937552bed9e2a
- ACL CAS test: b2858462e05d83bb27f80fde021ec738ecd67cbf483321a9df56c1c349841127
- effect sequence: 9525204742ad28b7aa4210120d9a6f70ef7dea61b02e8497510e1d1644706803
- sequence test: dc32e10869bece7c9befac544d8df638707fa89ebe91c58806b6cf5f97837de5
- root transition: 0fd16661fcaddf71088667c15eb48bc16425fbd08692bbaddf79c927ae55a258

## BLOCKED — do not activate

1. **Fresh installation admission and lifecycle:** not yet implemented end-to-end. Add an explicit versioned scoped mode and independently root-proven clean install admission BEFORE irreversible installer, schema, network, service stop or overwrite effects. Refuse old shared ACL and unsupported upgrade/reuse; no automatic conversion. Prove fresh owned bridge/default profile, protected mode metadata, initially fail-closed guard before Box NICs or boot; design per-Box ACL creation/deletion with durable claims, exact native readback, no name takeover and conservative failure recovery. Reuse the fresh IPv4 inventory checks where applicable (not as a universal proof). Review `internal/install/native_prereq_prepare.go`, `native_fresh_ipv4.go`, `lifecycle.go`, `install.sh`, native Create/Delete and SQLite bindings. Test old installation unchanged on rejection (binary/unit/DB/WAL/network, running workloads). Do not globally modify released RC13 behavior or existing L0 during development.
2. **Full-operation Root journal:** compose protected original/actor/claims, trusted root-collected SQLite+SDK+sysfs+nft preflight, locked stage intents before every ACL/conntrack/guard effect, independent postimage, terminal DB commit and UNKNOWN recovery. Existing guard-only journal is insufficient for a partial A/B ACL transition. Recovery must read actual state and exact provenance, not blindly repeat CAS or nft effects.
3. **Conntrack and reverse-direction safety:** current `NativePeerConntrackPurge` uses a shared ACL/all-stopped fence; private purge helper probes both A→B and B→A. Do not wire either as a live new-path substitute. Root must prove exact project/zone/tuple ownership and selected direction, avoid changing unrelated C/D or independently authorized reverse sessions. Current guard replies rely on ct established/related, not explicit original/reply direction. Specifically test an already established A→B flow when a separate B→A grant remains, revoke/regrant under traffic, TCP/UDP IPv4/IPv6, invalid/related/foreign/reincarnated tuples; prove stale flows cannot remain or re-form under the reverse grant. The planning word DROP is shorthand for blocking unauthorized selected-direction initiation, not evidence that legitimate reverse replies are blocked. If the real kernel cannot prove this invariant, refuse rather than loosen the guard.
4. **Trusted install/MCP→Root**: no product caller or privileged listener enabled. One-time least-privilege setup, peer credentials, fixed owned operation IDs only, authenticated actor and claims; never accept raw ACL, scripts, commands, or caller-supplied root proof. No all-Project stop fallback.
5. **Real installed acceptance**: fresh nested independent Incus only, A/B/C/D dual-stack TCP+UDP/Forward, continuous C/D sessions/uptime/service, autostart default and explicit false, crash/ACK loss/ETag and simultaneous originals, boot ordering and guard survival, native poststate/ledger readback. Mock tests and earlier manual prototypes are not product proof. Keep Draft, #177 frozen, no release or merge.

## Next Agent handoff

First read #178 current remote head and exact CI, then L0 host_status, box_list, work_ref=pr178 operations and output (the list is paginated/ascending). Inspect `/root/pr178/recovery-20261123/candidate` and backup; use exact remote source and hash comparison, not its unborn index or the older symlink. Preserve originals. Verify whether any newer work exists before repeating tests. Latest Owner fresh-only decision prevails. Implement fresh admission/lifecycle next, then durable scoped stage worker (including live conntrack and direction proof), then bounded authenticated MCP Root dispatch and installed fault matrix. Keep all unsupported/unknown states blocked, test in isolated Box, record exact operation and Guest exit, commit verified slices with CAS branch update, check exact-head CI. Never use old shared-ACL migration as a requirement or touch production L0 network.
