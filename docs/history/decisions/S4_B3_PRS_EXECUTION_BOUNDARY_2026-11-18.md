# S4-B3 decision — extend the B2 protected PRS executor (2026-11-18)

**Status: APPROVED architecture; implementation and B3 native acceptance still pending.** Sole Draft PR #153. PRS means installed `peer-root-serve`, **not** a general root proxy.

## Decision and why

B2's installed `internal/install/native_peer_root_integrated.go` is already an **atomic-in-one-root-executor operational sequence**, not merely a privileged checker: independently review an exact claimed SQLite original and native inventory; fsync an original no-replay journal; stop **all** owned shared-ACL NIC consumers; prove isolated bridge; recheck route/UsedBy/ownership/ETag; perform Incus SDK `UpdateNetworkACL`; exact ACL readback; conntrack handling; per-NIC nftables verification; and selective verified restart. B2 formally accepted a **revoke-only** scope. The native failure experiments show that ACL GET/ETag success is insufficient if one NIC fanout fails, so the isolation barrier must precede all grant effects.

**Keep grant and revoke inside the same restricted PRS execution boundary.** Do **not** split it into `PRS stop → controller/Scriptlet ACL PUT → PRS inspect/restart` merely to relocate SDK permissions: that adds a multi-process barrier owner, crash/lease recovery protocol and new TOCTOU conditions without replacing the B2 safety obligations. This is an architectural preference for the present v1 safety case, not a claim that Scriptlet can never be used elsewhere.

## Strict separation of responsibilities

| Controller / state (unprivileged) | PRS (host root, bounded execution) |
| --- | --- |
| Authenticate MCP Actor/token and admin role; approve a directional source→target, Project, UUID+incarnation, bridge/ACL; reject foreign/unowned peers | Validate **only** one operator-approved original and plan SHA; local Unix `SO_PEERCRED`; no caller-selected IP, ACL, arbitrary paths or shell |
| Admit an original idempotent Operation; hold ACL + both Box claims; durably stage PENDING grant/revoke; compute **entire** intended allowlist | Independently re-read canonical SQLite original/claims/ledger and complete native owner/route/global scope; bind plan to all claimed peers and pre-ACL ETag |
| Maintain ledger, retained address/Box leases, UNKNOWN/recovery decisions and caller-visible status | Own one durable no-replay original journal; stop entire protected Project before any native ACL PUT; retain isolation and claims when uncertain |
| Do not attest native permission from SQLite or ACL GET alone | SDK ETag GET/PUT + exact post-readback; verify effective per-NIC IPv4+IPv6 nft/forwarding; safe conntrack handling; selectively restore only after evidence |

## B3 implementation contract

1. **Do not weaken B2 revoke.** Introduce a carefully typed grant branch distinct from the existing `ReadRootPeerRevocation` (which enforces `allow=false`). Grant preflight must require a canonical `allow=true` original in DISPATCHING, exactly one PENDING grant row, exact Actor/admin/token, ACL+2 Box claims, complete PINNED NIC/IPv4/IPv6/MAC fences, no foreign attachment/overlap, and no unresolved peer or sibling Operation.
2. Read the **whole durable confirmed+one-pending** ACL snapshot. Compare native current ACL to the *confirmed-only* compiled baseline before isolating; compile target complete allowlist, not single-peer overwrite. After isolation, recheck same native ACL ETag/Project/lease/fence/claims snapshot; then SDK PUT and readback. Refuse host policy that differs even if it looks permissive.
3. On grant resume, verify exact **directional** A→B dual-stack rules and no B→A or C→B grants. B2's revoke-oriented `nativePeerRootVerifyNFT` expects DROP and cannot be reused unchanged as grant success. Validate actual IPv4/IPv6 TCP+UDP connectivity and negative directions in independent native test. Review conntrack semantics against desired rules; do not mark PASS by ACL GET.
4. Use the existing original-scoped root journal with fsync-before-effect, exact approval SHA and single-dispatch. On any uncertain stop, PUT, partial NIC fanout, packet proof, lost ACK or crash: durable UNKNOWN, retain all endpoint/ACL claims and pinned addresses, **no replay or implicit release**. Only independently evidenced completed journal permits read-only reconcile.
5. Expose `box_peer_set` over the installed controller MCP only after Root's grant+revoke execution, authorization, status transitions and native safety tests are complete. No universal `root_allow`, unconstrained Scriptlet or root parameter forwarding.
6. Sign B3 only after focused/race/full verify and **fresh Incus 7.6 native** IPv4/IPv6 TCP/UDP positive+negative, ETag/fanout/ACK loss/reboot/foreign Project/identity/UNKNOWN evidence, original receipts and exact cleanup. B2 PASS does not imply B3 PASS.

## Evidence anchor

B2 independent scoped signoff: `docs/verification/S4_B2_FINAL_INDEPENDENT_AUDIT_2026-11-17.md`; B3 already committed SQLite stage/snapshot/pure compiler: `docs/development/S4_B3_PROGRESS_2026-11-18.md`. This decision changes **architecture direction**, not the public capability or acceptance status. Current B3 remains PARTIAL until tested.
