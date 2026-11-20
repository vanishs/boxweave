# S4-B2 - protected SQLite -> real Incus root conntrack!-> one-shot journal (2026-11-17)

**S4_B2_PARTIAL; B3 NOT STARTED.** Test-only integration, not permission to enable Peer MCP writes.

## Design

Previously, the fsynced root DISPATCH_UNKNOWN/COMPLETE_VERIFIED journal ran only synthetic effects while native Incus and real Linux conntrack were tested independently.

The refactor introduces a shared internal root SDK implementation for the full protected read-only SQLite ledger!-> root Incus scope audit!-> stopped A/B/C barrier!-> zero Linux bridge ports!-> dual-stack exact DROP!-> four directed conntrack scopes!-> bridge port/SQLite revalidation. The production entrypoint preserves root EUID, exact boxweave service UID/project binding and fixed root Incus Unix socket. Only disposable CI injects the independently authenticated SDK client and fixed conntrack command.

The existing fresh Incus 7.6 native three-Box test now runs the *actual root conntrack kernel executor* inside the durable one-shot journal, verifies all four scopes and native digest, reads the persisted COMPLETE_VERIFIED result, denies a second dispatch, and rejects the ordinary writable service Store. New mandatory CI markers make a skipped test fail. It changes no shared L0 ACL or Scriptlet.

## Scope and limitations

New disposable L0 Box: 2c735208-1c89-46c8-b2ab-3eaba6f9ca7d. Original Create be3ae73d224cf12b8234d0b62b284fac, native 45e7105e-e607-46f9-a962-16702edbb570; source base 32b2fb0d34ebaf930c00da746a2f8d79f52b6372.

This remains a stopped-peer root native CI proof, not installed live TCP/UDP teardown, per-NIC nft reload failure recovery, production root/agent admission or approved Scriptlet parity. There is no root apply CLI, MCP write authorization, new PR, merge or RC. Independent new-HEAD native CI and verifier evidence are required before acceptance.
