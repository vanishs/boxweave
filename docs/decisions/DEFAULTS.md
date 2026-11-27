# Approved v1 operating defaults

| Area | Default or restriction |
| --- | --- |
| Guest Exec | UID 0 inside exact guest; optional supported UID/GID, never Host root. |
| Image/resources | Ubuntu LTS default, typed configurable CPU/RAM/disk; no mandatory storage driver. |
| Lifecycle | Start by default, explicit start=false; scoped boot.autostart=true preserves explicit false. |
| Nesting | Opt-in, conditional on native capabilities, no inherited parent socket/credential. |
| Incus | Reuse compatible daemon; separately reviewed local bootstrap/maintenance, no shared reset or automatic reboot. |
| Network | Default deny private peer, internet enabled by default with per-Box deny; exact directional Peer. |
| Forward | Loopback default; explicit external acknowledgment and service/native policy. Fresh installed service defaults to all; existing policy unchanged. |
| MCP/auth | Loopback HTTP; operator-owned HTTPS/Tunnel; independently revocable management Actors, no admin/viewer roles or permission scopes. |
| Files | List/read/exclusive create; conditional replace/delete unsupported. |
| State | Missing/inconsistent ownership fails closed; no replay UNKNOWN or silent adoption. |
| Maintenance | Current #191 Scoped fresh-only, separate normal and reviewed offline retirement/resume, and reviewed completed-data purge. Legacy upgrades are historical; no current product upgrade/rollback. |
| Publication | Protected v1 and exact release branch; stable and deployment are separate gates. |

See [product boundary](PRODUCT_BOUNDARY.md), [operational policy](P0_OPERATIONAL_POLICIES.md), [network](../design/NETWORK.md), [versioning](../process/VERSIONING.md) and [historical decisions](../history/README.md).
