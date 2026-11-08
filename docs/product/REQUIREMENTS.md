# Functional requirements — narrow v1

**Status:** approved design goals, NOT implemented facts. Supersedes broader numbered requirements previously proposed on the design-only branch.

| ID | Required outcome |
|---|---|
| R-001 | One controller and restricted Incus ownership domain per host/layer; implementation Go + official MCP/Incus SDK; no second management protocol or Containarium. |
| R-002 | Authenticated Streamable HTTP MCP binds loopback by default. Operator configures public HTTPS/Tunnel separately. |
| R-003 | Host-local CLI installs/upgrades/diagnoses/repairs and bootstraps multiple independently revocable `admin` or `viewer` tokens. Viewer read access can reveal sensitive guest data. |
| R-004 | Exact native Incus UUID/incarnation checked for every sensitive Box, port, peer, operation; foreign host/resources remain inaccessible through MCP. |
| R-005 | Basic owned Box create/list/get/update/start/stop/delete; default Ubuntu LTS guest, start=true/readiness observed, safe CPU/memory/disk growth/nesting/internet changes; no raw Incus settings or shrink. |
| R-006 | `box_exec` uses native Incus guest Exec as guest root by default, typed argv/cwd/env/limits, and never host Exec/SSH/guest Agent. |
| R-007 | Native guest file list/read and `box_file_apply(create|replace|delete)` for one regular file; safe original hash/revision and no false atomicity. Complex patching uses guest Git/Exec; no patch engine. |
| R-008 | Every mutation uses caller-provided reusable idempotency key, durable async journal, server Tracking ID, separate BoxWeave/native Operation IDs, scoped native post-state verification and UNKNOWN recovery without blind replay. |
| R-009 | `operation_get` has bounded long polling and a small redacted phase/error/evidence summary; `operation_output_get` has bounded original output and exit; cancellation is proof-based. No diagnostic MCP product. |
| R-010 | User-independent nested BoxWeave installations have the same supported capabilities; no cross-layer controller/forward coordination or inherited privilege. |
| R-011 | Native IPv4 and enabled IPv6 private Box-to-Box traffic denied by default; unsafe topology fails closed. `box_peer_set` allows/revokes a directional UUID pair only; no generic ACL DSL. |
| R-012 | Native Incus TCP/UDP forwards map exact owned Box UUID+guest port to explicit authorized host bind/port. No generic target IP/proxy daemon, and public bind requires explicit approval. |
| R-013 | No Caddy dependency or managed HTTP hostname/site route/TLS/certificate/redirect, and no separate managed web server. Operator external proxy/DNS/Tunnel is outside BoxWeave. |
| R-014 | Explicit local install/upgrade records exact BoxWeave/Go/Incus/OS versions and capabilities, installs/upgrades approved dependencies, reuses compatible existing Incus, and requires distinct approval for disruptive shared-daemon changes. No auto-reboot or foreign instance adoption. |
| R-015 | Initial officially supported host/guest Ubuntu LTS; certify other Linux guest distributions incrementally with actual native tests; no universal image claim. |
| R-016 | Correctly classify authorization/validation, native timeout, native error, observed Exec nonzero exit, postcondition uncertainty and retained UNKNOWN. No invented success. |
| R-017 | Permanent exclusions include snapshots/backup, multi-host, VMs, OAuth browser login, RBAC/per-Box token delegation, persistent background process manager, idle sleep/wake, guest file search/bulk/patch engine, Caddy and generalized ACL. |
| R-018 | Black-box test every exposed MCP and local CLI leaf and negative boundary; test complete Agent edit/build/fix/retest loop and L0/L1/L2 where supported; independent verifier does not become customer runtime. |
| R-019 | CI auto-publishes real verified new-version source build/artifact for `1.X.Y` on `v1`; does not deploy/upgrade hosts or publish docs-only pseudo-release. |
| R-020 | No implicit access to privileged Incus socket or host admin API by remote MCP; use a technically enforced restricted native/host-local setup boundary. |
| R-021 | A confirmed `box_delete` must clean and independently verify **all BoxWeave-owned dependent native port forwards and peer grants** under the exact Box UUID, with durable substeps and safe partial/UNKNOWN recovery; foreign resources must remain untouched. |
| R-022 | Existing `operation_get` must provide bounded, sanitized per-phase native error/observed dependent state and next safe diagnosis; `host_status` identifies exact running product/build. No additional MCP host-log tool. |
| R-023 | LAN service publishing requires an explicit reachable host LAN bind and independent probe from another LAN machine; guest-native service persistence and guest listener reachability are separate conditions. |
| R-024 | Inner localhost-only MCP is tested by an authorized guest-local client inside the nested host; physical LAN access to an inner service requires independent, explicit per-layer native forwards, never automatic cross-layer management. |
| R-025 | CI automatic formal publication of a real new-version bug fix is gated on original repro, exact candidate binary/source hashes, **independent real nested/native regression and negative acceptance before release**. Candidate may be installed in an isolated Box before formal publication; old host updates are always explicit. |

See [permanent boundary](../decisions/PRODUCT_BOUNDARY.md), [tool catalog](../design/MCP_TOOLS.md), [verification](../verification/ACCEPTANCE.md), [security](../design/SECURITY.md).
