# S4-B2 kernel Bridge port census (2026-11-17)

**S4_B2_PARTIAL; B3 NOT STARTED.** Root-only security prerequisite; never a public MCP network write.

An Incus UsedBy audit can miss a host-root direct link attached to the Linux bridge. When the whole shared ACL consumer set is stopped and autostart-disabled, the dedicated bridge must contain **zero** live kernel ports.

The new bounded `nativePeerBridgePortsEmptyAt` reads `/sys/class/net/<bridge>/brif` without a shell. Missing inventory, invalid bridge, cancellation, or even one foreign port causes rejection. The root-only `NativePeerConntrackPurge` checks this before its exact native A/B/C identity/ACL/DROP/conntrack procedure and again afterward. It does not call a global flush, expose host root to MCP or grant Peer access.

Unit tests check empty/foreign/invalid/unavailable bridge states. The disposable GitHub Incus 7.6 native test attaches a host-created veth outside Incus to the bridge after the three-Box stop fence, requires denial, removes it and requires zero ports. The workflow demands non-SKIP test markers.

This is **not** a lifetime host lock: trusted root can race after the final read. Effective per-NIC nft validation, service SQLite/root handoff, installed Scriptlet, failure recovery and independent complete B2 verification remain required. No B3.

Local final quality: disposable L0 `make verify` guest Exec `420a6f82798718cb7f5b270581dc416c`, native `6cc24d02-6b5e-4294-86b8-47796bb76b81`, both `SUCCEEDED`; Go race, vet, YAML, Go formatting, module checks and Python 7/7 passed. The native root port-link fixture still needs its own independent CI result.
