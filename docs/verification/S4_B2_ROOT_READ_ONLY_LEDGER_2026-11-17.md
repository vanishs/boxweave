# S4-B2  protected SQLite read-only root audit CLI (2026-11-17)

**S4_B2_PARTIAL; B3 NOT STARTED.** A narrow host-root read-only prerequisite, not an installed controller handoff, ACL mutation, Scriptlet authorization, or Peer MCP Tool.

## Rationale
The ordinary controller state.Open can create an owner database and initializes SQLite WAL, so it is inappropriate as a host-root security-audit reader.

## Implementation
- state.OpenNetworkAuditReadOnly requires an existing canonical protected DB and private real parent. Missing, symlink, hardlink, unsafe file mode, foreign schema/index or corrupt database refuses. No directory/database creation. It uses SQLite mode=ro and query_only=ON, the same extracted exact-schema validator, and no writable WAL initialization.
- Ordinary state.Open retains its previous full schema validation and writer initialization.
- Host-local CLI native-peer-audit uses the read-only store with exact --db, --project, --network and --acl, requiring effective host root. It calls NativePeerBridgeAuditFromStore, independently binding the service UID-derived Project to native global Incus bridge/ACL/Box UUID/MAC/IP readback. It returns only a bounded digest, not credentials. No MCP root delegation, ACL/conntrack write, guest restart or peer authorization.
- Local tests cover missing DB/parent, permissions, symlink/hardlink, corrupt extra schema, denied SQL write and visibility of live committed WAL state.
- The disposable Incus 7.6 native root identity test now uses the read-only connection and requires a new non-SKIP workflow marker.

## Evidence and remaining blockers
Development occurred only in a new L0 one-use Box c4d27042-49f4-44f1-8b66-379dbe9c8fce, original create 3e99653ed4f5cf3e93e12103e4f3fa2e, native b2fc453d-0db2-404b-9740-bad80a751298, based on e2858014a03db48f1ef1a7eccd108becf81ce630. Shared L0 global permissions, ACLs and existing Boxes were unchanged.

This read-only snapshot is not a durable network lease or a complete root-to-controller operation transaction. Exact-HEAD fresh native execution, installed version, privilege handoff, per-NIC nft DROP, old-client parity and crash/ACK loss recovery still require independent verification. Never infer S4_B2_PASS or start B3 from this evidence.
