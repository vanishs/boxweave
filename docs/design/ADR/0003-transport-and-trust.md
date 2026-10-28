# ADR-003 — Retain upstream transport and separate trust planes

Status: selected for phase 4 validation. Inputs: REQ-001, REQ-002, REQ-003, REQ-016, REQ-018; SRC-008, SRC-010, SRC-026.

Options: preserve upstream restricted SSH/MCP and add fixed REST application adapter; replace with custom MCP gateway; expose a new general remote HTTP/shell proxy. Select the first. It preserves provider metadata and avoids an unnecessary credential-bearing gateway. CLI/Web plans govern their own operations; they cannot restrict direct upstream Agent calls. Single trusted operator accepts the disclosed residual upstream authority; no team-RBAC claim is made.

Windows native and explicit WSL SSH are distinct profiles with separate key/trust stores. Pin host identity, require authenticated MCP lifecycle/read-only call, verify workload identity independently. No disabled verification, known_hosts shortcut, host credential edit or implicit provider upgrade. Web has its own scoped provider credential and HTTPS/session plane; neither borrows client GLM secrets.

For setup integrity, choose operator-delivered trusted descriptor with immutable source + exact payload digests and public host pin. Verify downloaded payload before execution. Alternative bare commit pinning is insufficient for the accepted verification contract; a signing/PKI release system could improve distribution later but adds initial operational burden. Trusted descriptor delivery is an explicit bootstrap assumption, not cryptographic proof from an unauthenticated checksum endpoint.

Validation: TASK-003 and TASK-012 tamper/trust/config merge cases; real native/WSL handshake and per-Agent registration in TASK-005/013. Reject unsupported forms safely. If the provider can't bind a target strongly enough, don't infer it from IP/name; revise target contract before execution.
