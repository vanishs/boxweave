# Security model

## Identity and authorization

1. The client machine generates a **dedicated** SSH key and pins the Ubuntu server's ED25519 host key from a trusted server-side snapshot. The private key never leaves the client.
2. Ubuntu stores the transport public key with `restrict,command="/usr/local/bin/boxweave-mcp"` in `authorized_keys`. It cannot start an arbitrary host shell through that key, but it **can use all MCP operations permitted by the provider credential**.
3. A root-owned launcher starts Containarium's MCP backend as an ordinary Ubuntu user. Its JWT lives in a protected file under that user's BoxWeave configuration directory. A root-only systemd service regularly rotates that credential.
4. A separate root-managed service reads authentic LXC SSH server public keys through the trusted local Incus control plane, compares all three expected algorithms against network-presented keys and changes only eligible project-container IP entries in the Ubuntu user's `known_hosts`.

## Operational risks

- **Admin JWT:** the initial provider token carries Containarium administrator privileges. Restrict trusted agents and users accordingly, audit access, and prefer scoped roles or approvals when supported. A forced SSH command is not authorization for each MCP action.
- **Root installer:** `curl | sudo bash` trusts code delivered over the network. Inspect the scripts and use an immutable reviewed ref before deploying to a production host.
- **Existing host listeners:** the installer does not harden the Containarium daemon's existing external network listeners. Verify the provider's own gRPC and REST bindings and firewall separately.
- **SSH enrollment:** first-time enrollment uses an existing password or alternate SSH key. The setup script will not enable insecure host-key checking, nor will it invent a first-login method.
- **IP reuse:** new LXC instances may initially fail with `knownhosts: key mismatch` until the timer catches up. This is expected; do not bypass SSH host-key checking.
- **Coexisting installers:** stop the previous bridge's background timers before installing BoxWeave; see `MIGRATION.md`. Two independent writers of `known_hosts` can race.
- **Credential expiration:** the service timer is responsible for token refresh. Monitor its health.
- **Scope:** the initial reconciler only supports private IPv4 addresses on trusted Incus managed networks and Containarium LXC metadata. It intentionally declines unknown hosts.

For production deployments, add peer-reviewed signed releases, least-privilege provider credentials, automated rollback and network access controls before treating this as hardened infrastructure.
