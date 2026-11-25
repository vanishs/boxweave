# PR188 H4A - MCP errors and recovery navigation

Base: 42ad6d63146cdfab0e394cf614f11a568c8704f1 (H3B). Exact owned Box A (9d374e18-196a-493d-bf1f-2d582589f04d), separate source worktree, original installed RC13/RC14 fixtures retained.

## Box evidence

- Authenticated official Go MCP SDK over HTTP and real temporary SQLite: schema/key byte limits, redacted error metadata/text equality, exact same-layer blocker, UNKNOWN inspect/no new admission, foreign/absent equivalence and original state navigation. Original dcf552442f66b9d3f99ee7d6a4fbab7a: exit zero.
- Admission predicates remain in the existing transaction; only proven same-layer original identifiers are exposed. Direction and address capacity have distinct durable classifications. No UNKNOWN release or speculative native retry.
- Focused regression a534659a558a7e6fd82033e96dfcde33: exit zero.
- Full regression b6c6aa49fb6da2b11c33f6bd795ecaae: state (145.073s), recovery, diagnostic and CLI passed; one control handshake compactness assertion failed. Reduced first paragraph while preserving safety language and the 512-byte constraint; no test relaxation. Rerun deb119c762ae725a94ba9f6d10d14d01: control, recovery, diagnostic, CLI all exit zero. State source unchanged since its full pass.
- Baseline Box B read-only continuity check fb41eb18c897bf9fbf02469386252ca1: exit zero, controller active PID 5487, n3-a/b/c/d RUNNING, Forward HTTP 200. This checks retained RC14, not installed H4A.

## Independent installed CI

All four lifecycle lanes must prove authenticated preflight, 23 tools, byte bounds and redaction, and no invalid admission. Scoped N2 and Legacy D4 must prove long real guest Exec, exact-key duplicate, normal wait linked to the original, terminal output and one admitted follow-up. Scoped and Legacy D4 additionally require exact UNKNOWN blocker -> inspect -> SAFE_RECOVER reconciliation hint, before existing reviewed apply. Existing native peer/forward, recovery, retirement and purge gates remain required.

Box testing exercises actual candidate SDK and ledger with controlled original state, not a newly installed candidate on a native clean host. Inherited nested LXC restrictions and retained fixtures make independent clean-VM installation and positive native effects the final gate. No existing installed binary, Host policy or fixture credentials replaced; no merge or release. H4B capacity design/reclamation is out of scope.
