# Security and ownership — small fixed policy

**Invariant:** A BoxWeave MCP caller, with any valid management credential, can inspect or mutate **only** this layer's BoxWeave-owned Incus resources and their approved consequences. It cannot acquire host shell, foreign Incus admin, host file, parent BoxWeave or arbitrary network/Caddy authority.

**Multi-Agent continuity does not expand authority.** Verified `actor_id`/`token_id` is from server authentication; `work_ref`, `agent_ref`, clientInfo, MCP sessions and future Task handles are never credentials, ownership grants or a reason to read other controllers. For `operation_list`, restrict the permitted controller/layer/owned resources **before** filtering and pagination; preserve bounded Exec output limits. Use separate revocable credentials for trustworthy per-Agent attribution, and never log raw guest args/tokens/prompt chains. [Design](AGENT_OPERATION_CONTINUITY.md) and [TC security cases](../verification/CONTINUITY_ACCEPTANCE.md).

## Owned native boundary

- Confirm exact Incus instance UUID/incarnation and local SQLite ownership before sensitive calls. Names/IPs are not identities; reuse cannot attach old peer grants or forwards. A restricted Incus Project assists isolation but is not by itself ownership or private packet isolation proof.
- Only BoxWeave-owned Boxes, native forwards, single directional peer grants and relevant operation receipts may be exposed. Existing unrelated Incus instances/projects/networks remain unmanaged. SQLite loss/corruption fails closed and requires host-local independently verified recovery.
- Guest-root `box_exec` can read/write/delete guest files even if those file actions are not individually exposed. Guest root is not host root: deny privileged container configuration, host mounts, Unix socket bind, devices, raw Incus API pass-through or arbitrary host shell/file operations.
- Private Bridge NIC ACL rules rely on native address matching because Bridge ACL group selectors are unsupported; make current Box UUID -> NIC -> IPv4/IPv6 bindings and anti-spoof controls an explicit proof obligation. Revoke/withhold grants if addresses change without verified atomic policy convergence. Never claim a Box UUID is itself a native ACL selector.
- Private traffic between managed Boxes is denied by default with actual IPv4/enabled IPv6 enforcement; controlled IPv6 disable or Box creation/start refusal if enforcement is unsupported. Simple `box_peer_set` may explicitly allow/revoke one directional owned Box pair. Never edit arbitrary CIDR/IP/host firewall.
- Native TCP/UDP forwards accept only owned Box UUID + guest port. `box_delete` must also clean up and verify **only** that Box's managed dependent forwards and peer grants. A partial delete keeps the original incident evidence and refuses unsafe rollback or foreign cleanup. In the P0 instance-owned proxy-device backend, deleting an instance also removes its native device; verify no stale BoxWeave ledger claim or unrelated host listener. Orphan *network-forward objects* are only relevant if that separate backend is explicitly introduced later. Explicit host public bind is an administrative exposure event; default local bind where possible, reject foreign port/device claims and collisions.
- Independent nested installations receive no inherited socket, Token or parent service authority. A Box containing an inner BoxWeave is an ordinary parent-owned Box with standard lifecycle/destructive rules.

## Native SDK and Root authority

The confined nonroot incus-only service uses the per-user socket. Verify exact restricted Project, owned UUID, native capabilities, allowed proxy/nesting settings and actual packet isolation. UseProject alone is not authorization. Scoped Peer v2 uses independent ACLs, a durable original-bound Root guard/lifecycle path and exact native/kernel validation; Legacy uses a separately reviewed shared-ACL path. Neither exposes arbitrary Root operations. Historical TLS and daemon-global Scriptlet experiments are retained as [historical evidence](../history/README.md), not authorization to change current daemon-global settings. See [network](NETWORK.md), [Incus integration](INCUS_INTEGRATION.md) and [maintenance](../getting-started/MANUAL_INSTALL.md).

## Minimal access control

- Local bootstrap issues independently revocable authenticated management credentials. OAuth grants retain a stable Actor across refresh; manual credentials each identify their own Actor. There are **no** application permission Scopes, viewer/admin roles, per-Box Token grants or separate per-Tool rights.
- Each approved Actor may read, change, Exec inside and delete **only** this Layer's owned managed Boxes. Reads and guest-root Exec are sensitive; preserve Layer/Project/UUID isolation, credential expiration, revocation and original-operation visibility boundaries.
- Revocation rejects new requests and provably undispatched queued operations. It does not imply that a dispatched/possibly dispatched operation was stopped, rolled back or resolved. A revoked Token cannot fetch old receipts; a valid authenticated Actor within the same Layer can inspect redacted audit evidence.

## MCP and async safety

- Bind authenticated Streamable HTTP MCP to `127.0.0.1` by default. Inner BoxWeave's localhost MCP is not reachable by addressing its bridge IP from the parent; a guest-local, separately authenticated test client may run as ordinary software under the parent's authorized guest Exec. Operator-managed HTTPS proxy/Cloudflare Tunnel is separate; enforce Token, size/rate and Origin/trusted-proxy checks regardless.
- Every mutation requires caller-provided reusable idempotency key; same actor/key/intent returns original Tracking/Operation IDs and conflicting reuse is denied. Go context is propagated through native adapters, but disconnected caller context does not undo accepted durable work.
- All accepted mutations persist safe intent before effect, cap output and retain UNKNOWN on ambiguous native results. **UNKNOWN manual claim release is host-local only**, requires exact target/operation review, short-lived one-use challenge, second explicit risk acknowledgement and protected audit; it never changes/retries/cancels the original operation. See [approved procedure](../process/UNKNOWN_MANUAL_RECOVERY.md). Verify post-state; no blind replay of create/delete/Exec/forward changes. `operation_get` includes small redacted diagnostic summary. Detailed logs and recovery stay in local protected CLI; **no separate diagnostics MCP service**.
- Guest file list/read/create/replace/delete are restricted to guest regular files with size/race/path checks. No built-in patch engine or host fallback. Fail UNSUPPORTED when native safe overwrite/delete cannot be verified.
- Automatic dependency reconciliation is host-local and explicitly invoked. Reuse/manage an existing Incus daemon without unapproved shared service disruption, and never adopt arbitrary existing instances. No Caddy dependency or automatic host reboot.

## Negative acceptance

Test foreign Box UUID/host file, raw Incus settings, old IP/name reuse, unsafe guest file symlink/hardlink, unauthenticated/expired/revoked Token Exec/write, revoked queued/active operations, uncontrolled IPv4/IPv6 paths, foreign port listener, public bind authorization, nested parent/sibling privilege, duplicate idempotency and UNKNOWN. Refusals must have independent no-side-effect evidence.

See [scope](../product/SCOPE.md), [MCP tool contracts](MCP_TOOL_CONTRACTS.md), [network](NETWORK.md), [operations](OPERATIONS.md).
