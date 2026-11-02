# Phase 2 R-02 — Gateway task receipts and explicit provisioning

**State:** candidate v0.32.0 source; installed runtime remains unverified. Related designs: [MCP Task Reliability R1](MCP_TASK_RELIABILITY_R1.md), [Master Roadmap](FEATURE_ROADMAP_MASTER.md).

## Delivered surface

- boxweave_task_submit accepts a trusted Box, 32-character random hexadecimal request_key, and bounded exec; it requires existing connect permission and delegates through the original trusted connect adapter.
- boxweave_task_get by task_id or request_key, and boxweave_task_list by Box, return durable metadata only; no command body or stdout/stderr persistence.
- Gateway-private SQLite path /home/boxweave/.boxweave/gateway-tasks.sqlite3 is 0600 under a user-owned 0700 directory. It does not grant access to Web/app receipts or OAuth tokens.
- Setup is explicit: sudo boxweave task provision. The root-owned host CLI validates the fixed root helper and invokes it as the unprivileged boxweave service user. The Gateway never creates persistent storage.

## Persistence and failure semantics

The gateway commits the task record and a dispatch_intent transaction before any remote send. Duplicate requests with identical key/payload/exact observed Box incarnation return the original task ID; changed arguments or target return REQUEST_KEY_CONFLICT. After a crash at dispatch_intent, the lookup conservatively reports UNKNOWN and NEVER redispatches. Known nonzero exit status is completed with its actual code. An old, untracked connect(exec) call is neither idempotent nor covered by this new receipt.

The initial capacity is 10,000 metadata rows and full output retention is disabled. No automatic deletion or expiration silently permits redispatch. The current authority model remains one trusted operator; request keys do not grant authorization.

## Acceptance boundaries

Full fixture CI must cover concurrent SQLite key races, process/reply loss, malformed schema, unsafe modes, exact source release manifest, deny-before-forward tool policy, and synthetic gateway recovery. None of these proves a live guest Box, external OAuth client, installed file ownership, mixed version upgrade, or root CLI execution. Those are LIVE U until separately observed.

R-04 cross-entry Web/SSH/HTTP resource guards, optional Box leases, P1-B2, guarded BOI file edits, stdout/stderr replay and full live acceptance remain separate work. There is NO global conflict-safety claim in R-02.
