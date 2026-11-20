# S4-B2  SQLite-bound scoped conntrack prerequisite (2026-11-17)

**S4_B2_PARTIAL; B3 NOT STARTED.** This is a root-only internal prerequisite, not an MCP command or a network grant.

`NativePeerConntrackPurgeFromStore` derives its complete A/B/C UUID, MAC, canonical IPv4/IPv6 and bridge/ACL target exclusively from the protected SQLite network ledger. Caller input can only select two distinct native UUIDs already in that ledger. A missing/foreign UUID or quarantined/unpinned lease rejects before root native work.

After the existing host-root stopped-Box, Incus UsedBy, per-NIC identity, exact DROP, empty bridge-port and narrow conntrack checks, the full SQLite identity snapshot is re-read. Any change refuses a successful result rather than allowing resource release. It does not replay uncertain native deletion, flush conntrack globally, elevate the MCP service or enable B3.

Local test cases require the complete three-Box ledger, reject invalid pairs and a quarantined dual-stack lease. Test database parent is explicitly private mode 0700. Independent installed end-to-end / race / true per-NIC nft / root-to-worker handoff remain open. There is no `S4_B2_PASS` yet.

**Local quality receipt:** Guest Exec `891900906046ad0282e0d5b433a4c72a`, original native `a9b6d782-5e1a-4e90-8c1b-2462ea7116b3`, `SUCCEEDED` with full `make verify`, Go race, vet, YAML/format, module checks and Python 7/7.

**Separate baseline native witness:** [CI #44147086095](https://github.com/vanishs/boxweave/actions/runs/44147086095), job `134265057852`, SHA `f7a4455f45586c5eed59947b7906ed6b4d865269`, SUCCESS with zero global flush, native root SQLite identity/MAC/quarantine refusal, raw host NIC/kernel port refusal and exact cleanup. All four later legacy diagnostic pair deletes logged `EMPTY_OR_UNCERTAIN`; they do **not** prove four established sessions were deleted or establish installed root/SQLite worker acceptance. The new FromStore API itself has local tests only.
