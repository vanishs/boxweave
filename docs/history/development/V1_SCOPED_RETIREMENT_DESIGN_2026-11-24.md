# Scoped Peer v2 retirement — I4 design review

This is a design and acceptance boundary, NOT an implemented removal procedure. PR #181 continues the #177 installation incidents after #180. Current source remains fresh-only; do not run Legacy uninstall against scoped state. No generic Root repair API is introduced.

## Why retirement is a separate transaction

Unlike Legacy, a scoped installation owns a Root guard journal, guard and lifecycle systemd units, an Incus service dependency drop-in, a local lifecycle socket, quarantine/default profile policy, per-incarnation ACLs, scoped operation journals, tombstones and retained address/peer history. Guard boot deliberately refuses incomplete policy history; removing the binary or guard while Incus still depends on it may break boot or strand protected workloads. A missing manifest alone cannot establish absence. A plain controller stop cannot prove native work is settled or policy removed.

Sources of truth: [fresh installation and boot order](V1_PR178_N2_FRESH_SCOPED_LIFECYCLE_2026-11-23.md), [N5 delivery](V1_PR178_N5_DELIVERY_2026-11-23.md), [D4 reviewed recovery](V1_PR177_D4_REVIEWED_RECOVERY_2026-11-23.md), and current source `internal/install/network_peer_scoped_install.go`, `network_peer_fresh_journal.go`, `network_peer_scoped_lifecycle_journal.go`. Use exact Root proof and independently verified inventory, not filenames or database counts alone.

## Proposed future contract (requires separate approval)

A dedicated root-local retirement coordinator should share the existing owner-reviewed plan, exact host/project/binary identity, archive and durable original-audit principles. It should not add a public MCP tool. Normal Box, Peer and Forward cleanup should prefer supported MCP operations, with original IDs and terminal evidence.

| Design boundary | Required evidence before permitting a future implementation |
| --- | --- |
| R0 — Admission | Exact scoped mode, verified immutable install, supported root journal format and matching live controller identity. Reject partial/unknown provenance, missing authority, foreign resources and unverified boot dependencies. |
| R1 — Inventory and archive | Complete exact owned Box UUID/incarnation, Peer/Forward, native ACL and guard scope, operation claims, retained tombstones, helper dependencies and private archive inventory. Report archive capacity and independent source epochs; non-atomic observations must not be described as a global snapshot. |
| R2 — Review and fence | Fresh digest, explicit local approval, durable original retirement ID, exclusive admission/maintenance fencing with no new scoped mutations or unresolved originals. Revalidate all evidence before effects. Existing unrelated workloads and shared Incus remain out of scope. |
| R3 — Retire owned workloads | Only original exact owned identities and verified supported lifecycle operations; no guessed replay, age-based unlock, arbitrary native deletion or force stop of unrelated Boxes. No transition to guard retirement until both native inventory and protected operation history prove the relevant scoped consumers are settled. |
| R4 — Native and boot dependency retirement | Separate reviewed transaction for exact owned quarantine/profile/ACL state and kernel guard/boot dependency lifecycle. Prove the guard is never removed while an owned consumer may require it. Preserve foreign settings and shared Incus. No blanket ACL, nftables or conntrack flush; no unreviewed daemon restart. Failure retains evidence and a safe fencing state. |
| R5 — Controller and archive closure | Independently verify exact owned helper, unit, socket and binary identities, native poststate, absence of scoped consumers and retained evidence. Archive private receipts and outputs without reusing old UUID/address policy history. Close only the original audit after verified completion. |

A future plan must distinguish read-only preview, approved execution, durable partial receipt, review-required recovery and completion. Repeated same original request must return the original outcome or safe navigation; it must not reissue uncertain native changes. A failed proof is a blocker, not authorization to remove files. No phase above is executable in this PR.

## Acceptance matrix for the future PR

- Empty, active and previously retired scoped installations; exact incarnation and retained history.
- Foreign Project/profile/ACL/guard/boot drop-in, unknown modes, inaccessible paths, dangling symlinks, changed review, incomplete Root history and interrupted transactions.
- Fault injection at every persisted phase, lost acknowledgement, reboot before/after each boot-dependency change, and recovery without duplicate effects.
- Verify live unrelated TCP/UDP and native service continuity; unchanged foreign state, original autostart policy and host confinement; never equate service status with packet evidence.
- Both architectures, clean physical VM and nested test evidence where supported. Independent root/daemon/MCP observations must agree. Preserve actual physical Host acceptance separately.

## Current I4 evidence (no destructive implementation)

Existing installed scoped Box `febeb346-d924-40e0-83fc-99b4e984b16f`, exact checked wrapper source `19e81ddc033e4931fe6c4692fec392a3746a5352`: original operation `8a71ccaf95122b59b40ba7e49936e757`, guest exit 0. All three scope previews and original-resume preview refused with the scoped-mode message; controller/guard/lifecycle PID and active states and five key binary/manifest/unit/drop-in hashes unchanged. No uninstall audit created. Root journal contained four child directories; native service Requires/After included the guard. This is actual refusal/unchanged-state evidence, NOT scoped removal or reboot acceptance. Existing synthetic scoped journal/manifest/residual-path refusal tests remain mandatory.

Decision: the design is sufficiently specified to review a separate implementation PR. Do not enable scoped uninstall/upgrade/rollback in PR #181. Owner approval of the future destructive transaction design is still needed. Legacy optimization and final I5 CI must not be represented as scoped retirement support.
