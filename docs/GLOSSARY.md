# Glossary — BoxWeave v1 terminology

Use this glossary in onboarding, design and acceptance documents. Older dated experiments may use superseded wording.

| Term | Precise meaning |
| --- | --- |
| **Human / operator** | Person who controls the host, Incus, approvals, credentials, networking and public exposure. |
| **Agent / MCP client** | ChatGPT or another coding client that calls typed MCP Tools; usually runs **outside** a Box. |
| **Box** | Exact native UUID-registered, **BoxWeave-owned Linux Incus container**; never synonymous with any host container or VM. |
| **Host (L0), nested layers (L1/L2)** | Linux host of one controller; a nesting-capable Box may independently host another Incus+BoxWeave controller. No cross-layer takeover. |
| **Controller** | Single Go runtime, local CLI, policy and durable SQLite state for one confined Incus Project. |
| **Incus** | Operator-managed daemon/API implementing guest lifecycle, native network/storage and Exec. |
| **Project** | Incus-scoped namespace; the service user is confined to `user-<UID>` through the non-admin Incus socket, not simply a query selector. |
| **Profile** | Incus instance default configuration; the restricted Project's `default` Profile contains reviewed root disk and NIC. |
| **Pool / storage driver** | Incus storage pool and implementation (`btrfs`, `dir`, `zfs`, `lvm` ...). A mount path does not determine storage driver/quota capability. |
| **Bridge, NIC, ACL** | Native managed private network, attached Box virtual NIC and access-control rules; actual packet isolation needs separate testing. |
| **MCP / tool** | Model Context Protocol, the bounded Agent-to-BoxWeave interface. Not a host-shell or generic REST admin API. |
| **`/mcp`** | Single authenticated Streamable HTTP endpoint; local default `127.0.0.1:18650/mcp`, optionally tunneled with external HTTPS. |
| **Bearer / admin / viewer** | Revocable token and existing two roles. Viewer is read-only, may read sensitive guest data; admin may mutate owned guests. |
| **Embedded OAuth** | Optional browser approval/credential-issuance path in same Go service, not an identity platform or separate MCP endpoint. |
| **Public origin** | Exact external `https://hostname` configured locally for OAuth, **without** `/mcp`, query, or trailing path; full client MCP URL ends in `/mcp`. |
| **Tunnel** | Optional operator-run HTTPS exposure such as Cloudflare Quick Tunnel; not part of BoxWeave installer or daemon management. |
| **Tracking ID** | Generated request correlation identifier, not the durable mutation identity. |
| **Idempotency key** | Client-reusable mutation key; changing it after a timeout may duplicate native effects. |
| **Operation ID / journal** | Original durable asynchronous write receipt, native evidence, status and bounded output; queried after reconnect. |
| **UNKNOWN** | Native mutation's final state cannot yet be proved. Preserve its original identity; do not auto-retry. |
| **`service_ready`, `mcp_ready`** | Restricted actor/Incus API and MCP/SQLite **service installation** readiness; not proof of Box create ability. |
| **`native.ready`, `doctor ready`** | Stronger native operation admission prerequisites; not Guest image, packet enforcement or quota proof. |
| **`native-prepare`** | Separately reviewed root-local Project/Profile/exclusive-ACL preparation on **empty owned** Project, with digest and journal. |
| **`native-prereq-verify`** | Matching-digest review of eight real Incus configuration checks; separate packet witness required. |
| **RC / stable / S5** | Release Candidate has published immutable build but **not** stable S5 production acceptance. Stable pointer can remain `UNPUBLISHED`. |
| **HOST_POSTFLIGHT / HOST_GO** | Broader native-host verification gates with real operations, fault/negative tests and independent review, not just `doctor` or OAuth reads. |

Related: [manual install](getting-started/MANUAL_INSTALL.md) · [architecture](design/ARCHITECTURE.md) · [Operations](design/OPERATIONS.md) · [product boundary](decisions/PRODUCT_BOUNDARY.md).
