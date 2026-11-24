# RC.3: separate installation from Guest creation; delegate images to Incus

## Scope and owner intent
BoxWeave is a narrow Incus management layer, not a second host-environment policy engine. Installing and starting BoxWeave must not require Ubuntu 24.04 Guest images, a specific storage driver, or a prepared first Box. RC.2 unnecessarily coupled installer activation to full Box readiness.

## Runtime boundary
- `native.service_ready`: dedicated nonroot actor, Incus-exclusive restricted Project, supported Incus daemon version (7.5+ on the currently tested major), and required native API extensions. `mcp_ready` additionally confirms the SQLite schema. The installer uses these only, plus its own exact binary/unit/startup probe.
- `native.ready`: separate, stronger admission proof for Create, Exec and other native writes. It still requires an operator-approved root disk, safe NIC and ACL, and a usable selected Incus storage pool. A missing Guest image is not a global readiness failure.
- The old `doctor` command remains a full native-operation admission check; installation uses read-only `status` and `native.service_ready`.
- Auto setup remains **bounded**: Incus creates the per-user Project/bridge from its own multi-user defaults; a separate root-local `native-prepare-plan/apply`, with a matching SHA and explicit approval, tightens the owned Project/Profile and creates the exclusive ACL. No new privileged public MCP endpoint, no touching other projects, and no unapproved native mutations during installation.

## Guest image policy
- `box_create.image` is optional. A requested `local:<alias>` (or bare alias) uses only the actor's pre-existing local Incus image. `images:<catalog alias>` requests an image from Incus's official HTTPS simplestreams catalog; no untrusted caller-supplied URL or remote host is accepted.
- Without an image argument: prefer the locally approved `boxweave-default`, then the legacy `boxweave-ubuntu-24.04` for existing deployments; if neither exists, Incus pulls `ubuntu/24.04` as part of the original durable Create operation, so it does not create an untracked pre-Create side effect.
- Local image architecture and container type are validated by Incus; non-container images are refused in advance. The intent includes the requested image so changing image under an idempotency key cannot silently alter the operation.
- If the remote catalog cannot be reached or the requested image is unavailable, the **original** Box Create operation fails or remains uncertain according to existing receipts. It is never automatically retried or masked as a successful Box. The administrator is responsible for public HTTPS egress, storage capacity and supply-chain trust.

## Acceptance boundaries
- RC.3 may be published once the exact-source Go/race/quality, isolated S2, nested S2.5 and dual-arch release workflow pass. Native tests with local images must remain compatible; RC.3-specific service-before-image and on-demand remote Guest behavior require dedicated physical-host verification.
- Installation does not confer HOST_GO or S5 signoff. Do not remove restricted nonroot Project, ACL, private-network or native UNKNOWN protections.
