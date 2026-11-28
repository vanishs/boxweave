# Incus integration

Official typed Incus Go SDK APIs supply native instances, Exec, operations, files, proxy devices, ACLs and networking. BoxWeave provides ownership and durable domain operations, not a second Incus implementation or generic API passthrough.

## Authority

Nonroot service user in incus, not incus-admin, uses /var/lib/incus/unix.socket.user. Require the restricted per-user Project, reviewed profile/bridge/pool, native extensions and exact owned UUID/incarnation. UseProject selects a Project, not authority. Native API authorization and packet isolation are distinct. Root/general Host maintenance is not exposed over MCP.

Historical Legacy installations used a separately reviewed shared-ACL authority path; it is not a current #191 runtime. The current Scoped candidate installs bounded guard/lifecycle helpers, independently verifies original claims, native identity/ETag and kernel policy. [Network](NETWORK.md) and [security](SECURITY.md) are authoritative. Historical restricted TLS/global Scriptlet tests are not authorization for current daemon-global changes.

## Dependencies and evidence

Reuse compatible initialized Incus. Exact reviewed native bootstrap/migration helpers exist under their own restrictions; no shared-daemon upgrade license. See [dependency lifecycle](../process/DEPENDENCY_LIFECYCLE.md) and [installation](../getting-started/MANUAL_INSTALL.md). Incus 7.6 is tested; check extensions and actual behavior, not a guessed compatibility range. Ubuntu LTS default guest; image availability, download, quota, storage driver and nested kernel capability remain separate from controller readiness.

Verify native poststate and original UUID/ETag; a missing native Operation or transport error does not prove no effect. Separate actual guest exit from native terminal status. [go.mod](../../go.mod) pins the dependencies. [Acceptance](../verification/ACCEPTANCE.md) and [history](../history/README.md) retain original native authz, storage and network evidence.
