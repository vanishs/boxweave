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


## 2026-11-03 published release and real installed acceptance

**BOTH RELEASE AND LIVE GATES PASSED WITH SPECIFIED LIMITS.** This supersedes the candidate-era pending-live statements above, not the exclusions for real encrypted restoration.

- Code [PR #55](https://github.com/vanishs/boxweave/pull/55) merged; PR CI run 41708327533 SUCCESS; main source commit 4f5ffe7fae586c79e6d75ed90560ee196e618ec3; main verification and publication CI run 41708839793 SUCCESS.
- [Release v0.42.11](https://github.com/vanishs/boxweave/releases/tag/v0.42.11) published, non-draft and non-prerelease with 23 assets, including mcp_compat.py, restore_guard.py and release.json. Default PUBLISHED_RELEASE stayed **0.42.1**.
- The new isolated Box bwcompat-key-1103-container (Ubuntu 24.04, 2 CPU, 4 GB RAM, 30 GB disk, nesting enabled only on this Box) installed official v0.42.10 to phase=ready, then official versioned v0.42.11 to phase=ready, standalone source 4f5ffe7fae586c79e6d75ed90560ee196e618ec3. The local Containarium daemon remained **v0.120.1**.
- The installed mcp_compat.py and restore_guard.py files were present in the official release directory. sudo boxweave doctor returned exit 0, overall=passed, **8/8 required checks passed**. Its optional authenticated_mcp check was not requested, so no such claim is made from doctor alone.
- A separate **real installed stdio MCP exercise** executed the gateway as the BoxWeave service user and sent initialize, initialized notification, tools/list, source=compat diagnostic, and a deliberately nonexistent/unauthorized restore request. The gateway exited 0, returned all four requested responses, and produced no parse errors or stderr.
- Initialize returned gateway version **0.42.11**. Actual tools/list advertised **67 tools**, including boxweave_diagnose but **not restore_box_backup**. The source=compat result returned MCP_COMPAT_SUBSET_MATCH, compatible_subset, and matching_subset on each of list_containers, get_container, connect, create_container, debug_container. SHA-256 fingerprint was e9a4592ed6e79db40632a17eb17484f0f58df2fcffcaa5cba518e8b9476b9a7e. The upstream MCP server version came from initialize.serverInfo.version=0.120.1. Runtime result semantics remain not_verified.
- The deliberately unauthorized restore call returned JSON-RPC code -32602 and RESTORE_KEY_OR_POLICY_UNSAFE with **no successful restore result**. This validates local rejection; it does not demonstrate an actual encrypted backup or successful restore.
- **Open capability gap:** the **installed pinned Containarium v0.120.1 MCP runtime does not advertise restore_box_backup**. The BoxWeave gateway guard is present and its simulated approved path and default-deny behavior are tested, but **approved end-to-end encrypted whole-Box restore cannot be claimed available on this pinned runtime**. A compatible upstream Provider release, matching backup/key and separately authorized restoration test must be validated before claiming recovery.
- Full Python regression suite on the isolated source checkout: **678/678 PASS**; release/source inventory, knowledge checks and GitHub CI passed. No change to the original host Provider, no automatic upstream upgrade, no stable pin promotion and no real recovery of user data.

**Scope left open:** full cross-version interface stability, real encrypted restore/verification, whole-Box backup create MCP parity, remote-host key references, and external OAuth/MCP-client acceptance.
