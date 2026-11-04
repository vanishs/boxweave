# Upstream Incus and Containarium ownership (R-09)

The new app uses [Provider adapter](../bwmini/provider.py), [access adapter](../bwmini/access_provider.py), and [MCP transport](../bwmini/mcp.py), not old host-managed wrapper services. Official reference information: [Containarium](https://github.com/footprintai/Containarium) and [Incus](https://linuxcontainers.org/incus/docs/main/installing/). Upstream installation/version facts are external and can change; this document is **not** a fresh compatibility certification.

| Layer | Responsible owner | v2 behavior and required proof |
| --- | --- | --- |
| Incus daemon/client, storage, host nesting and policy | Host admin | New `install.sh` refuses implicit existing-host changes; verify exact approved release, user/daemon and confinement separately |
| Containarium daemon/CLI and upstream release | Provider admin | A root-authorized, explicitly selected BoxWeave v0.44.0+ streamed install provisions an absent Provider or upgrades verified below-baseline components; see the version upgrade SOP; no downgrade or signer/storage adoption |
| Provider JWT/token rotation, host key and SSH trust | Operator/security owner | The app does not mint, read, export, refresh or replace credentials/trust automatically |
| Independent application MCP launcher | App/operator | Must be explicitly configured under a private file and independently authorized; actual initialize, tools discovery and `list_containers` read required |
| Tenant native contract, Guard, network policy, port route | Provider/Guard owner | Unsupported native capabilities block the write. A route API response cannot prove real packet filtering |
| App DB/admin, versioned archive, API/worker user units | App distribution | Non-root immutable release, private state, backup/repair marker, loopback bind; does not provide host TLS/firewall or boot proof |

Archive `patches/containarium/v0.114.2-tenant-network.patch` and associated scripts are **historical only**. The old patch README reported upstream v0.120.1 native `native-network-tenant/v1`; this is archival evidence, not a guarantee for an unknown installed Provider. Public upstream docs now distinguish `containarium` client from `containariumd` daemon and describe changing Ubuntu/Incus prerequisites. Do not apply old patches or trust unchecked floating root install scripts. An explicit administrator-selected published `--upstream-latest` resolves a versioned upstream release with digest validation; do not bypass preflight just because a feature is missing.

A future Agent must capture the approved upstream source/package digest/signature and version, Incus and Containarium actual versions, launcher owner and isolated credential handling, negotiated MCP protocol/tool/catalog, Tenant native/Guard capability and failure downgrade. Real packet/TLS/reboot/confinement require separately scoped tests. See [R-02/09 validation](verification/REMEDIATION_TEST_QUEUE.md).
