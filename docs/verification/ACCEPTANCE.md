# v1 real functional and security acceptance matrix

**Status:** tests specified only; no v1 runtime or live PASS evidence exists. This matrix covers the current **20-tool** permanent product, not the deleted Caddy/general ACL/patch/diagnostic capabilities. Do not reduce security verification merely because the product is smaller.

Evidence: **T** typed/unit simulation; **I** real installed black-box MCP/CLI; **N** real native/nested Incus; **V** independent native post-read/Verifier. Every item remains UNTESTED until actual evidence.

## Required functional cases

| ID | Exact requirement to test | Evidence |
|---|---|---|
| A-001 | Authenticated loopback HTTP MCP initialize/list/call; missing/revoked admin and viewer credentials denied appropriately; optional operator Tunnel not managed by BoxWeave | I |
| A-002 | Local install/status/upgrade/bootstrap/doctor and protected credential recovery; no remote host admin route | I |
| A-003 | Viewer role may read owned Box metadata/files/output but cannot create/update/delete/Exec/peer/forward/cancel; admin only manages owned resources | I + V |
| A-004 | Exact native Box UUID/incarnation owned identity for list/get/create/read; no foreign host/Incus file or Box disclosure | I + V |
| A-005 | Create reviewed Ubuntu guest with defaults, start=true verified ready; start=false leaves stopped; opt-in nesting | I + N |
| A-006 | CPU/memory/safe disk growth, nesting/internet typed update, rejection of shrink/raw Incus/host mount/device | I + V |
| A-007 | Start, stop, delete with explicit destructive confirmation and actual native after-state readback | I + V |
| A-008 | Reused Box name/IP or new incarnation cannot inherit former ownership/peer/forward claims | I + V |
| A-009 | Agentless guest-root Exec typed argv/cwd/env/timeout/outcome observed, with no host shell or installed management client | I + N |
| A-010 | Exec client disconnect/reconnect recovers same original operation/outcome, bounded stdout/stderr and observed nonzero exit as completed failed command | I + V |
| A-011 | Guest file list/read bounded, meaningful hash/coverage, no host/foreign special-file access | I + V |
| A-012 | `box_file_apply(create|replace|delete)` only; safe revision and native post-state, no internal patch engine or recursive deletion | I + V |
| A-013 | Concurrent guest writer, unsafe path/symlink/hardlink/device and stale hash reject unsafe file changes; no fake atomic CAS | I + V |
| A-014 | Guest Git/Exec can clone and edit a normal code repository; no required BoxWeave Git/upload/patch service or leaked private credentials | I |
| A-015 | Full create -> clone/read -> edit -> failing Go test -> inspect original output -> fix -> passing test -> independent native readback | I + N + V |
| A-016 | Every mutation requires caller reusable idempotency key, rejects missing/conflict, returns original Tracking+Operation on same retry without duplicated effect | T + I |
| A-017 | Go context/Tracking ID available throughout MCP/domain/SQLite/worker/native adapters with separate request/idempotency/Incus IDs | T + I |
| A-018 | Crash/lost native result/timeout preserves UNKNOWN and original intent; no blind redispatch | T + I + V |
| A-019 | `operation_get(wait_ms)` read wait is bounded, returns changed state and never cancels original native work on timeout/disconnect | I |
| A-020 | `operation_list` and `operation_output_get` scope/retention/truncation protect original output and ownership | I + V |
| A-021 | `operation_cancel` proves queued-undispatched cancellation; active Exec stop is only claimed with native termination proof; unsupported and UNKNOWN are honest | I + V |
| A-022 | Redacted phase/error/verification summary in `operation_get`, early failure with Tracking ID and local CLI logs; no `diagnostic_get` endpoint | I |
| A-023 | Private direct Box A->B IPv4 and enabled IPv6 denied by native evidence under supported Bridge/OVN NIC topology | I + N |
| A-024 | Unsafe network topology or IPv6 bypass fails closed; any controlled IPv6 disable leaves foreign host/network unchanged | I + N + V |
| A-025 | `box_peer_set(A,B,true)` allows directional private A-initiated connections/replies; `false` revokes; B independently initiating A remains denied unless granted | I + N + V |
| A-026 | Native Box IP change, identifier spoof, unmanaged instance and foreign NIC/ACL cannot acquire or lose another Box's peer rights | I + N + V |
| A-027 | Internet egress defaults allowed; per-Box deny toggle works without changing unrelated host networking | I + N |
| A-028 | `port_forward_create` maps owned Box TCP service to explicit host port (loopback by default when supported), external bind needs explicit admin action | I + N + V |
| A-029 | UDP forward uses verified native Incus facilities, host port collision/unsupported network denied, no arbitrary target IP/native proxy takeover | I + N + V |
| A-030 | `port_forward_list/delete` show/remove only owned forwards, survive/reconcile service restart and never disturb foreign forward | I + N + V |
| A-031 | No Caddy daemon/config/installer dependency or published HTTP route tools; operator-managed proxy/Tunnel can separately consume native host-published port | I + V |
| A-032 | Existing compatible Incus reused with restricted BoxWeave Project, actual version/extensions recorded, foreign projects and instances untouched | I + V |
| A-033 | Missing/old approved Incus installed/upgraded by explicit local workflow; shared disruptive update requires separate approval, truthful failure/rollback and no automatic host reboot | I + V |
| A-034 | Corrupt/lost ownership SQLite blocks dangerous mutations, independently reviewed local recovery preserves foreign resources | I + V |
| A-035 | L0/L1/L2 use same installed BoxWeave code and 20 MCP tools if supported; each manages only its own Boxes and native ports, no parent token/socket inheritance | I + N + V |
| A-036 | An inner BoxWeave candidate can be installed/upgraded and run complete Agent loop without changing the outer layer or Caddy (absent) | I + N + V |
| A-037 | All installed MCP tools and local CLI leaves receive positive/negative tests, native after-state/oracle and explicit SKIP/BLOCKED/UNKNOWN outcomes | I + V |
| A-038 | Exact verified Go SDK/native package versions, binary hashes and immutable GitHub new-version release with no automatic live host deployment | T + I |
| A-039 | Initial Ubuntu LTS guest works; any additional guest distro is separately validated before added to supported images; unsupported cases honest | I + N |
| A-040 | All excluded product features are absent: no generic ACL editor, per-Box RBAC, Caddy/domain/HTTPS, native snapshot tools, automatic wake or guest text patch engine | I |

## Adversarial combinations

Test lost MCP reply after create/Exec/forward, concurrent Box delete vs peer/forward edits, token revocation just before and after dispatch, stale guest hashes and symlink races, fixed network ACL vs peer grants, IPv6 bypass, host listener and port collisions, simulated parent restart during nested work and existing shared Incus upgrade. No native effect outside the owned set is ever permitted.

The independent Finder/Fixer/Verifier harness belongs in repository/CI/disposable Boxes, not customer runtime. Record tool name, original Tracking/Operation IDs, exact source SHA, installed bytes, native Incus version, outcome and evidence. No false PASS from a mock.

See [scope](../product/SCOPE.md), [tools](../design/MCP_TOOLS.md), [network](../design/NETWORK.md), [permanent boundary](../decisions/PRODUCT_BOUNDARY.md).
