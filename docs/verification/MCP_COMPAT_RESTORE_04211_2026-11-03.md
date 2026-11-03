# MCP compatibility diagnostics + same-host restore guard — 2026-11-03

**Candidate version:** BoxWeave `0.42.11`, based on main `c8fd5535ec984aa734a653b9d035b54075b3dd1c`. **Scope:** lightweight gateway changes; existing Containarium pin **0.120.1**. A source test result is **not** a published release or proof that an encrypted whole-Box backup was restored.

[Design](../design/MCP_COMPAT_AND_LOCAL_RESTORE_GUARD_04211.md) · [Operator instructions](../process/MCP_COMPAT_RESTORE_OPERATOR_04211.md) · [Risks](OPEN_RISKS.md)

## Implemented source scope

- On-demand `boxweave_diagnose({"source":"compat"})`: read-only, bounded pagination from upstream `tools/list`; five *known required* tools with key field/type and newly-required-argument checks; respects configured gateway tool policy. Computes a SHA-256 normalized subset fingerprint; server version is echoed only when semver was provided by upstream's existing MCP `initialize`, otherwise not verified. All changes and errors are static classifications. No auto-retry, active Box command, schema database or periodic polling.
- Protected same-host `restore_box_backup` delegation: upstream catalog entry is adapted on **every tools/list page**, hides arbitrary `age_identity_file`, replaces it with a key alias and exact confirmation. Default DENY until root-owned `/etc/boxweave/restore-policy.json` grants the *exact* backup/source/target/alias. The gateway checks ownership, safe modes, fixed key directory, regular single-link key, no forced overwrite and service-account readability before forwarding one call. The gateway does not read key contents; the **existing upstream MCP process** reads that file and sends the identity via its existing API.
- Three deliberately excluded features: no new Containarium version integration; no new whole-box **backup creation** MCP tool; no host-wide key escrow, KMS or private-key alias protocol upstream.

## Executed tests

- New `tests/test_mcp_compat.py` covers matching subset, legacy alternative connect key, missing tool/field/type/new required field, operator allowlist, paginated catalog, malformed data and untrusted text containment.
- New `tests/test_restore_guard.py` covers exact admin preauthorization, confirmation, no force, alias-only arguments, unsafe/missing/symlink/hardlink/unreadable directory/key, wrong target/backup/alias, unsupported schema, and evidence that failed preflight **never forwards** an upstream call.
- New `tests/test_gateway_compat_restore_flow.py` covers the full gateway JSON-RPC data path for compat read-only catalog, upstream MCP version capture, approved restore argument rewriting and paginated restore catalog shielding.
- `python3 -m py_compile ...`, `bash -n install.sh`, `git diff --check`, and `python3 scripts/check_knowledge.py` were executed in the new disposable `bwcompat-key-1103-container`.
- Locked Python 3.12 virtual environment, full test suite `.venv/bin/python -m unittest discover -s tests -q`: **677/677 PASS** in latest pre-documentation run (rerun after final source edits before tagging).
- A repository-shipped release and a real installed v0.42.11 compatibility/restore preflight check remain separate gates until GitHub CI and isolated installation have passed.

## Not claimed

No actual backup decryption/restore, private key movement, full Containarium MCP contract stability, end-to-end version compatibility of newer Containarium release, automatic retry, stable installer pin promotion, or user-host mutation. An authorized real restore needs a separately obtained matching encrypted backup, exact operator preapproval and valid key. This release adds **protective preflight only**. Other upstream backup/contract requests remain open.

