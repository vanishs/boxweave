# S4-B2  exact root-only conntrack review plan (2026-11-17)

**Status: S4_B2_PARTIAL. B3 not started.** This change is a read-only operator verification prerequisite. It is not an ACL write, root permission delegation, an MCP Peer Tool, or an authorization to delete conntrack entries.

## Design

- `native-peer-purge-plan` is a host-root local CLI accepting only a protected existing SQLite `--db`, exact `--project`, `--network`, `--acl`, and `--peer-source-uuid`/`--peer-target-uuid`. It does not expose shell or arbitrary IP selectors.
- The root-side service Store must be the separately validated read-only handle (`mode=ro`, `query_only`); writable `state.Open` is now rejected by the root-scoped conntrack purge prerequisite and by the new plan reviewer.
- The review requires the complete SQLite PINNED owner/MAC/IPv4/IPv6 and RESERVED dual-stack fence ledger, the real Incus global bridge/ACL/Project/Profile/Box inventory, all protected native Boxes stopped and autostart-disabled, no live Linux Bridge ports, and exact bidirectional IPv4+IPv6 DROP. It repeats root identity/stop/port and SQLite readback before returning.
- The SHA-256 plan binds a host machine-ID digest, ordered full SQLite identity set digest, root native topology/ETag digest, exact source/target UUID and direction. `mutation_authorized=false` is always returned. There is deliberately **no apply CLI**: a durable no-replay UNKNOWN journal, host-root executor boundary, explicit local operator approval, and independent verifier must be implemented before any write is allowed.

## Validation

- Source and pure unit tests require changed host, native ETag digest, direction, MAC and IPv6 identity to change the plan hash. Foreign/duplicate pairs and missing digests reject.
- The CLI missing-database test proves it cannot create state or proceed on an unprepared host.
- A disposable Incus 7.6 root test is attached to the existing fully stopped A/B/C fixture: it creates only a temporary private SQLite ledger, reopens read-only, demands a root/native verified plan and tests refusal of writable DB, foreign UUID and poisoned MAC. Its explicit PASS markers are mandatory in the workflow, **but this exact new test still requires a fresh CI run**.
- Existing shared L0 host ACL/Scriptlet, previous boxes and user resources are not mutated. The security review is not a lifetime lock against unrelated physical-root writers.

**Still missing for B2 PASS:** durable host-root/control-plane handoff and unknown-journal, true installed conntrack live-session teardown and foreign-flow preservation, effective kernel/nft per-NIC rollback proof, installed Scriptlet parity/failure recovery, final independent same-HEAD verifier. Do not start B3.
