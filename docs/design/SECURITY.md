# Security and ownership — small fixed policy

**Invariant:** A BoxWeave MCP caller, including an `admin` token, can inspect or mutate **only** this layer's BoxWeave-owned Incus resources and their approved consequences. It cannot acquire host shell, foreign Incus admin, host file, parent BoxWeave or arbitrary network/Caddy authority.

## Owned native boundary

- Confirm exact Incus instance UUID/incarnation and local SQLite ownership before sensitive calls. Names/IPs are not identities; reuse cannot attach old peer grants or forwards. A restricted Incus Project assists isolation but is not by itself ownership or private packet isolation proof.
- Only BoxWeave-owned Boxes, native forwards, single directional peer grants and relevant operation receipts may be exposed. Existing unrelated Incus instances/projects/networks remain unmanaged. SQLite loss/corruption fails closed and requires host-local independently verified recovery.
- Guest-root `box_exec` can read/write/delete guest files even if those file actions are not individually exposed. Guest root is not host root: deny privileged container configuration, host mounts, Unix socket bind, devices, raw Incus API pass-through or arbitrary host shell/file operations.
- Private traffic between managed Boxes is denied by default with actual IPv4/enabled IPv6 enforcement; controlled IPv6 disable or Box creation/start refusal if enforcement is unsupported. Simple `box_peer_set` may explicitly allow/revoke one directional owned Box pair. Never edit arbitrary CIDR/IP/host firewall.
- Native TCP/UDP forwards accept only owned Box UUID + guest port. `box_delete` must also clean up and verify **only** that Box's managed dependent forwards and peer grants. A partial delete keeps the original incident evidence and refuses unsafe rollback or foreign cleanup. Explicit host public bind is an administrative exposure event; default local bind where possible, reject foreign port/device claims and collisions.
- Independent nested installations receive no inherited socket, Token or parent service authority. A Box containing an inner BoxWeave is an ordinary parent-owned Box with standard lifecycle/destructive rules.

## Minimal access control

- Local bootstrap creates super-admin; multiple independently revocable MCP credentials are `admin` or `viewer`. No per-Box Token grants, separate file/delete/cancel/tool rights or general RBAC platform. Admin can issue authorized mutations and guest-root Exec only within this controller's ownership domain.
- Viewer **cannot mutate or Exec**, but can read owned Box metadata, files and bounded Exec output, which can contain secrets. Never market viewer as a privacy/redaction role; revoke/limit issuance as appropriate.
- Revocation rejects new requests and provably undispatched queued operations. It does not imply that a dispatched/possibly dispatched operation was stopped, rolled back or resolved. A revoked Token cannot fetch old receipts; a valid admin can inspect redacted audit evidence.

## MCP and async safety

- Bind authenticated Streamable HTTP MCP to `127.0.0.1` by default. Inner BoxWeave's localhost MCP is not reachable by addressing its bridge IP from the parent; a guest-local, separately authenticated test client may run as ordinary software under the parent's authorized guest Exec. Operator-managed HTTPS proxy/Cloudflare Tunnel is separate; enforce Token, size/rate and Origin/trusted-proxy checks regardless.
- Every mutation requires caller-provided reusable idempotency key; same actor/key/intent returns original Tracking/Operation IDs and conflicting reuse is denied. Go context is propagated through native adapters, but disconnected caller context does not undo accepted durable work.
- All accepted mutations persist safe intent before effect, cap output and retain UNKNOWN on ambiguous native results. Verify post-state; no blind replay of create/delete/Exec/forward changes. `operation_get` includes small redacted diagnostic summary. Detailed logs and recovery stay in local protected CLI; **no separate diagnostics MCP service**.
- Guest file list/read/create/replace/delete are restricted to guest regular files with size/race/path checks. No built-in patch engine or host fallback. Fail UNSUPPORTED when native safe overwrite/delete cannot be verified.
- Automatic dependency reconciliation is host-local and explicitly invoked. Reuse/manage an existing Incus daemon without unapproved shared service disruption, and never adopt arbitrary existing instances. No Caddy dependency or automatic host reboot.

## Negative acceptance

Test foreign Box UUID/host file, raw Incus settings, old IP/name reuse, unsafe guest file symlink/hardlink, token viewer Exec/write, revoked queued/active operations, uncontrolled IPv4/IPv6 paths, foreign port listener, public bind authorization, nested parent/sibling privilege, duplicate idempotency and UNKNOWN. Refusals must have independent no-side-effect evidence.

See [scope](../product/SCOPE.md), [MCP tool contracts](MCP_TOOL_CONTRACTS.md), [network](NETWORK.md), [operations](OPERATIONS.md).
