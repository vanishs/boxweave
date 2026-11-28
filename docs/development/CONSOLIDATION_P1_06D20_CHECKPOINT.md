# P1-06d20: Scoped maintenance database inspection

Base: f1b848436eff00ca80301cb239690d5fddd31aa4.

- Require schema 6 and empty legacy retirement, queue and compaction tables for Scoped SQLite inspection before any retirement effects.
- Preserve this mode through cold copies, planning, revalidation, resume and Drain capture. Legacy inspection retains its historical schema and compaction semantics.
- Keep the current schema and historical digest shape; a schema transition requires a separate coherent runtime/DDL/fixture change.

Box evidence (9d374e18-196a-493d-bf1f-2d582589f04d):

- 4c8bc8123315cd23871d1597fbf9397d: exact parent sync and call inventory, exit zero.
- 0586a73bca273a7f0ea81eb9550c265f: edit, diff check and Python compilation, exit zero.
- 628754cf8d9b9972df60b0228a720481: new cold/live synthetic SQLite refusal and unchanged-source regression, then 75 uninstall/Drain/offline/LXC tests, all pass. No real host effects.

Independent CI pending; no schema deletion, final P1 acceptance, merge or release.
